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
package v1alpha1

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/coreos/butane/config"
	"github.com/coreos/butane/config/common"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var butaneconfiglog = logf.Log.WithName("butaneconfig-resource")

// SetupWebhookWithManager will setup the manager to manage the webhooks
func (r *ButaneConfig) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &ButaneConfig{}).
		WithValidator(&ButaneConfigCustomValidator{}).
		Complete()
}

//+kubebuilder:webhook:path=/validate-butane-operators-naval-group-com-v1alpha1-butaneconfig,mutating=false,failurePolicy=fail,sideEffects=None,groups=butane.operators.naval-group.com,resources=butaneconfigs,verbs=create;update,versions=v1alpha1,name=validating.butaneconfigs.operators.naval-group.com,admissionReviewVersions=v1

// +kubebuilder:object:generate=false

// ButaneConfigCustomValidator implements admission.Validator[*ButaneConfig]
type ButaneConfigCustomValidator struct{}

// ValidateCreate implements validation logic for ButaneConfig creation
func (v *ButaneConfigCustomValidator) ValidateCreate(ctx context.Context, obj *ButaneConfig) (admission.Warnings, error) {
	butaneconfiglog.Info("validate create", "name", obj.Name)

	// Validate the Butane configuration on creation
	if err := validateButaneConfig(obj); err != nil {
		return nil, err
	}

	return nil, nil
}

// ValidateUpdate implements validation logic for ButaneConfig updates
func (v *ButaneConfigCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj *ButaneConfig) (admission.Warnings, error) {
	butaneconfiglog.Info("validate update", "name", newObj.Name)

	// Validate the Butane configuration on update
	if err := validateButaneConfig(newObj); err != nil {
		return nil, err
	}

	return nil, nil
}

// ValidateDelete implements validation logic for ButaneConfig deletion
func (v *ButaneConfigCustomValidator) ValidateDelete(ctx context.Context, obj *ButaneConfig) (admission.Warnings, error) {
	butaneconfiglog.Info("validate delete", "name", obj.Name)
	// Optionally implement validation on delete, if necessary
	return nil, nil
}

// validateButaneConfig checks if the Butane configuration is valid by attempting to translate it to Ignition
func validateButaneConfig(r *ButaneConfig) error {
	var butane interface{}
	if err := json.Unmarshal(r.Spec.Config.Raw, &butane); err != nil {
		return fmt.Errorf("failed to unmarshal Butane config: %v", err)
	}

	// Attempt to translate Butane config to Ignition
	_, report, err := config.TranslateBytes(r.Spec.Config.Raw, common.TranslateBytesOptions{})
	if err != nil || len(report.Entries) > 0 {
		return fmt.Errorf("failed to translate Butane to Ignition: %v", report.String())
	}

	return nil
}
