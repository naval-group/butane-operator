# Butane Operator Examples

This directory contains example ButaneConfig manifests demonstrating various use cases.

## Examples

### 01-basic-motd.yaml
A simple example that sets the message of the day (MOTD) file.

```bash
kubectl apply -f 01-basic-motd.yaml
```

### 02-systemd-service.yaml
Demonstrates creating a custom systemd service and configuration file.

```bash
kubectl apply -f 02-systemd-service.yaml
```

### 03-user-management.yaml
Shows how to configure users, SSH keys, and user-specific files.

**Note:** Update the SSH key in this file before deploying.

```bash
kubectl apply -f 03-user-management.yaml
```

### 04-docker-compose.yaml
Example of deploying a Docker Compose application as a systemd service.

```bash
kubectl apply -f 04-docker-compose.yaml
```

### 05-network-config.yaml
Network configuration example including sysctl settings and custom hosts file.

```bash
kubectl apply -f 05-network-config.yaml
```

## Applying All Examples

To apply all examples at once:

```bash
kubectl apply -f .
```

## Viewing Generated Secrets

Each ButaneConfig will generate a corresponding secret with the Ignition configuration:

```bash
# List all secrets
kubectl get secrets | grep ignition

# View a specific secret
kubectl get secret basic-motd-ignition -o yaml

# Decode the Ignition data
kubectl get secret basic-motd-ignition -o jsonpath='{.data.ignition}' | base64 -d | jq
```

## Using with KubeVirt

Here's an example of using the generated secret with a KubeVirt VirtualMachine:

```yaml
apiVersion: kubevirt.io/v1
kind: VirtualMachine
metadata:
  name: fcos-vm
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
        - name: cloudinitdisk
          cloudInitConfigDrive:
            secretRef:
              name: basic-motd-ignition  # Reference to the generated secret
```

## Cleanup

To delete all example ButaneConfigs:

```bash
kubectl delete -f .
```
