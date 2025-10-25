# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial open source release
- LGPL 3.0 license
- Community governance files (CONTRIBUTING.md, CODE_OF_CONDUCT.md, SECURITY.md)
- GitHub issue and pull request templates
- OpenSSF Scorecard integration
- Dependabot configuration for dependency updates

### Changed
- License changed from Apache 2.0 to LGPL 3.0
- Updated module path to github.com/naval-group/butane-operator

### Security
- Added security policy and reporting guidelines
- Implemented OpenSSF Scorecard for supply-chain security

## [0.1.0] - TBD

### Added
- ButaneConfig custom resource definition
- Controller for automatic Butane to Ignition conversion
- Kubernetes secret generation for Ignition configurations
- Support for Fedora CoreOS (FCOS) configurations
- Integration with KubeVirt VirtualMachines
- Comprehensive test suite
- Docker container support
- Makefile for build automation

### Features
- Automatic conversion from Butane YAML to Ignition JSON
- Secure storage of generated configurations in Kubernetes secrets
- Kubernetes-native resource management
- Extensible architecture for future enhancements

### Documentation
- Complete README with usage examples
- Getting started guide
- Development setup instructions
- Contributing guidelines

---

## Release Process

1. Update the version in this file
2. Update version tags in relevant files
3. Create a git tag with the version number
4. Create a GitHub release with release notes
5. Build and publish container images
6. Update documentation as needed

## Security Releases

Security releases follow the same process but may be expedited. See [SECURITY.md](SECURITY.md) for vulnerability reporting.

## Support

For questions about releases or to report issues:
- Create an issue on GitHub
- Check the documentation
- Contact the maintainers

## Links

- [GitHub Repository](https://github.com/naval-group/butane-operator)
- [Container Images](https://quay.io/repository/naval-group/butane-operator)
- [Documentation](https://github.com/naval-group/butane-operator/tree/main/docs)