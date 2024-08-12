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

package controllers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/coreos/butane/config"
	"github.com/coreos/butane/config/common"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	butanev1alpha1 "github.com/example/butane-operator/api/v1alpha1"
)

// ButaneConfigReconciler reconciles a ButaneConfig object
type ButaneConfigReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=butane.openshift.io,resources=butaneconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=butane.openshift.io,resources=butaneconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=butane.openshift.io,resources=butaneconfigs/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete

func (r *ButaneConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("butaneconfig", req.NamespacedName)

	// Récupérer l'objet ButaneConfig
	var butaneConfig butanev1alpha1.ButaneConfig
	if err := r.Get(ctx, req.NamespacedName, &butaneConfig); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Convertir le ButaneConfig en Ignition config
	ignitionConfig, rpt, err := config.TranslateBytes([]byte(butaneConfig.Spec.Config))
	if err != nil || rpt.IsFatal() {
		log.Error(err, "Erreur lors de la conversion de ButaneConfig en Ignition config")
		return ctrl.Result{}, err
	}

	ignitionJSON, err := json.Marshal(ignitionConfig)
	if err != nil {
		log.Error(err, "Erreur lors de la génération du JSON Ignition")
		return ctrl.Result{}, err
	}

	// Créer ou mettre à jour le secret contenant la configuration Ignition
	secretName := fmt.Sprintf("%s-ignition", butaneConfig.Name)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: butaneConfig.Namespace,
		},
		Data: map[string][]byte{
			"ignition.json": ignitionJSON,
		},
	}

	// Associer le secret au ButaneConfig
	if err := controllerutil.SetControllerReference(&butaneConfig, secret, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	// Créer ou mettre à jour le secret
	if err := r.Client.Create(ctx, secret); err != nil {
		if apierrors.IsAlreadyExists(err) {
			if err := r.Client.Update(ctx, secret); err != nil {
				return ctrl.Result{}, err
			}
		} else {
			return ctrl.Result{}, err
		}
	}

	// Mettre à jour le statut de ButaneConfig
	butaneConfig.Status.SecretName = secretName
	if err := r.Status().Update(ctx, &butaneConfig); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("ButaneConfig traité avec succès", "secretName", secretName)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ButaneConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&butanev1alpha1.ButaneConfig{}).
		Complete(r)
}
