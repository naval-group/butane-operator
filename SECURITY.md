# Security Policy

## Supported Versions

We release patches for security vulnerabilities. Which versions are eligible for receiving such patches depends on the CVSS v3.0 Rating:

| Version | Supported          |
| ------- | ------------------ |
| main    | :white_check_mark: |
| 0.x.x   | :white_check_mark: |

## Reporting a Vulnerability

The Butane Operator team and community take security bugs seriously. We appreciate your efforts to responsibly disclose your findings, and will make every effort to acknowledge your contributions.

### How to Report Security Issues

**Please do not report security vulnerabilities through public GitHub issues.**

Instead, please report them by email to: [INSERT SECURITY EMAIL]

If you prefer to encrypt your report, you can use our PGP key: [INSERT PGP KEY OR LINK]

You should receive a response within 48 hours. If for some reason you do not, please follow up via email to ensure we received your original message.

Please include the requested information listed below (as much as you can provide) to help us better understand the nature and scope of the possible issue:

* Type of issue (e.g. buffer overflow, SQL injection, cross-site scripting, etc.)
* Full paths of source file(s) related to the manifestation of the issue
* The location of the affected source code (tag/branch/commit or direct URL)
* Any special configuration required to reproduce the issue
* Step-by-step instructions to reproduce the issue
* Proof-of-concept or exploit code (if possible)
* Impact of the issue, including how an attacker might exploit the issue

This information will help us triage your report more quickly.

### Preferred Languages

We prefer all communications to be in English.

## Security Response Process

For each vulnerability report, we:

1. **Acknowledge** the receipt of the vulnerability report within 48 hours
2. **Confirm** the vulnerability and determine affected versions
3. **Audit** the code to find any similar problems
4. **Prepare** fixes for all supported versions
5. **Release** new versions with the fixes
6. **Announce** the vulnerability publicly after fixes are available

## Security Considerations for Users

### Running in Production

* Always run the latest stable version
* Use RBAC to limit operator permissions
* Monitor operator logs for unusual activity
* Keep Kubernetes cluster up to date
* Use network policies to restrict operator network access

### Configuration Security

* Validate all ButaneConfig resources before applying
* Use Kubernetes secrets for sensitive data
* Regularly audit RBAC permissions
* Enable audit logging in your cluster

### Container Security

* Use official container images from trusted registries
* Scan images for vulnerabilities regularly
* Run containers as non-root when possible
* Use read-only root filesystems where applicable

## Known Security Considerations

### ButaneConfig Processing

The operator processes ButaneConfig resources and converts them to Ignition configurations. Users should:

* Validate the content of ButaneConfig resources
* Be aware that processed configurations will be stored in Kubernetes secrets
* Ensure proper RBAC is in place to control access to these secrets

### RBAC Requirements

The operator requires specific RBAC permissions to function. Review the default RBAC configuration and adjust according to your security requirements.

## Security Updates

Security advisories will be published on:

* GitHub Security Advisories
* Release notes
* Project documentation

## Bug Bounty Program

We do not currently have a bug bounty program. We rely on the community to report security issues responsibly.

## Comments on this Policy

If you have suggestions on how this process could be improved, please submit a pull request or create an issue to discuss.