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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// ButaneConfigSpec defines the desired state of ButaneConfig
type ButaneConfigSpec struct {
	// An object that follows Butane specifications.
	// More info: https://coreos.github.io/butane/specs/
	Config runtime.RawExtension `json:"config,omitempty"`
}

// ButaneConfigStatus defines the observed state of ButaneConfig
type ButaneConfigStatus struct {
	// The name of the generated secret containing the ignition content in userdata key
	// More info: https://coreos.github.io/ignition/specs/
	SecretName string `json:"secretName,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ButaneConfig is a resource that transplane Butane config
// into an Ignition formatted secret.
type ButaneConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ButaneConfigSpec   `json:"spec,omitempty"`
	Status ButaneConfigStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ButaneConfigList contains a list of ButaneConfig
type ButaneConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ButaneConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ButaneConfig{}, &ButaneConfigList{})
}
