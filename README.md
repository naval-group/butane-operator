# butane-operator

[![CodeQL](https://github.com/naval-group/butane-operator/actions/workflows/codeql-analysis.yml/badge.svg)](https://github.com/naval-group/butane-operator/actions/workflows/codeql-analysis.yml)
[![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/naval-group/butane-operator/badge)](https://scorecard.dev/viewer/?uri=github.com/naval-group/butane-operator)

![](assets/title.png)

## Description

The **Butane Operator** is a Kubernetes operator that automates the conversion of Butane configuration files from specific CRD into Ignition configurations into a secret. For example to give this secret to a KubeVirt CoreOS Machine.
The operator watches for custom ButaneConfig resources and automatically translates them into Ignition JSON, storing the results in Kubernetes secrets for secure and easy access.

Butane is a tool that simplifies the creation of Ignition files, which are essential for provisioning CoreOS-based systems.

Documenations Links : [Butane](https://coreos.github.io/butane/) | [Ignition](https://coreos.github.io/ignition/)

## Features

- **Automatic Conversion**: Watches for ButaneConfig custom resources and automatically converts them to Ignition configurations.
- **Secure Storage**: Stores the resulting Ignition JSON in Kubernetes secrets.
- **Kubernetes-Native**: Leverages Kubernetes' native capabilities for managing and storing configurations.
- **Extensible**: Can be extended to support additional features or custom workflows.

## Usage

Create a ButaneConfig Resource:

Define your Butane configuration in a YAML file and apply it to your cluster.

```yaml
apiVersion: butane.operators.naval-group.com/v1alpha1
kind: ButaneConfig
metadata:
  name: my-butane-config
  namespace: default
spec:
  config:
    variant: fcos
    version: 1.5.0
    storage:
      files:
        - path: /etc/motd
          contents:
            inline: |
              Hello, CoreOS!
```

Apply the resource:

```sh
kubectl apply -f my-butane-config.yaml
```

Check the Generated Secret:
The operator will generate a Kubernetes secret containing the Ignition configuration.

```sh
kubectl get secret my-butane-config-ignition -o yaml
```

You can directly use secret inside KubeVirt VirtualMachine

```yaml
apiVersion: kubevirt.io/v1
kind: VirtualMachine
metadata:
  name: my-fcos
spec:
  runStrategy: Always
  template:
    spec:
      domain:
        devices:
          disks:
            - name: containerdisk
              disk:
                bus: virtio
            - name: cloudinitdisk
              disk:
                bus: virtio
          rng: {}
        resources:
          requests:
            memory: 2048M
      volumes:
        - name: containerdisk
          containerDisk:
            image: quay.io/fedora/fedora-coreos-kubevirt:stable
            imagePullPolicy: Always
        - name: cloudinitdisk
          cloudInitConfigDrive:
            secretRef:
              name: my-butane-config-ignition
```

## Getting Started

### Prerequisites

- go version v1.22.0+
- docker version 17.03+ (or podman)
- kubectl version v1.11.3+
- Access to a Kubernetes v1.11.3+ cluster

### Quick Install

Install the latest version directly from GitHub:

```sh
kubectl apply -f https://github.com/naval-group/butane-operator/releases/latest/download/install.yaml
```

Or install a specific version:

```sh
kubectl apply -f https://github.com/naval-group/butane-operator/releases/download/v0.0.1/install.yaml
```

### To Deploy on the cluster (from source)

**Build and push your image to the location specified by `IMG`:**

```sh
make docker-build docker-push IMG=ghcr.io/naval-group/butane-operator:tag
```

**NOTE:** This image ought to be published in the registry you specified.
And it is required to have access to pull the image from the working environment.
Make sure you have the proper permission to the registry if the above commands don't work.

**Install the CRDs into the cluster:**

```sh
make install
```

**Deploy the Manager to the cluster with the image specified by `IMG`:**

```sh
make deploy IMG=ghcr.io/naval-group/butane-operator:tag
```

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin
> privileges or be logged in as admin.

**Create instances of your solution**

You can apply the samples (examples) from the examples folder:

```sh
kubectl apply -k examples/
```

> **NOTE**: See the [examples directory](examples/) for various use cases including:
> - Basic MOTD configuration
> - Systemd service management
> - User management with SSH keys
> - Docker Compose deployment
> - Network configuration

### To Uninstall

**Delete the instances (CRs) from the cluster:**

```sh
kubectl delete -k examples/
```

**Delete the APIs(CRDs) from the cluster:**

```sh
make uninstall
```

**UnDeploy the controller from the cluster:**

```sh
make undeploy
```

## Project Distribution

Following are the steps to build the installer and distribute this project to users.

1. Build the installer for the image built and published in the registry:

```sh
make build-installer IMG=ghcr.io/naval-group/butane-operator:v0.0.1
```

**NOTE:** The makefile target mentioned above generates an `install.yaml`
file in the dist directory. This file contains all the resources built
with Kustomize, which are necessary to install this project without
its dependencies.

2. Using the installer

Users can install the project directly from GitHub releases:

```sh
# Install latest release
kubectl apply -f https://github.com/naval-group/butane-operator/releases/latest/download/install.yaml

# Or install a specific version
kubectl apply -f https://github.com/naval-group/butane-operator/releases/download/v0.0.1/install.yaml
```

### Container Images

Official container images are available at:

```
ghcr.io/naval-group/butane-operator:latest
ghcr.io/naval-group/butane-operator:v0.0.1
```

Images are signed with cosign. Verify with:

```sh
cosign verify ghcr.io/naval-group/butane-operator:v0.0.1 \
  --certificate-identity-regexp="https://github.com/naval-group/butane-operator" \
  --certificate-oidc-issuer="https://token.actions.githubusercontent.com"
```

## Examples

Comprehensive examples are available in the [examples/](examples/) directory:

- **[01-basic-motd.yaml](examples/01-basic-motd.yaml)** - Simple MOTD file configuration
- **[02-systemd-service.yaml](examples/02-systemd-service.yaml)** - Custom systemd service
- **[03-user-management.yaml](examples/03-user-management.yaml)** - User creation with SSH keys
- **[04-docker-compose.yaml](examples/04-docker-compose.yaml)** - Docker Compose deployment
- **[05-network-config.yaml](examples/05-network-config.yaml)** - Network configuration with sysctl

See the [examples README](examples/README.md) for detailed usage instructions.

## Contributing

Contributions are welcome! Please see our [Contributing Guide](CONTRIBUTING.md) for details on:

- Development setup and workflow
- Coding standards and best practices
- Testing requirements
- Submitting pull requests
- Reporting issues and feature requests

**NOTE:** Run `make help` for more information on all potential `make` targets

## Resources

- [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html) - Framework used for building this operator
- [Butane Specification](https://coreos.github.io/butane/) - Input configuration format
- [Ignition Specification](https://coreos.github.io/ignition/) - Output configuration format
- [Contributing Guide](CONTRIBUTING.md) - How to contribute to this project
- [Security Policy](SECURITY.md) - How to report security vulnerabilities

## License

This project is licensed under the LGPL-3.0-or-later - see the [LICENSE](LICENSE) file for details.
