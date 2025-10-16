# Changelog

All notable changes to the kube-stager project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2025-10-15

### Added
- Config validation for ports, deadlines, TTLs, and backoff limits
- Comprehensive functional test plan (FUNCTIONAL_TEST_PLAN_V1.0.0.md)

### Changed
- **BREAKING**: Upgraded to Go 1.24
- **BREAKING**: Upgraded controller-runtime from v0.13.0 to v0.20.4
- **BREAKING**: Upgraded Kubernetes APIs from v0.25.0 to v0.32.1
- **BREAKING**: Upgraded kubebuilder from v4-alpha to v4
- **BREAKING**: Redis handler now properly uses TLS and authentication. TLS certificate verification is enabled by default - set `verifyTlsServerCertificate: false` in RedisConfig for self-signed certificates
- Replaced deprecated config/v1alpha1 with custom ProjectConfig implementation
- Migrated webhook.Defaulter to admission.CustomDefaulter
- Updated admission.Decoder from pointer to interface type
- Upgraded sentry-go from v0.11.0 to v0.29.0
- Upgraded go-sql-driver/mysql from v1.5.0 to v1.9.2
- Upgraded sethvargo/go-password from v0.2.0 to v0.3.1
- Upgraded onsi/ginkgo/v2 from v2.22.0 to v2.26.0
- Upgraded onsi/gomega from v1.36.1 to v1.38.2
- Upgraded sigs.k8s.io/yaml from v1.4.0 to v1.6.0
- Upgraded go-logr/logr from v1.4.2 to v1.4.3
- Replaced grokify/gotilla with grokify/mogo v0.71.3

### Fixed
- **SECURITY**: SQL injection vulnerability in MySQL handler (added identifier quoting)
- **SECURITY**: Redis client resource leak (added proper cleanup)
- Webhook decoder injection for controller-runtime v0.20.4 (fixes nil pointer panics in stagingsite, serviceconfig, and backup webhooks)
- Ignored error from password generation in webhook
- Variable shadowing in main.go

### Deprecated
- None

### Removed
- Deprecated config/v1alpha1.ControllerManagerConfigurationSpec

## [0.3.0] and earlier
See git history for changes in previous releases.
