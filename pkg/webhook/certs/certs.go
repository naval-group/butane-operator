/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package certs

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Config holds the configuration for webhook certificate provisioning.
type Config struct {
	ServiceName       string
	Namespace         string
	SecretName        string
	WebhookConfigName string
	CertDir           string
	CertValidity      time.Duration
	RenewalThreshold  time.Duration
}

// Ensure provisions TLS certificates for the webhook server. It checks for an
// existing Secret, regenerates certs if missing or near expiry, writes them to
// disk, and patches the ValidatingWebhookConfiguration caBundle.
func Ensure(ctx context.Context, client kubernetes.Interface, cfg Config, log logr.Logger) error {
	dnsNames := dnsNamesForService(cfg.ServiceName, cfg.Namespace)

	// Check for existing secret
	existing, err := client.CoreV1().Secrets(cfg.Namespace).Get(ctx, cfg.SecretName, metav1.GetOptions{})
	if err == nil {
		if !needsRenewal(existing, cfg.RenewalThreshold) {
			log.Info("existing webhook certs are still valid, skipping regeneration")
			if err := writeCertsToDisk(cfg.CertDir, existing.Data["tls.crt"], existing.Data["tls.key"]); err != nil {
				return fmt.Errorf("writing existing certs to disk: %w", err)
			}
			return patchWebhookConfig(ctx, client, cfg.WebhookConfigName, existing.Data["ca.crt"], log)
		}
		log.Info("existing webhook certs need renewal")
	} else if !apierrors.IsNotFound(err) {
		return fmt.Errorf("getting secret %s/%s: %w", cfg.Namespace, cfg.SecretName, err)
	} else {
		log.Info("webhook cert secret not found, generating new certs")
	}

	// Generate CA
	caCert, caKey, caPEM, err := generateCA(cfg.CertValidity)
	if err != nil {
		return fmt.Errorf("generating CA: %w", err)
	}

	// Generate server cert signed by CA
	serverCertPEM, serverKeyPEM, err := generateServerCert(caCert, caKey, dnsNames, cfg.CertValidity)
	if err != nil {
		return fmt.Errorf("generating server cert: %w", err)
	}

	// Create or update the secret
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cfg.SecretName,
			Namespace: cfg.Namespace,
		},
		Type: corev1.SecretTypeTLS,
		Data: map[string][]byte{
			"ca.crt":  caPEM,
			"tls.crt": serverCertPEM,
			"tls.key": serverKeyPEM,
		},
	}

	if existing != nil && existing.Name != "" {
		secret.ResourceVersion = existing.ResourceVersion
		if _, err := client.CoreV1().Secrets(cfg.Namespace).Update(ctx, secret, metav1.UpdateOptions{}); err != nil {
			return fmt.Errorf("updating secret: %w", err)
		}
		log.Info("updated webhook cert secret")
	} else {
		if _, err := client.CoreV1().Secrets(cfg.Namespace).Create(ctx, secret, metav1.CreateOptions{}); err != nil {
			return fmt.Errorf("creating secret: %w", err)
		}
		log.Info("created webhook cert secret")
	}

	// Write certs to disk for the webhook server
	if err := writeCertsToDisk(cfg.CertDir, serverCertPEM, serverKeyPEM); err != nil {
		return fmt.Errorf("writing certs to disk: %w", err)
	}
	log.Info("wrote webhook certs to disk", "dir", cfg.CertDir)

	// Patch the ValidatingWebhookConfiguration with the CA bundle
	return patchWebhookConfig(ctx, client, cfg.WebhookConfigName, caPEM, log)
}

func dnsNamesForService(serviceName, namespace string) []string {
	return []string{
		fmt.Sprintf("%s.%s.svc", serviceName, namespace),
		fmt.Sprintf("%s.%s.svc.cluster.local", serviceName, namespace),
	}
}

func generateCA(validity time.Duration) (*x509.Certificate, *rsa.PrivateKey, []byte, error) {
	key, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("generating CA key: %w", err)
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("generating serial number: %w", err)
	}

	now := time.Now()
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   "webhook-ca",
			Organization: []string{"butane-operator"},
		},
		NotBefore:             now,
		NotAfter:              now.Add(validity),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("creating CA certificate: %w", err)
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("parsing CA certificate: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	return cert, key, certPEM, nil
}

func generateServerCert(caCert *x509.Certificate, caKey *rsa.PrivateKey, dnsNames []string, validity time.Duration) (certPEM, keyPEM []byte, err error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, fmt.Errorf("generating server key: %w", err)
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, nil, fmt.Errorf("generating serial number: %w", err)
	}

	now := time.Now()
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   dnsNames[0],
			Organization: []string{"butane-operator"},
		},
		DNSNames:  dnsNames,
		NotBefore: now,
		NotAfter:  now.Add(validity),
		KeyUsage:  x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
		},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, caCert, &key.PublicKey, caKey)
	if err != nil {
		return nil, nil, fmt.Errorf("creating server certificate: %w", err)
	}

	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	return certPEM, keyPEM, nil
}

func needsRenewal(secret *corev1.Secret, threshold time.Duration) bool {
	certPEM, ok := secret.Data["tls.crt"]
	if !ok || len(certPEM) == 0 {
		return true
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		return true
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return true
	}

	remaining := time.Until(cert.NotAfter)
	return remaining < threshold
}

func writeCertsToDisk(certDir string, certPEM, keyPEM []byte) error {
	if err := os.MkdirAll(certDir, 0750); err != nil {
		return fmt.Errorf("creating cert dir %s: %w", certDir, err)
	}
	if err := os.WriteFile(filepath.Join(certDir, "tls.crt"), certPEM, 0640); err != nil {
		return fmt.Errorf("writing tls.crt: %w", err)
	}
	if err := os.WriteFile(filepath.Join(certDir, "tls.key"), keyPEM, 0640); err != nil {
		return fmt.Errorf("writing tls.key: %w", err)
	}
	return nil
}

func patchWebhookConfig(ctx context.Context, client kubernetes.Interface, name string, caBundle []byte, log logr.Logger) error {
	vwc, err := client.AdmissionregistrationV1().ValidatingWebhookConfigurations().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("getting ValidatingWebhookConfiguration %s: %w", name, err)
	}

	updated := false
	for i := range vwc.Webhooks {
		vwc.Webhooks[i].ClientConfig.CABundle = caBundle
		updated = true
	}

	if updated {
		if _, err := client.AdmissionregistrationV1().ValidatingWebhookConfigurations().Update(ctx, vwc, metav1.UpdateOptions{}); err != nil {
			return fmt.Errorf("updating ValidatingWebhookConfiguration: %w", err)
		}
		log.Info("patched ValidatingWebhookConfiguration with CA bundle", "name", name)
	}

	// Also patch MutatingWebhookConfiguration if it exists
	mwcName := mutatingWebhookName(name)
	mwc, err := client.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(ctx, mwcName, metav1.GetOptions{})
	if err == nil {
		for i := range mwc.Webhooks {
			mwc.Webhooks[i].ClientConfig.CABundle = caBundle
		}
		if _, err := client.AdmissionregistrationV1().MutatingWebhookConfigurations().Update(ctx, mwc, metav1.UpdateOptions{}); err != nil {
			return fmt.Errorf("updating MutatingWebhookConfiguration: %w", err)
		}
		log.Info("patched MutatingWebhookConfiguration with CA bundle", "name", mwcName)
	} else if !apierrors.IsNotFound(err) {
		return fmt.Errorf("getting MutatingWebhookConfiguration %s: %w", mwcName, err)
	}

	return nil
}

// mutatingWebhookName derives the MutatingWebhookConfiguration name from the
// ValidatingWebhookConfiguration name by replacing "validating" with "mutating".
func mutatingWebhookName(validatingName string) string {
	// Convention: butane-operator-validating-webhook-configuration ->
	//             butane-operator-mutating-webhook-configuration
	const (
		old = "validating"
		new = "mutating"
	)
	for i := 0; i <= len(validatingName)-len(old); i++ {
		if validatingName[i:i+len(old)] == old {
			return validatingName[:i] + new + validatingName[i+len(old):]
		}
	}
	return validatingName
}
