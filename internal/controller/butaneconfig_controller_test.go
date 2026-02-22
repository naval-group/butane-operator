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
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	butanev1alpha1 "github.com/naval-group/butane-operator/api/v1alpha1"
)

var _ = Describe("ButaneConfig Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}
		butaneconfig := &butanev1alpha1.ButaneConfig{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind ButaneConfig")
			err := k8sClient.Get(ctx, typeNamespacedName, butaneconfig)
			if err != nil && errors.IsNotFound(err) {
				// Valid minimal Butane configuration as a map
				butaneConfig := map[string]interface{}{
					"variant": "fcos",
					"version": "1.5.0",
					"storage": map[string]interface{}{
						"files": []map[string]interface{}{
							{
								"path": "/etc/hostname",
								"mode": 420, // 0644 in decimal
								"contents": map[string]interface{}{
									"inline": "test-hostname",
								},
							},
						},
					},
				}

				// Marshal the config to JSON
				configJSON, err := json.Marshal(butaneConfig)
				Expect(err).NotTo(HaveOccurred())

				resource := &butanev1alpha1.ButaneConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: butanev1alpha1.ButaneConfigSpec{
						Config: runtime.RawExtension{
							Raw: configJSON,
						},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &butanev1alpha1.ButaneConfig{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance ButaneConfig")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &ButaneConfigReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				Log:      ctrl.Log.WithName("controllers").WithName("ButaneConfig"),
				Recorder: events.NewFakeRecorder(100),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking if the Secret was created")
			secretName := resourceName + "-ignition"
			secret := &corev1.Secret{}
			secretKey := types.NamespacedName{
				Name:      secretName,
				Namespace: "default",
			}
			Eventually(func() error {
				return k8sClient.Get(ctx, secretKey, secret)
			}).Should(Succeed())

			By("Verifying the Secret contains ignition data")
			Expect(secret.Data).To(HaveKey("userdata"))
			Expect(secret.Data["userdata"]).NotTo(BeEmpty())

			By("Verifying the Butane config was converted to valid Ignition JSON")
			var ignitionConfig map[string]interface{}
			err = json.Unmarshal(secret.Data["userdata"], &ignitionConfig)
			Expect(err).NotTo(HaveOccurred(), "Secret should contain valid JSON")

			// Verify Ignition format structure
			Expect(ignitionConfig).To(HaveKey("ignition"), "Should have ignition key")
			ignitionMeta, ok := ignitionConfig["ignition"].(map[string]interface{})
			Expect(ok).To(BeTrue(), "ignition should be an object")
			Expect(ignitionMeta).To(HaveKey("version"), "Should have ignition version")

			// Verify the file from Butane config was converted
			Expect(ignitionConfig).To(HaveKey("storage"), "Should have storage section")
			storage, ok := ignitionConfig["storage"].(map[string]interface{})
			Expect(ok).To(BeTrue(), "storage should be an object")
			Expect(storage).To(HaveKey("files"), "Should have files in storage")

			files, ok := storage["files"].([]interface{})
			Expect(ok).To(BeTrue(), "files should be an array")
			Expect(files).To(HaveLen(1), "Should have one file")

			file, ok := files[0].(map[string]interface{})
			Expect(ok).To(BeTrue(), "file should be an object")
			Expect(file).To(HaveKey("path"), "File should have path")
			Expect(file["path"]).To(Equal("/etc/hostname"), "File path should match Butane config")

			// Verify file contents were encoded (Ignition uses base64 for inline content)
			Expect(file).To(HaveKey("contents"), "File should have contents")
			contents, ok := file["contents"].(map[string]interface{})
			Expect(ok).To(BeTrue(), "contents should be an object")
			Expect(contents).To(HaveKey("source"), "Contents should have source (base64 data URL)")

			By("Checking if the ButaneConfig status was updated")
			updatedButaneConfig := &butanev1alpha1.ButaneConfig{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedButaneConfig)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedButaneConfig.Status.SecretName).To(Equal(secretName))
		})

		It("should handle invalid Butane configuration", func() {
			By("Creating a ButaneConfig with invalid config")
			invalidResourceName := "test-invalid-resource"
			invalidTypeNamespacedName := types.NamespacedName{
				Name:      invalidResourceName,
				Namespace: "default",
			}

			// Invalid Butane config (missing required fields)
			invalidConfig := map[string]interface{}{
				"variant": "fcos",
				// Missing version field - this should cause an error
			}

			configJSON, err := json.Marshal(invalidConfig)
			Expect(err).NotTo(HaveOccurred())

			invalidResource := &butanev1alpha1.ButaneConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      invalidResourceName,
					Namespace: "default",
				},
				Spec: butanev1alpha1.ButaneConfigSpec{
					Config: runtime.RawExtension{
						Raw: configJSON,
					},
				},
			}
			Expect(k8sClient.Create(ctx, invalidResource)).To(Succeed())

			By("Reconciling the invalid resource")
			controllerReconciler := &ButaneConfigReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				Log:      ctrl.Log.WithName("controllers").WithName("ButaneConfig"),
				Recorder: events.NewFakeRecorder(100),
			}

			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: invalidTypeNamespacedName,
			})
			Expect(err).To(HaveOccurred(), "Should return error for invalid Butane config")

			By("Verifying no Secret was created for invalid config")
			secretName := invalidResourceName + "-ignition"
			secret := &corev1.Secret{}
			secretKey := types.NamespacedName{
				Name:      secretName,
				Namespace: "default",
			}
			err = k8sClient.Get(ctx, secretKey, secret)
			Expect(errors.IsNotFound(err)).To(BeTrue(), "Secret should not exist for invalid config")

			By("Cleanup the invalid resource")
			Expect(k8sClient.Delete(ctx, invalidResource)).To(Succeed())
		})

		It("should handle missing Config field", func() {
			By("Creating a ButaneConfig without config field")
			missingConfigResourceName := "test-missing-config"
			missingConfigNamespacedName := types.NamespacedName{
				Name:      missingConfigResourceName,
				Namespace: "default",
			}

			resourceWithoutConfig := &butanev1alpha1.ButaneConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      missingConfigResourceName,
					Namespace: "default",
				},
				Spec: butanev1alpha1.ButaneConfigSpec{
					// Config is not set
				},
			}
			Expect(k8sClient.Create(ctx, resourceWithoutConfig)).To(Succeed())

			By("Reconciling the resource without config")
			controllerReconciler := &ButaneConfigReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				Log:      ctrl.Log.WithName("controllers").WithName("ButaneConfig"),
				Recorder: events.NewFakeRecorder(100),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: missingConfigNamespacedName,
			})
			Expect(err).To(HaveOccurred(), "Should return error for missing config")
			Expect(err.Error()).To(ContainSubstring("missing Config"), "Error should indicate missing config")

			By("Cleanup the resource without config")
			Expect(k8sClient.Delete(ctx, resourceWithoutConfig)).To(Succeed())
		})
	})
})
