# Changelog

All notable changes to the kube-stager project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.3] - 2026-10-29

### Added
- Prometheus metrics instrumentation with 15 custom application-specific metrics covering:
  - Resource inventory gauges (staging sites, databases, jobs by state)
  - StagingSite lifecycle counters (state transitions, auto-disable, auto-delete) and provisioning duration histogram
  - Database operation counters and duration histograms (MySQL, MongoDB, Redis)
  - Job completion counters and duration histograms (init, migration, backup)
  - Error counter with controller and finality classification
  - Webhook denied counter with reason labels
  - Build info gauge with version from build-time ldflags
- Test isolation guard: envtest suites refuse to run against a non-loopback API server and never read the
  developer's kubeconfig

### Changed
- Upgraded Go builder image and CI to Go 1.27 (module minimum stays Go 1.26)
- Upgraded k8s.io/api, k8s.io/apimachinery and k8s.io/client-go from v0.35.2 to v0.36.6
- Upgraded sigs.k8s.io/controller-runtime from v0.23.3 to v0.24.1
- Upgraded sentry-go from v0.43.0 to v0.50.0
- Upgraded go-sql-driver/mysql from v1.9.3 to v1.10.1
- Upgraded grokify/mogo from v0.73.4 to v0.75.0
- Upgraded sethvargo/go-password from v0.3.1 to v0.4.0
- Upgraded mongo-driver from v1.17.9 to v1.17.10
- Upgraded prometheus/client_golang from v1.23.2 to v1.25.0
- Upgraded onsi/ginkgo/v2 from v2.27.2 to v2.33.1 and onsi/gomega from v1.38.2 to v1.44.0
- Upgraded go-logr/logr from v1.4.3 to v1.4.4
- Regenerated CRDs with controller-gen v0.23.0. The ServiceConfig pod template follows Kubernetes 1.36: the alpha
  `workloadRef` field is removed and `schedulingGroup` is added
- Tested against Kubernetes 1.29, 1.32, 1.35, 1.36 and 1.37

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
