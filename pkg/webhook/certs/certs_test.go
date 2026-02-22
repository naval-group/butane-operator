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
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-logr/logr"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestGenerateCA(t *testing.T) {
	cert, key, certPEM, err := generateCA(365 * 24 * time.Hour)
	if err != nil {
		t.Fatalf("generateCA() error = %v", err)
	}
	if cert == nil || key == nil || len(certPEM) == 0 {
		t.Fatal("generateCA() returned nil values")
	}
	if !cert.IsCA {
		t.Error("expected CA certificate")
	}
	if cert.Subject.CommonName != "webhook-ca" {
		t.Errorf("expected CN=webhook-ca, got %s", cert.Subject.CommonName)
	}
	if cert.KeyUsage&x509.KeyUsageCertSign == 0 {
		t.Error("expected KeyUsageCertSign")
	}
}

func TestGenerateServerCert(t *testing.T) {
	caCert, caKey, _, err := generateCA(365 * 24 * time.Hour)
	if err != nil {
		t.Fatalf("generateCA() error = %v", err)
	}

	dnsNames := []string{"svc.ns.svc", "svc.ns.svc.cluster.local"}
	certPEM, keyPEM, err := generateServerCert(caCert, caKey, dnsNames, 365*24*time.Hour)
	if err != nil {
		t.Fatalf("generateServerCert() error = %v", err)
	}
	if len(certPEM) == 0 || len(keyPEM) == 0 {
		t.Fatal("generateServerCert() returned empty PEM")
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		t.Fatal("failed to decode server cert PEM")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("failed to parse server cert: %v", err)
	}
	if cert.IsCA {
		t.Error("server cert should not be a CA")
	}
	if len(cert.DNSNames) != 2 {
		t.Errorf("expected 2 DNS names, got %d", len(cert.DNSNames))
	}

	// Verify the cert is signed by the CA
	roots := x509.NewCertPool()
	roots.AddCert(caCert)
	if _, err := cert.Verify(x509.VerifyOptions{
		Roots:     roots,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}); err != nil {
		t.Errorf("server cert verification failed: %v", err)
	}
}

func TestNeedsRenewal(t *testing.T) {
	threshold := 30 * 24 * time.Hour

	// Generate a cert valid for 1 year — should NOT need renewal
	caCert, caKey, _, _ := generateCA(365 * 24 * time.Hour)
	certPEM, _, _ := generateServerCert(caCert, caKey, []string{"test.svc"}, 365*24*time.Hour)

	secret := &corev1.Secret{
		Data: map[string][]byte{"tls.crt": certPEM},
	}
	if needsRenewal(secret, threshold) {
		t.Error("cert valid for 1 year should not need renewal")
	}

	// Generate a cert valid for 1 day — should need renewal
	certPEM, _, _ = generateServerCert(caCert, caKey, []string{"test.svc"}, 24*time.Hour)
	secret.Data["tls.crt"] = certPEM
	if !needsRenewal(secret, threshold) {
		t.Error("cert valid for 1 day should need renewal with 30-day threshold")
	}

	// Empty secret — should need renewal
	secret.Data["tls.crt"] = nil
	if !needsRenewal(secret, threshold) {
		t.Error("empty cert should need renewal")
	}

	// Invalid PEM — should need renewal
	secret.Data["tls.crt"] = []byte("not a pem")
	if !needsRenewal(secret, threshold) {
		t.Error("invalid PEM should need renewal")
	}
}

func TestWriteCertsToDisk(t *testing.T) {
	dir := t.TempDir()
	certDir := filepath.Join(dir, "certs")

	certData := []byte("cert-data")
	keyData := []byte("key-data")

	if err := writeCertsToDisk(certDir, certData, keyData); err != nil {
		t.Fatalf("writeCertsToDisk() error = %v", err)
	}

	got, err := os.ReadFile(filepath.Join(certDir, "tls.crt"))
	if err != nil {
		t.Fatalf("reading tls.crt: %v", err)
	}
	if string(got) != "cert-data" {
		t.Errorf("tls.crt = %q, want %q", got, "cert-data")
	}

	got, err = os.ReadFile(filepath.Join(certDir, "tls.key"))
	if err != nil {
		t.Fatalf("reading tls.key: %v", err)
	}
	if string(got) != "key-data" {
		t.Errorf("tls.key = %q, want %q", got, "key-data")
	}
}

func TestDNSNamesForService(t *testing.T) {
	names := dnsNamesForService("webhook-service", "my-ns")
	if len(names) != 2 {
		t.Fatalf("expected 2 DNS names, got %d", len(names))
	}
	if names[0] != "webhook-service.my-ns.svc" {
		t.Errorf("names[0] = %q", names[0])
	}
	if names[1] != "webhook-service.my-ns.svc.cluster.local" {
		t.Errorf("names[1] = %q", names[1])
	}
}

func TestMutatingWebhookName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"butane-operator-validating-webhook-configuration", "butane-operator-mutating-webhook-configuration"},
		{"no-match-here", "no-match-here"},
	}
	for _, tt := range tests {
		got := mutatingWebhookName(tt.input)
		if got != tt.want {
			t.Errorf("mutatingWebhookName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestEnsure_CreatesNewCerts(t *testing.T) {
	ctx := context.Background()
	client := fake.NewClientset()

	// Create a ValidatingWebhookConfiguration for patching
	sideEffects := admissionregistrationv1.SideEffectClassNone
	vwc := &admissionregistrationv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{Name: "test-validating-webhook-configuration"},
		Webhooks: []admissionregistrationv1.ValidatingWebhook{
			{
				Name:                    "test.webhook.io",
				SideEffects:             &sideEffects,
				AdmissionReviewVersions: []string{"v1"},
				ClientConfig:            admissionregistrationv1.WebhookClientConfig{},
			},
		},
	}
	if _, err := client.AdmissionregistrationV1().ValidatingWebhookConfigurations().Create(ctx, vwc, metav1.CreateOptions{}); err != nil {
		t.Fatalf("creating VWC: %v", err)
	}

	certDir := filepath.Join(t.TempDir(), "certs")
	cfg := Config{
		ServiceName:       "webhook-service",
		Namespace:         "test-ns",
		SecretName:        "webhook-server-cert",
		WebhookConfigName: "test-validating-webhook-configuration",
		CertDir:           certDir,
		CertValidity:      365 * 24 * time.Hour,
		RenewalThreshold:  30 * 24 * time.Hour,
	}

	if err := Ensure(ctx, client, cfg, logr.Discard()); err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}

	// Verify secret was created
	secret, err := client.CoreV1().Secrets("test-ns").Get(ctx, "webhook-server-cert", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("secret not created: %v", err)
	}
	if len(secret.Data["ca.crt"]) == 0 {
		t.Error("secret missing ca.crt")
	}
	if len(secret.Data["tls.crt"]) == 0 {
		t.Error("secret missing tls.crt")
	}
	if len(secret.Data["tls.key"]) == 0 {
		t.Error("secret missing tls.key")
	}

	// Verify certs on disk
	if _, err := os.Stat(filepath.Join(certDir, "tls.crt")); err != nil {
		t.Error("tls.crt not written to disk")
	}
	if _, err := os.Stat(filepath.Join(certDir, "tls.key")); err != nil {
		t.Error("tls.key not written to disk")
	}

	// Verify VWC was patched
	updatedVWC, err := client.AdmissionregistrationV1().ValidatingWebhookConfigurations().Get(ctx, "test-validating-webhook-configuration", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("getting VWC: %v", err)
	}
	if len(updatedVWC.Webhooks[0].ClientConfig.CABundle) == 0 {
		t.Error("VWC caBundle not patched")
	}
}

func TestEnsure_SkipsValidCerts(t *testing.T) {
	ctx := context.Background()

	// Pre-generate valid certs
	caCert, caKey, caPEM, _ := generateCA(365 * 24 * time.Hour)
	certPEM, keyPEM, _ := generateServerCert(caCert, caKey, []string{"svc.ns.svc"}, 365*24*time.Hour)

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "webhook-server-cert",
			Namespace: "test-ns",
		},
		Type: corev1.SecretTypeTLS,
		Data: map[string][]byte{
			"ca.crt":  caPEM,
			"tls.crt": certPEM,
			"tls.key": keyPEM,
		},
	}

	sideEffects := admissionregistrationv1.SideEffectClassNone
	vwc := &admissionregistrationv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{Name: "test-vwc"},
		Webhooks: []admissionregistrationv1.ValidatingWebhook{
			{
				Name:                    "test.webhook.io",
				SideEffects:             &sideEffects,
				AdmissionReviewVersions: []string{"v1"},
				ClientConfig:            admissionregistrationv1.WebhookClientConfig{},
			},
		},
	}

	client := fake.NewClientset(secret, vwc)

	certDir := filepath.Join(t.TempDir(), "certs")
	cfg := Config{
		ServiceName:       "webhook-service",
		Namespace:         "test-ns",
		SecretName:        "webhook-server-cert",
		WebhookConfigName: "test-vwc",
		CertDir:           certDir,
		CertValidity:      365 * 24 * time.Hour,
		RenewalThreshold:  30 * 24 * time.Hour,
	}

	if err := Ensure(ctx, client, cfg, logr.Discard()); err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}

	// Certs should have been written to disk (from existing secret)
	if _, err := os.Stat(filepath.Join(certDir, "tls.crt")); err != nil {
		t.Error("tls.crt not written to disk")
	}
}
