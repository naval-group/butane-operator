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

package e2e

import (
	"fmt"
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/naval-group/butane-operator/test/utils"
)

const namespace = "butane-operator-system"

var _ = Describe("controller", Ordered, func() {
	BeforeAll(func() {
		By("installing prometheus operator")
		Expect(utils.InstallPrometheusOperator()).To(Succeed())

		// NOTE: Cert-manager installation skipped for e2e tests
		// We use config/e2e which has webhooks disabled to avoid cert issues

		By("creating manager namespace")
		cmd := exec.Command("kubectl", "create", "ns", namespace)
		_, _ = utils.Run(cmd)
	})

	AfterAll(func() {
		By("uninstalling the Prometheus manager bundle")
		utils.UninstallPrometheusOperator()

		By("removing manager namespace")
		cmd := exec.Command("kubectl", "delete", "ns", namespace)
		_, _ = utils.Run(cmd)
	})

	Context("Operator", func() {
		It("should run successfully", func() {
			var controllerPodName string
			var err error

			// projectimage stores the name of the image used in the example
			var projectimage = "example.com/butane-operator:v0.0.1"

			By("building the manager(Operator) image")
			cmd := exec.Command("make", "docker-build", fmt.Sprintf("IMG=%s", projectimage))
			_, err = utils.Run(cmd)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("loading the the manager(Operator) image on Kind")
			err = utils.LoadImageToKindClusterWithName(projectimage)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("installing CRDs")
			cmd = exec.Command("make", "install")
			_, err = utils.Run(cmd)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("deploying the controller-manager (without webhooks for e2e)")
			// Use config/e2e which has webhooks disabled
			cmd = exec.Command("sh", "-c",
				fmt.Sprintf("cd config/manager && ../../bin/kustomize edit set image controller=%s && cd ../.. && bin/kustomize build config/e2e | kubectl apply -f -", projectimage))
			_, err = utils.Run(cmd)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("validating that the controller-manager pod is running as expected")
			verifyControllerUp := func() error {
				// Get pod name

				cmd = exec.Command("kubectl", "get",
					"pods", "-l", "control-plane=controller-manager",
					"-o", "go-template={{ range .items }}"+
						"{{ if not .metadata.deletionTimestamp }}"+
						"{{ .metadata.name }}"+
						"{{ \"\\n\" }}{{ end }}{{ end }}",
					"-n", namespace,
				)

				podOutput, err := utils.Run(cmd)
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				podNames := utils.GetNonEmptyLines(string(podOutput))
				if len(podNames) != 1 {
					return fmt.Errorf("expect 1 controller pods running, but got %d", len(podNames))
				}
				controllerPodName = podNames[0]
				ExpectWithOffset(2, controllerPodName).Should(ContainSubstring("controller-manager"))

				// Validate pod status
				cmd = exec.Command("kubectl", "get",
					"pods", controllerPodName, "-o", "jsonpath={.status.phase}",
					"-n", namespace,
				)
				status, err := utils.Run(cmd)
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				if string(status) != "Running" {
					return fmt.Errorf("controller pod in %s status", status)
				}
				return nil
			}
			EventuallyWithOffset(1, verifyControllerUp, time.Minute, time.Second).Should(Succeed())

		})

		It("should create Ignition secret from ButaneConfig", func() {
			By("creating a ButaneConfig resource")
			sampleFile := "config/samples/butane_v1alpha1_butaneconfig.yaml"
			// Apply without namespace flag - the sample has namespace: default
			cmd := exec.Command("kubectl", "apply", "-f", sampleFile)
			_, err := utils.Run(cmd)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("validating that the ButaneConfig was created")
			Eventually(func() error {
				cmd = exec.Command("kubectl", "get", "butaneconfig", "butaneconfig-sample", "-n", "default")
				_, err := utils.Run(cmd)
				return err
			}, time.Minute, time.Second).Should(Succeed())

			By("validating that the Ignition secret was created")
			Eventually(func() error {
				cmd = exec.Command("kubectl", "get", "secret", "butaneconfig-sample-ignition", "-n", "default")
				_, err := utils.Run(cmd)
				return err
			}, 2*time.Minute, time.Second).Should(Succeed())

			By("validating that the secret contains userdata key")
			cmd = exec.Command("kubectl", "get", "secret", "butaneconfig-sample-ignition",
				"-n", "default", "-o", "jsonpath={.data.userdata}")
			output, err := utils.Run(cmd)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())
			ExpectWithOffset(1, string(output)).NotTo(BeEmpty(), "Secret should contain userdata")

			By("validating that the ButaneConfig status was updated")
			cmd = exec.Command("kubectl", "get", "butaneconfig", "butaneconfig-sample",
				"-n", "default", "-o", "jsonpath={.status.secretName}")
			output, err = utils.Run(cmd)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())
			ExpectWithOffset(1, string(output)).To(Equal("butaneconfig-sample-ignition"))

			By("validating that the secret contains valid Ignition JSON")
			cmd = exec.Command("kubectl", "get", "secret", "butaneconfig-sample-ignition",
				"-n", "default", "-o", "jsonpath={.data.userdata}")
			output, err = utils.Run(cmd)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			// Decode base64 and validate JSON structure
			cmd = exec.Command("bash", "-c",
				fmt.Sprintf("echo '%s' | base64 -d | jq -e '.ignition.version'", string(output)))
			_, err = utils.Run(cmd)
			ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Secret should contain valid Ignition JSON with ignition.version")

			By("cleaning up the ButaneConfig resource")
			cmd = exec.Command("kubectl", "delete", "-f", sampleFile)
			_, err = utils.Run(cmd)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("validating that the secret was cleaned up (owner reference)")
			Eventually(func() error {
				cmd = exec.Command("kubectl", "get", "secret", "butaneconfig-sample-ignition", "-n", "default")
				_, err := utils.Run(cmd)
				if err == nil {
					return fmt.Errorf("secret still exists")
				}
				return nil
			}, time.Minute, time.Second).Should(Succeed())
		})

		It("should update secret when ButaneConfig is modified", func() {
			By("creating a temporary ButaneConfig manifest")
			tempManifest := "/tmp/butaneconfig-update-test.yaml"
			initialConfig := `apiVersion: butane.operators.naval-group.com/v1alpha1
kind: ButaneConfig
metadata:
  name: butaneconfig-update-test
  namespace: default
spec:
  config:
    variant: fcos
    version: 1.5.0
    storage:
      files:
        - path: /etc/hostname
          contents:
            inline: initial-hostname
`
			cmd := exec.Command("bash", "-c", fmt.Sprintf("cat > %s << 'EOF'\n%s\nEOF", tempManifest, initialConfig))
			_, err := utils.Run(cmd)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("creating the initial ButaneConfig")
			cmd = exec.Command("kubectl", "apply", "-f", tempManifest)
			_, err = utils.Run(cmd)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("waiting for initial secret creation")
			var initialSecretData string
			Eventually(func() error {
				cmd = exec.Command("kubectl", "get", "secret", "butaneconfig-update-test-ignition",
					"-n", "default", "-o", "jsonpath={.data.userdata}")
				output, err := utils.Run(cmd)
				if err != nil {
					return err
				}
				initialSecretData = string(output)
				if initialSecretData == "" {
					return fmt.Errorf("secret data is empty")
				}
				return nil
			}, 2*time.Minute, time.Second).Should(Succeed())

			By("updating the ButaneConfig with new content")
			updatedConfig := `apiVersion: butane.operators.naval-group.com/v1alpha1
kind: ButaneConfig
metadata:
  name: butaneconfig-update-test
  namespace: default
spec:
  config:
    variant: fcos
    version: 1.5.0
    storage:
      files:
        - path: /etc/hostname
          contents:
            inline: updated-hostname
`
			cmd = exec.Command("bash", "-c", fmt.Sprintf("cat > %s << 'EOF'\n%s\nEOF", tempManifest, updatedConfig))
			_, err = utils.Run(cmd)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			cmd = exec.Command("kubectl", "apply", "-f", tempManifest)
			_, err = utils.Run(cmd)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("validating that the secret was updated")
			Eventually(func() bool {
				cmd = exec.Command("kubectl", "get", "secret", "butaneconfig-update-test-ignition",
					"-n", "default", "-o", "jsonpath={.data.userdata}")
				output, err := utils.Run(cmd)
				if err != nil {
					return false
				}
				updatedSecretData := string(output)
				return updatedSecretData != initialSecretData && updatedSecretData != ""
			}, 2*time.Minute, time.Second).Should(BeTrue(), "Secret should be updated with new content")

			By("cleaning up the test resources")
			cmd = exec.Command("kubectl", "delete", "-f", tempManifest)
			_, _ = utils.Run(cmd)
			cmd = exec.Command("rm", "-f", tempManifest)
			_, _ = utils.Run(cmd)
		})

		It("should handle invalid ButaneConfig gracefully", func() {
			By("creating an invalid ButaneConfig manifest")
			tempManifest := "/tmp/butaneconfig-invalid-test.yaml"
			invalidConfig := `apiVersion: butane.operators.naval-group.com/v1alpha1
kind: ButaneConfig
metadata:
  name: butaneconfig-invalid-test
  namespace: default
spec:
  config:
    variant: fcos
    # Missing required version field
    storage:
      files:
        - path: /etc/test
          contents:
            inline: test
`
			cmd := exec.Command("bash", "-c", fmt.Sprintf("cat > %s << 'EOF'\n%s\nEOF", tempManifest, invalidConfig))
			_, err := utils.Run(cmd)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("attempting to create the invalid ButaneConfig")
			cmd = exec.Command("kubectl", "apply", "-f", tempManifest)
			_, err = utils.Run(cmd)
			ExpectWithOffset(1, err).NotTo(HaveOccurred()) // Creation may succeed but reconciliation should fail

			By("validating that no secret is created for invalid config")
			Consistently(func() error {
				cmd = exec.Command("kubectl", "get", "secret", "butaneconfig-invalid-test-ignition", "-n", "default")
				_, err := utils.Run(cmd)
				if err == nil {
					return fmt.Errorf("secret should not exist for invalid config")
				}
				return nil
			}, 30*time.Second, 2*time.Second).Should(Succeed())

			By("cleaning up the test resources")
			cmd = exec.Command("kubectl", "delete", "-f", tempManifest, "--ignore-not-found")
			_, _ = utils.Run(cmd)
			cmd = exec.Command("rm", "-f", tempManifest)
			_, _ = utils.Run(cmd)
		})
	})
})
