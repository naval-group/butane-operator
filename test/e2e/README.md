# End-to-End (E2E) Tests

This directory contains end-to-end tests for the Butane Operator. These tests run against a real Kubernetes cluster (typically Kind) and verify the complete functionality of the operator.

## Prerequisites

Before running e2e tests, ensure you have the following tools installed:

- **kubectl**: Kubernetes command-line tool
- **kind**: Kubernetes in Docker (for local testing)
- **docker**: Container runtime
- **jq**: JSON processor (for validating Ignition JSON)

**Note**: The e2e tests use a special configuration (`config/e2e`) that has webhooks disabled to avoid cert-manager certificate issues during testing. This allows the tests to run faster and more reliably.

### Installing Prerequisites

```bash
# macOS
brew install kubectl kind docker jq

# Linux
# Install kubectl
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

# Install kind
curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-amd64
chmod +x ./kind
sudo mv ./kind /usr/local/bin/kind

# Install jq
sudo apt-get install jq
```

## Test Scenarios

The e2e test suite includes the following scenarios:

### 1. Operator Deployment Test
- **Test**: "should run successfully"
- **Description**: Verifies that the operator can be built, deployed, and runs successfully in a Kubernetes cluster
- **Steps**:
  - Builds the operator Docker image
  - Loads the image into Kind cluster
  - Installs CRDs
  - Deploys the controller-manager
  - Verifies the controller pod is running

### 2. ButaneConfig to Ignition Conversion Test
- **Test**: "should create Ignition secret from ButaneConfig"
- **Description**: Tests the core functionality of converting Butane configurations to Ignition secrets
- **Steps**:
  - Creates a ButaneConfig resource from the sample
  - Verifies the ButaneConfig is created successfully
  - Validates that an Ignition secret is automatically created
  - Checks the secret contains the `userdata` key
  - Verifies the ButaneConfig status is updated with the secret name
  - Validates the secret contains valid Ignition JSON with proper structure
  - Deletes the ButaneConfig and verifies cleanup (owner reference)

### 3. ButaneConfig Update Test
- **Test**: "should update secret when ButaneConfig is modified"
- **Description**: Tests that modifying a ButaneConfig triggers secret updates
- **Steps**:
  - Creates a ButaneConfig with initial content
  - Waits for initial secret creation and captures its data
  - Updates the ButaneConfig with different content
  - Verifies the secret data changes to reflect the update
  - Cleans up test resources

### 4. Invalid Configuration Handling Test
- **Test**: "should handle invalid ButaneConfig gracefully"
- **Description**: Tests error handling for invalid Butane configurations
- **Steps**:
  - Creates an invalid ButaneConfig (missing required version field)
  - Verifies resource creation succeeds (API validation passes)
  - Confirms no secret is created for invalid configs (reconciliation fails)
  - Ensures the operator doesn't crash or enter error loops
  - Cleans up test resources

## Running E2E Tests

### Setup Kind Cluster

First, create a Kind cluster if you don't have one:

```bash
kind create cluster --name butane-operator
```

### Run Tests

Run the complete e2e test suite:

```bash
make test-e2e
```

Or run tests directly with Go:

```bash
cd test/e2e
go test -v -timeout 30m
```

### Run Specific Test

To run a specific test scenario:

```bash
cd test/e2e
go test -v -timeout 30m -ginkgo.focus "should create Ignition secret"
```

### Cleanup

After testing, you can delete the Kind cluster:

```bash
kind delete cluster --name butane-operator
```

## Test Configuration

### Environment Variables

- `KIND_CLUSTER`: Name of the Kind cluster (default: "kind")

Example:
```bash
export KIND_CLUSTER=my-test-cluster
make test-e2e
```

### Timeouts

The tests use the following timeouts:
- Controller startup: 1 minute
- Secret creation: 2 minutes
- Resource updates: 2 minutes
- Cleanup verification: 1 minute

## Debugging Failed Tests

### View Controller Logs

If tests fail, check the controller logs:

```bash
kubectl logs -n butane-operator-system \
  -l control-plane=controller-manager \
  --tail=100 -f
```

### Check ButaneConfig Status

```bash
kubectl get butaneconfig -n butane-operator-system -o yaml
```

### Check Secrets

```bash
kubectl get secrets -n butane-operator-system
kubectl get secret <secret-name> -n butane-operator-system -o yaml
```

### Decode Secret Data

To view the actual Ignition JSON in a secret:

```bash
kubectl get secret <secret-name> -n butane-operator-system \
  -o jsonpath='{.data.userdata}' | base64 -d | jq .
```

### Check Events

```bash
kubectl get events -n butane-operator-system --sort-by='.lastTimestamp'
```

## CI/CD Integration

These tests are designed to run in CI/CD pipelines. Example GitHub Actions workflow:

```yaml
name: E2E Tests
on: [push, pull_request]

jobs:
  e2e:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.24'

      - name: Install dependencies
        run: |
          sudo apt-get update
          sudo apt-get install -y jq

      - name: Create Kind cluster
        uses: helm/kind-action@v1
        with:
          cluster_name: butane-operator

      - name: Run e2e tests
        run: make test-e2e
```

## Troubleshooting

### "Connection refused" errors
- Ensure Kind cluster is running: `kind get clusters`
- Verify kubectl context: `kubectl config current-context`

### "Image not found" errors
- Ensure Docker is running
- Check image was built: `docker images | grep butane-operator`

### "Timeout" errors
- Increase timeout values in test code
- Check cluster resources: `kubectl top nodes`
- Verify controller logs for errors

### "Secret not created" errors
- Check controller is running: `kubectl get pods -n butane-operator-system`
- View controller logs for conversion errors
- Verify ButaneConfig is valid Butane format

## Contributing

When adding new e2e tests:

1. Follow the existing test structure using Ginkgo/Gomega
2. Use appropriate timeouts with `Eventually` and `Consistently`
3. Clean up all resources in test cleanup sections
4. Add descriptive `By()` messages for each step
5. Update this README with new test scenarios
