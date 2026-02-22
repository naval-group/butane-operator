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

package controller

import (
	"context"
	"fmt"

	"github.com/coreos/butane/config"
	"github.com/coreos/butane/config/common"
	"github.com/go-logr/logr"
	butanev1alpha1 "github.com/naval-group/butane-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ButaneConfigReconciler reconciles a ButaneConfig object
type ButaneConfigReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder events.EventRecorder
}

//+kubebuilder:rbac:groups=butane.operators.naval-group.com,resources=butaneconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=butane.operators.naval-group.com,resources=butaneconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=butane.operators.naval-group.com,resources=butaneconfigs/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=events.k8s.io,resources=events,verbs=create;patch;update
//+kubebuilder:rbac:groups=admissionregistration.k8s.io,resources=validatingwebhookconfigurations,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=admissionregistration.k8s.io,resources=mutatingwebhookconfigurations,verbs=get;list;watch;update;patch

func (r *ButaneConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("butaneconfig", req.NamespacedName)

	// Fetch the ButaneConfig instance
	var butaneConfig butanev1alpha1.ButaneConfig
	if err := r.Get(ctx, req.NamespacedName, &butaneConfig); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	r.Recorder.Eventf(&butaneConfig, nil, corev1.EventTypeNormal, "ConfigRetrieved", "ConfigRetrieved", "Successfully retrieved ButaneConfig")

	// Extract the raw Butane configuration from the runtime.RawExtension
	rawConfig := butaneConfig.Spec.Config.Raw
	if rawConfig == nil {
		log.Error(nil, "ButaneConfig is missing a Config")
		return ctrl.Result{}, fmt.Errorf("missing Config in ButaneConfig %s", butaneConfig.Name)
	}

	// Convert the ButaneConfig to an Ignition config
	ignitionConfig, rpt, err := config.TranslateBytes(rawConfig, common.TranslateBytesOptions{})
	if err != nil || len(rpt.Entries) > 0 {
		log.Error(err, "Error translating ButaneConfig to Ignition config")
		r.Recorder.Eventf(&butaneConfig, nil, corev1.EventTypeWarning, "ConversionFailed", "ConversionFailed", "Failed to convert ButaneConfig to Ignition config: %s", rpt.String())
		return ctrl.Result{}, err
	}

	// Create or update the Secret containing the Ignition configuration
	secretName := fmt.Sprintf("%s-ignition", butaneConfig.Name)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: butaneConfig.Namespace,
		},
		Data: map[string][]byte{
			"userdata": ignitionConfig,
		},
	}

	// Set the owner reference to the ButaneConfig instance
	if err := controllerutil.SetControllerReference(&butaneConfig, secret, r.Scheme); err != nil {
		r.Recorder.Eventf(&butaneConfig, nil, corev1.EventTypeWarning, "SetOwnerReferenceFailed", "SetOwnerReferenceFailed", "Failed to set owner reference for the Secret")
		return ctrl.Result{}, err
	}

	// Create or update the Secret in the cluster
	if err := r.Create(ctx, secret); err != nil {
		if apierrors.IsAlreadyExists(err) {
			if err := r.Update(ctx, secret); err != nil {
				r.Recorder.Eventf(&butaneConfig, nil, corev1.EventTypeWarning, "SecretUpdateFailed", "SecretUpdateFailed", "Failed to update the Secret")
				return ctrl.Result{}, err
			}
		} else {
			r.Recorder.Eventf(&butaneConfig, nil, corev1.EventTypeWarning, "SecretCreateFailed", "SecretCreateFailed", "Failed to create the Secret")
			return ctrl.Result{}, err
		}
	}

	// Update the status of ButaneConfig
	butaneConfig.Status.SecretName = secretName
	if err := r.Status().Update(ctx, &butaneConfig); err != nil {
		r.Recorder.Eventf(&butaneConfig, nil, corev1.EventTypeWarning, "StatusUpdateFailed", "StatusUpdateFailed", "Failed to update ButaneConfig status")
		return ctrl.Result{}, err
	}

	log.Info("Successfully processed ButaneConfig", "secretName", secretName)
	r.Recorder.Eventf(&butaneConfig, nil, corev1.EventTypeNormal, "ReconciliationSucceeded", "ReconciliationSucceeded", "Successfully reconciled ButaneConfig")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ButaneConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Recorder = mgr.GetEventRecorder("butaneconfig-controller")
	return ctrl.NewControllerManagedBy(mgr).
		For(&butanev1alpha1.ButaneConfig{}).
		Complete(r)
}
