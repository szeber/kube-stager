# Functional Test Plan - kube-stager v1.0.0 Release

**Release Context**: Major upgrade from Kubernetes v0.25 to v0.32, restoring config file functionality, multiple API migrations

**Version**: 1.0.0
**Date**: 2025-10-15
**Status**: Draft

---

## Table of Contents
1. [Critical Release Success Criteria](#critical-release-success-criteria)
2. [Test Environment Setup](#test-environment-setup)
3. [Test Scenarios](#test-scenarios)
   - [1. Config File Loading](#1-config-file-loading)
   - [2. Backward Compatibility](#2-backward-compatibility)
   - [3. New Functionality](#3-new-functionality)
   - [4. Integration Scenarios](#4-integration-scenarios)
   - [5. Failure Scenarios](#5-failure-scenarios)
   - [6. Upgrade Path](#6-upgrade-path)
4. [Smoke Test Checklist](#smoke-test-checklist)
5. [Rollback Test Scenarios](#rollback-test-scenarios)
6. [Test Execution Strategy](#test-execution-strategy)
7. [Risk Assessment](#risk-assessment)

---

## Critical Release Success Criteria

The following MUST work for v1.0.0 to be considered production-ready:

1. **P0 - BLOCKING**: Config file loading with health, metrics, and webhook port settings
2. **P0 - BLOCKING**: In-place upgrade from v0.3.0 without service disruption
3. **P0 - BLOCKING**: Existing StagingSite CRs continue functioning after upgrade
4. **P0 - BLOCKING**: Webhook mutations and validations operational (ServiceConfig, StagingSite, Backup)
5. **P0 - BLOCKING**: Job controllers use config values correctly (init, migration, backup)
6. **P0 - BLOCKING**: Health and readiness probes respond correctly
7. **P1 - HIGH**: Leader election when enabled
8. **P1 - HIGH**: Metrics endpoint accessible and scraping properly

---

## Test Environment Setup

### Required Infrastructure
- Kubernetes cluster v1.32.x
- cert-manager installed (for webhook certificates)
- kubectl configured
- helm 3.x installed

### Test Namespaces
- `kube-stager-test-fresh` - Fresh v1.0.0 installation
- `kube-stager-test-upgrade` - Existing v0.3.0 for upgrade testing
- `kube-stager-test-config` - Config file testing
- `kube-stager-test-failure` - Failure scenario testing

### Test Data Requirements
- Sample ServiceConfig CRs (MySQL, Mongo, Redis)
- Sample StagingSite CRs with various configurations
- Test database environments (MySQL, MongoDB, Redis)
- Sample config files (valid, invalid, partial)

---

## Test Scenarios

### 1. Config File Loading

#### TC-CONFIG-001: Valid Config File with All Fields (P0)
**Priority**: P0 - BLOCKING
**Test Type**: Automated + Manual
**Description**: Load a complete configuration file with all possible fields populated

**Prerequisites**:
- Clean Kubernetes namespace
- Config file with all fields defined

**Test Data** (`test-config-complete.yaml`):
```yaml
apiVersion: controller-config.operator.kube-stager.io/v1
kind: ProjectConfig
health:
  healthProbeBindAddress: ":8085"
metrics:
  bindAddress: "127.0.0.1:8082"
webhook:
  port: 9445
leaderElection: true
cacheNamespace: "test-namespace"
sentryDsn: "https://test@sentry.io/123456"
initJobConfig:
  deadlineSeconds: 900
  ttlSeconds: 1200
  backoffLimit: 0
migrationJobConfig:
  deadlineSeconds: 1200
  ttlSeconds: 1800
  backoffLimit: 5
backupJobConfig:
  deadlineSeconds: 1500
  ttlSeconds: 2400
  backoffLimit: 3
```

**Steps**:
1. Create ConfigMap with complete config file
2. Deploy kube-stager with `--config=/config/controller_manager_config.yaml`
3. Wait for operator pod to reach Ready state
4. Check operator logs for "Loading configuration from file" message
5. Check operator logs for "Using config" with all values applied
6. Verify health endpoint responds on port 8085: `curl http://pod-ip:8085/healthz`
7. Verify metrics endpoint on port 8082: `curl http://127.0.0.1:8082/metrics` (via port-forward)
8. Verify webhook server listening on port 9445
9. Create a test StagingSite and verify webhook is called
10. Verify leader election is active (check logs for election messages)

**Expected Results**:
- Pod starts successfully
- All config values logged correctly
- Health endpoint returns 200 OK on port 8085
- Metrics endpoint returns prometheus metrics on port 8082
- Webhook server accepts connections on port 9445
- Leader election messages appear in logs
- StagingSite webhook processes requests successfully

**Actual Results**: _______________

**Status**: [ ] Pass [ ] Fail [ ] Blocked

**Notes**: _______________

---

#### TC-CONFIG-002: Config File with Partial Fields (Defaults) (P0)
**Priority**: P0 - BLOCKING
**Test Type**: Automated
**Description**: Load config with only some fields, verify defaults apply

**Test Data** (`test-config-partial.yaml`):
```yaml
apiVersion: controller-config.operator.kube-stager.io/v1
kind: ProjectConfig
health:
  healthProbeBindAddress: ":8090"
initJobConfig:
  backoffLimit: 1
```

**Steps**:
1. Create ConfigMap with partial config file
2. Deploy kube-stager with config flag
3. Verify health endpoint on custom port 8090
4. Verify metrics endpoint still on default :8080
5. Verify webhook port still on default 9443
6. Create DbInitJob and verify it uses backoffLimit: 1
7. Create DbMigrationJob and verify it uses default backoffLimit: 3
8. Check that default values are applied for unspecified fields

**Expected Results**:
- Health probe on port 8090
- Metrics on default :8080
- Webhook on default 9443
- InitJobConfig.BackoffLimit = 1
- MigrationJobConfig uses defaults (600, 600, 3)
- BackupJobConfig uses defaults (600, 600, 3)
- LeaderElection defaults to false

**Actual Results**: _______________

**Status**: [ ] Pass [ ] Fail [ ] Blocked

---

#### TC-CONFIG-003: Missing Config File (P0)
**Priority**: P0 - BLOCKING
**Test Type**: Automated
**Description**: Operator runs with default config when no file specified

**Steps**:
1. Deploy operator WITHOUT --config flag
2. Verify pod starts successfully
3. Check logs - should NOT see "Loading configuration from file"
4. Verify health endpoint on default :8081
5. Verify metrics endpoint on default :8080
6. Verify webhook on default 9443
7. Verify leader election is false

**Expected Results**:
- Pod starts with default configuration
- Health probe: :8081
- Metrics: :8080
- Webhook: 9443
- No config file loading messages in logs

**Actual Results**: _______________

**Status**: [ ] Pass [ ] Fail [ ] Blocked

---

#### TC-CONFIG-004: Malformed YAML (P1)
**Priority**: P1 - HIGH
**Test Type**: Automated
**Description**: Operator fails gracefully with clear error for invalid YAML

**Test Data** (`test-config-malformed.yaml`):
```yaml
apiVersion: controller-config.operator.kube-stager.io/v1
kind: ProjectConfig
health:
  healthProbeBindAddress: ":8081"
  invalid_indentation
metrics:
```

**Steps**:
1. Create ConfigMap with malformed YAML
2. Deploy operator with config flag
3. Check pod status
4. Check operator logs

**Expected Results**:
- Pod fails to start or enters CrashLoopBackOff
- Logs contain error: "unable to parse config file"
- Error message is clear and actionable
- Exit code 1

**Actual Results**: _______________

**Status**: [ ] Pass [ ] Fail [ ] Blocked

---

#### TC-CONFIG-005: Invalid Values (P1)
**Priority**: P1 - HIGH
**Test Type**: Automated
**Description**: Test config with invalid/out-of-range values

**Test Data** (`test-config-invalid.yaml`):
```yaml
apiVersion: controller-config.operator.kube-stager.io/v1
kind: ProjectConfig
health:
  healthProbeBindAddress: "invalid-address"
webhook:
  port: 99999
initJobConfig:
  deadlineSeconds: -100
  backoffLimit: -5
```

**Steps**:
1. Create ConfigMap with invalid values
2. Deploy operator with config flag
3. Observe behavior

**Expected Results**:
- Operator either:
  - Rejects invalid config and fails to start with clear error, OR
  - Accepts config but fails when trying to bind to invalid address/port
- Error messages clearly indicate which field is invalid

**Actual Results**: _______________

**Status**: [ ] Pass [ ] Fail [ ] Blocked

---

#### TC-CONFIG-006: File Permissions Errors (P2)
**Priority**: P2 - MEDIUM
**Test Type**: Manual
**Description**: Test behavior when config file is not readable

**Steps**:
1. Create ConfigMap with config file
2. Mount config volume with wrong permissions (e.g., mode 0000)
3. Deploy operator
4. Check logs

**Expected Results**:
- Pod fails to start
- Log contains: "unable to read config file"
- Permission denied error message
- Exit code 1

**Actual Results**: _______________

**Status**: [ ] Pass [ ] Fail [ ] Blocked

---

### 2. Backward Compatibility

#### TC-COMPAT-001: Upgrade from v0.3.0 to v1.0.0 (P0)
**Priority**: P0 - BLOCKING
**Test Type**: Manual
**Description**: In-place upgrade without service disruption

**Prerequisites**:
- Running v0.3.0 helm deployment
- Active StagingSite CRs
- Active ServiceConfig, MongoConfig, MysqlConfig, RedisConfig CRs
- Active deployments created by operator

**Steps**:
1. Deploy v0.3.0 helm chart
2. Create sample ServiceConfig CR
3. Create sample StagingSite CR
4. Wait for StagingSite to reach Complete state
5. Verify deployments are running
6. Record current state: `kubectl get stagingsite,serviceconfig,deployment`
7. Upgrade helm release to v1.0.0: `helm upgrade kube-stager ./charts/kube-stager`
8. Monitor operator pod restart
9. Wait for new operator pod to become Ready
10. Verify existing StagingSite still shows Complete state
11. Verify existing deployments still running
12. Make a small change to StagingSite (e.g., imageTag)
13. Verify change is reconciled successfully
14. Create new StagingSite to test fresh creation

**Expected Results**:
- Upgrade completes without errors
- Existing StagingSite CRs remain in Complete state
- Existing deployments continue running
- No downtime for managed applications
- Modifications to existing resources are reconciled
- New resources can be created successfully
- Webhooks continue to function

**Actual Results**: _______________

**Status**: [ ] Pass [ ] Fail [ ] Blocked

---

#### TC-COMPAT-002: Existing ProjectConfig CRs (P0)
**Priority**: P0 - BLOCKING
**Test Type**: Automated
**Description**: Verify ProjectConfig CRD is compatible and functional

**Steps**:
1. Check CRD definition in v1.0.0
2. Verify ProjectConfig CRD is installed: `kubectl get crd projectconfigs.controller-config.operator.kube-stager.io`
3. Create ProjectConfig CR (even though operator uses file-based config)
4. Verify CR is accepted
5. Check if operator reads from CR (should not, uses file now)

**Expected Results**:
- ProjectConfig CRD exists and is valid
- ProjectConfig CRs can be created
- No errors in operator logs about CRD
- Operator uses file-based config, not CR

**Actual Results**: _______________

**Status**: [ ] Pass [ ] Fail [ ] Blocked

---

#### TC-COMPAT-003: Existing Webhooks Continue Working (P0)
**Priority**: P0 - BLOCKING
**Test Type**: Automated + Manual
**Description**: All webhook handlers remain functional after upgrade

**Webhooks to Test**:
1. ServiceConfig validation (create/update)
2. ServiceConfig deletion validation
3. MongoConfig deletion validation
4. MysqlConfig deletion validation
5. RedisConfig deletion validation
6. StagingSite mutation (advanced)
7. StagingSite defaulting (standard)
8. Backup mutation (advanced)

**Steps**:
1. Upgrade to v1.0.0
2. For each webhook:
   - Attempt to create/update/delete the resource
   - Verify webhook is called
   - Verify expected mutation/validation occurs
3. Test ServiceConfig creation with invalid shortName (should be rejected)
4. Test ServiceConfig deletion when in use by StagingSite (should be rejected)
5. Test StagingSite creation - verify defaults are applied
6. Test Backup creation - verify mutations occur

**Expected Results**:
- All webhooks respond successfully
- Validation webhooks reject invalid resources
- Mutation webhooks apply expected changes
- No "webhook unavailable" errors
- Webhook certificates are valid

**Actual Results**: _______________

**Status**: [ ] Pass [ ] Fail [ ] Blocked

---

#### TC-COMPAT-004: API Version Migration (P0)
**Priority**: P0 - BLOCKING
**Test Type**: Manual
**Description**: Verify K8s API migrations from v0.25 to v0.32 work correctly

**API Changes to Verify**:
- controller-runtime v0.20.4
- k8s.io/api v0.32.1
- k8s.io/apimachinery v0.32.1
- k8s.io/client-go v0.32.1

**Steps**:
1. Review code for deprecated API usage
2. Check for batch/v1beta1 CronJob usage (deprecated in v1.21+)
3. Check for networking.k8s.io/v1beta1 Ingress (removed in v1.22+)
4. Check for rbac.authorization.k8s.io/v1beta1 (removed in v1.22+)
5. Run operator against v1.32 cluster
6. Create resources using all CRDs
7. Verify no deprecation warnings in logs

**Expected Results**:
- No usage of deprecated APIs
- All resources use current API versions
- No warnings about deprecated APIs
- Full functionality on K8s v1.32

**Actual Results**: _______________

**Status**: [ ] Pass [ ] Fail [ ] Blocked

---

### 3. New Functionality

#### TC-NEW-001: Sidecar Containers with restartPolicy Always (P1)
**Priority**: P1 - HIGH
**Test Type**: Automated
**Description**: Verify sidecar containers can use restartPolicy: Always (K8s 1.29+ feature)

**Prerequisites**:
- Kubernetes cluster v1.29+
- ServiceConfig with sidecar container defined

**Test Data**:
```yaml
apiVersion: config.operator.kube-stager.io/v1
kind: ServiceConfig
metadata:
  name: test-with-sidecar
spec:
  shortName: sidecar
  deploymentPodSpec:
    containers:
    - name: main
      image: nginx:latest
      ports:
      - containerPort: 80
    initContainers:
    - name: sidecar-container
      image: busybox:latest
      command: ["sh", "-c", "while true; do echo alive; sleep 30; done"]
      restartPolicy: Always
```

**Steps**:
1. Create ServiceConfig with sidecar container
2. Create StagingSite referencing the ServiceConfig
3. Verify deployment is created
4. Check pod spec for sidecar with restartPolicy: Always
5. Verify pod starts successfully
6. Kill the sidecar container process
7. Verify sidecar restarts automatically
8. Kill main container process
9. Verify entire pod restarts (standard behavior)

**Expected Results**:
- Deployment created with sidecar container
- restartPolicy: Always is present in pod spec
- Sidecar runs continuously
- Sidecar restarts independently when killed
- Main container restart triggers pod restart

**Actual Results**: _______________

**Status**: [ ] Pass [ ] Fail [ ] Blocked

---

#### TC-NEW-002: Updated Webhook API (CustomDefaulter) (P0)
**Priority**: P0 - BLOCKING
**Test Type**: Automated
**Description**: Verify new webhook API using admission.CustomDefaulter interface

**Code Reference**: `/Users/szeber/dev/clients/szeber/kube-stager/apis/site/v1/stagingsite_webhook.go`

**Steps**:
1. Review webhook code - verify StagingSiteDefaulter implements admission.CustomDefaulter
2. Create StagingSite without optional fields
3. Retrieve created StagingSite
4. Verify defaults were applied by webhook:
   - DomainPrefix defaults to name
   - DbName defaults to sanitized name
   - Username defaults to sanitized dbName
   - Password defaults to random generated value
   - DisableAfter defaults to 2 days
   - DeleteAfter defaults to 7 days
5. Check webhook annotations on object

**Expected Results**:
- Webhook implements admission.CustomDefaulter correctly
- All default values are applied
- No errors in webhook logs
- Annotation "stagingsite-last-spec-change-at" is set

**Actual Results**: _______________

**Status**: [ ] Pass [ ] Fail [ ] Blocked

---

#### TC-NEW-003: Config Field Application - Health Port (P0)
**Priority**: P0 - BLOCKING
**Test Type**: Automated
**Description**: Verify health probe port configuration is applied

**Steps**:
1. Deploy operator with health.healthProbeBindAddress: ":8090"
2. Wait for pod to be Ready
3. Get pod IP
4. Test health endpoint: `curl http://<pod-ip>:8090/healthz`
5. Verify response is 200 OK
6. Test on default port 8081 - should fail or not respond

**Expected Results**:
- Health probe responds on port 8090
- Response body: "ok"
- HTTP 200 status
- Default port 8081 is not used

**Actual Results**: _______________

**Status**: [ ] Pass [ ] Fail [ ] Blocked

---

#### TC-NEW-004: Config Field Application - Metrics Port (P0)
**Priority**: P0 - BLOCKING
**Test Type**: Automated
**Description**: Verify metrics endpoint port configuration is applied

**Steps**:
1. Deploy operator with metrics.bindAddress: "127.0.0.1:8085"
2. Wait for pod to be Ready
3. Port-forward to pod: `kubectl port-forward <pod> 8085:8085`
4. Curl metrics endpoint: `curl http://127.0.0.1:8085/metrics`
5. Verify prometheus metrics are returned
6. Check for kube-stager specific metrics

**Expected Results**:
- Metrics endpoint responds on configured port
- Valid prometheus format metrics
- Includes controller-runtime metrics
- Includes custom kube-stager metrics if any

**Actual Results**: _______________

**Status**: [ ] Pass [ ] Fail [ ] Blocked

---

#### TC-NEW-005: Config Field Application - Webhook Port (P0)
**Priority**: P0 - BLOCKING
**Test Type**: Manual
**Description**: Verify webhook server port configuration is applied

**Steps**:
1. Deploy operator with webhook.port: 9445
2. Wait for pod to be Ready
3. Check pod logs for webhook server startup
4. Verify MutatingWebhookConfiguration has correct port in clientConfig
5. Create StagingSite to trigger webhook
6. Verify webhook is called on port 9445
7. Check webhook service definition

**Expected Results**:
- Webhook server listens on port 9445
- MutatingWebhookConfiguration uses port 9445
- ValidatingWebhookConfiguration uses port 9445
- Webhooks process requests successfully
- No certificate errors

**Actual Results**: _______________

**Status**: [ ] Pass [ ] Fail [ ] Blocked

---

### 4. Integration Scenarios

#### TC-INT-001: Full Operator Deployment with Config (P0)
**Priority**: P0 - BLOCKING
**Test Type**: End-to-End Manual
**Description**: Complete deployment flow using helm chart with custom config

**Steps**:
1. Create custom values.yaml:
```yaml
config:
  health:
    healthProbeBindAddress: ":8081"
  metrics:
    bindAddress: "127.0.0.1:8080"
  webhook:
    port: 9443
  leaderElection: true
  initJobConfig:
    deadlineSeconds: 900
    ttlSeconds: 1200
    backoffLimit: 0
  migrationJobConfig:
    deadlineSeconds: 1200
    ttlSeconds: 1800
    backoffLimit: 5
  backupJobConfig:
    deadlineSeconds: 1500
    ttlSeconds: 2400
    backoffLimit: 3
```
2. Install chart: `helm install kube-stager ./charts/kube-stager -f custom-values.yaml`
3. Verify all resources created (deployment, service, webhook configs, RBAC)
4. Wait for operator pod Ready
5. Create test infrastructure:
   - MongoConfig
   - MysqlConfig
   - RedisConfig
   - ServiceConfig
6. Create StagingSite
7. Monitor reconciliation to completion
8. Verify all resources created by operator:
   - Secrets (db credentials)
   - ConfigMaps
   - Services
   - Ingress
   - Deployments
   - Jobs (init, migration)

**Expected Results**:
- All helm resources deployed successfully
- Operator pod reaches Ready state
- Config file mounted correctly
- All config values applied
- StagingSite reaches Complete state
- All managed resources created correctly
- Application pods running

**Actual Results**: _______________

**Status**: [ ] Pass [ ] Fail [ ] Blocked

---

#### TC-INT-002: Webhook Mutations and Validations (P0)
**Priority**: P0 - BLOCKING
**Test Type**: Automated + Manual
**Description**: Comprehensive webhook integration testing

**Test Cases**:

**A. ServiceConfig Validation (Create/Update)**
1. Create valid ServiceConfig - should succeed
2. Create ServiceConfig with invalid shortName (>9 chars) - should fail
3. Create ServiceConfig with shortName containing uppercase - should fail
4. Update existing ServiceConfig - should succeed

**B. ServiceConfig Deletion Validation**
1. Create ServiceConfig
2. Create StagingSite using it
3. Try to delete ServiceConfig - should fail (in use)
4. Delete StagingSite
5. Try to delete ServiceConfig - should succeed

**C. Database Config Deletion Validation**
1. Create MongoConfig/MysqlConfig/RedisConfig
2. Create StagingSite using them
3. Try to delete config - should fail (in use)
4. Delete StagingSite
5. Try to delete config - should succeed

**D. StagingSite Advanced Mutation**
1. Create StagingSite without services specified
2. Verify webhook adds necessary mutations
3. Check annotations added by webhook

**E. Backup Mutation**
1. Create Backup CR
2. Verify webhook applies mutations
3. Verify job is created with correct spec

**Expected Results**:
- All validation webhooks enforce rules correctly
- Invalid resources are rejected with clear error messages
- Mutation webhooks apply expected changes
- No webhook timeout errors
- Certificate validation succeeds

**Actual Results**: _______________

**Status**: [ ] Pass [ ] Fail [ ] Blocked

---

#### TC-INT-003: Job Controller Config (P0)
**Priority**: P0 - BLOCKING
**Test Type**: Automated
**Description**: Verify job controllers use config values correctly

**Steps**:
1. Deploy operator with custom job config:
```yaml
initJobConfig:
  deadlineSeconds: 120
  ttlSeconds: 300
  backoffLimit: 0
migrationJobConfig:
  deadlineSeconds: 180
  ttlSeconds: 600
  backoffLimit: 5
backupJobConfig:
  deadlineSeconds: 240
  ttlSeconds: 900
  backoffLimit: 2
```
2. Create ServiceConfig with dbInitPodSpec
3. Create StagingSite
4. Wait for DbInitJob to be created
5. Check DbInitJob spec:
   - activeDeadlineSeconds: 120
   - ttlSecondsAfterFinished: 300
   - backoffLimit: 0
6. Verify job runs and completes
7. After 300 seconds, verify job is cleaned up
8. Create ServiceConfig with migrationJobPodSpec
9. Create StagingSite
10. Verify DbMigrationJob uses migration config values
11. Trigger backup creation
12. Verify Backup job uses backup config values

**Expected Results**:
- DbInitJob uses initJobConfig values
- DbMigrationJob uses migrationJobConfig values
- Backup uses backupJobConfig values
- Jobs complete within deadline
- Jobs are cleaned up after TTL
- Backoff limits are respected

**Actual Results**: _______________

**Status**: [ ] Pass [ ] Fail [ ] Blocked

---

#### TC-INT-004: Leader Election Behavior (P1)
**Priority**: P1 - HIGH
**Test Type**: Manual
**Description**: Test leader election when enabled

**Steps**:
1. Deploy operator with leaderElection: true and replicas: 3
2. Wait for all pods to be Ready
3. Check logs of all pods
4. Identify which pod is leader
5. Check for leader election messages
6. Delete the leader pod
7. Verify new leader is elected
8. Verify no reconciliation happens on non-leader pods
9. Verify reconciliation continues on leader

**Expected Results**:
- Only one pod becomes leader
- Leader election logs visible
- Non-leader pods wait
- When leader dies, new leader elected quickly
- No split-brain scenarios
- Reconciliation only on leader

**Actual Results**: _______________

**Status**: [ ] Pass [ ] Fail [ ] Blocked

---

#### TC-INT-005: Metrics Endpoint (P1)
**Priority**: P1 - HIGH
**Test Type**: Automated
**Description**: Verify metrics endpoint and scraping

**Steps**:
1. Deploy operator with metrics endpoint configured
2. Port-forward to metrics port
3. Curl /metrics endpoint
4. Verify prometheus format
5. Check for standard controller-runtime metrics:
   - controller_runtime_reconcile_total
   - controller_runtime_reconcile_errors_total
   - workqueue_depth
   - rest_client_requests_total
6. If custom metrics exist, verify them
7. Configure prometheus to scrape metrics
8. Verify metrics appear in prometheus

**Expected Results**:
- /metrics endpoint responds
- Valid prometheus format
- Standard controller metrics present
- Metrics update as operator reconciles
- Prometheus can scrape successfully

**Actual Results**: _______________

**Status**: [ ] Pass [ ] Fail [ ] Blocked

---

#### TC-INT-006: Health Probe Endpoint (P0)
**Priority**: P0 - BLOCKING
**Test Type**: Automated
**Description**: Verify health and readiness probes

**Steps**:
1. Deploy operator
2. Check deployment spec for liveness and readiness probes
3. Verify probes are configured correctly
4. Test /healthz endpoint - should return 200 OK
5. Test /readyz endpoint - should return 200 OK
6. Simulate unhealthy state (if possible)
7. Verify probe detects unhealthy state
8. Verify Kubernetes restarts unhealthy pod

**Expected Results**:
- /healthz returns 200 OK with body "ok"
- /readyz returns 200 OK with body "ok"
- Probes are called by kubelet
- Unhealthy state detected
- Pod restarted when unhealthy

**Actual Results**: _______________

**Status**: [ ] Pass [ ] Fail [ ] Blocked

---

### 5. Failure Scenarios

#### TC-FAIL-001: Invalid Config Values Recovery (P1)
**Priority**: P1 - HIGH
**Test Type**: Manual
**Description**: Test operator behavior with invalid config values

**Scenarios**:
A. Invalid port numbers (negative, >65535)
B. Invalid bind addresses
C. Negative deadline/ttl/backoff values
D. Extremely large values

**Steps**:
1. Deploy operator with invalid config
2. Observe startup behavior
3. Check logs for errors
4. Verify pod status
5. Correct config
6. Verify recovery

**Expected Results**:
- Operator fails to start with clear error
- Error message indicates which field is invalid
- No partial startup state
- After fixing config, operator starts successfully

**Actual Results**: _______________

**Status**: [ ] Pass [ ] Fail [ ] Blocked

---

#### TC-FAIL-002: Resource Limits (P2)
**Priority**: P2 - MEDIUM
**Test Type**: Manual
**Description**: Test operator under resource constraints

**Steps**:
1. Deploy operator with very low resource limits:
   - CPU: 10m
   - Memory: 32Mi
2. Create multiple StagingSites rapidly
3. Monitor operator behavior
4. Check for OOMKilled events
5. Check for CPU throttling

**Expected Results**:
- Operator handles resource pressure gracefully
- May be slower but continues functioning
- No data corruption
- Clear metrics showing resource usage
- May need to increase limits for production

**Actual Results**: _______________

**Status**: [ ] Pass [ ] Fail [ ] Blocked

---

#### TC-FAIL-003: Network Failures (P2)
**Priority**: P2 - MEDIUM
**Test Type**: Manual
**Description**: Test operator behavior during network issues

**Scenarios**:
A. Cannot reach Kubernetes API server
B. Webhook endpoint unreachable
C. Database endpoints unreachable (for job init)

**Steps**:
1. Deploy operator and StagingSite
2. Block network traffic to API server using network policy
3. Observe operator behavior
4. Restore network
5. Verify operator recovers
6. Check for reconciliation backoff

**Expected Results**:
- Operator retries API calls with backoff
- No crash loops
- After network recovery, reconciliation resumes
- No data loss or corruption

**Actual Results**: _______________

**Status**: [ ] Pass [ ] Fail [ ] Blocked

---

#### TC-FAIL-004: CRD Validation Failures (P1)
**Priority**: P1 - HIGH
**Test Type**: Automated
**Description**: Test CRD validation at API level

**Test Cases**:
1. Create ServiceConfig with shortName > 9 characters
2. Create ServiceConfig with invalid shortName pattern
3. Create StagingSite with invalid dbName pattern
4. Create StagingSite with replicas > 3
5. Create invalid TimeInterval values

**Expected Results**:
- API server rejects invalid resources before reaching operator
- Clear validation error messages
- Error indicates which field failed validation

**Actual Results**: _______________

**Status**: [ ] Pass [ ] Fail [ ] Blocked

---

#### TC-FAIL-005: Webhook Certificate Expiry (P2)
**Priority**: P2 - MEDIUM
**Test Type**: Manual
**Description**: Test behavior when webhook certificates expire

**Steps**:
1. Deploy operator with cert-manager
2. Verify webhook certificates are created
3. Check certificate expiry time
4. Simulate certificate expiry (or wait for it)
5. Verify cert-manager renews certificate
6. Verify webhooks continue working after renewal

**Expected Results**:
- Certificates are auto-renewed by cert-manager
- No webhook downtime during renewal
- Old resources continue working
- New resources can be created

**Actual Results**: _______________

**Status**: [ ] Pass [ ] Fail [ ] Blocked

---

#### TC-FAIL-006: Database Connection Failures (P1)
**Priority**: P1 - HIGH
**Test Type**: Manual
**Description**: Test job failures when database is unreachable

**Steps**:
1. Create ServiceConfig with dbInitPodSpec
2. Create MongoConfig and MysqlConfig with invalid connection strings
3. Create StagingSite
4. Verify DbInitJob is created
5. Verify job fails
6. Check job status and error messages
7. Verify backoff limit is respected
8. Fix database config
9. Delete and recreate StagingSite
10. Verify job succeeds

**Expected Results**:
- Job fails with clear error message
- Error indicates database connection failure
- Job retries according to backoffLimit
- After backoff limit, job marked as Failed
- StagingSite status shows error
- After fixing config, new StagingSite succeeds

**Actual Results**: _______________

**Status**: [ ] Pass [ ] Fail [ ] Blocked

---

### 6. Upgrade Path

#### TC-UPGRADE-001: In-place Upgrade from v0.3.0 (P0)
**Priority**: P0 - BLOCKING
**Test Type**: Manual
**Description**: Comprehensive upgrade testing from v0.3.0

**Prerequisites**:
- v0.3.0 helm chart deployed
- Multiple StagingSites in various states
- Active deployments and services
- Webhooks configured and functioning

**Pre-Upgrade State Capture**:
```bash
kubectl get stagingsite -o yaml > pre-upgrade-stagingsites.yaml
kubectl get serviceconfig -o yaml > pre-upgrade-serviceconfigs.yaml
kubectl get deployment -l operator.kube-stager.io/managed-by=kube-stager -o yaml > pre-upgrade-deployments.yaml
kubectl get svc -l operator.kube-stager.io/managed-by=kube-stager -o yaml > pre-upgrade-services.yaml
kubectl get ingress -l operator.kube-stager.io/managed-by=kube-stager -o yaml > pre-upgrade-ingresses.yaml
```

**Steps**:
1. Install v0.3.0 helm chart
2. Create test data:
   - 3 ServiceConfigs (MySQL, Mongo, Redis variants)
   - 5 StagingSites (different states: Pending, Complete, with different services)
   - Verify all reach Complete state
3. Capture pre-upgrade state (see above)
4. Run upgrade: `helm upgrade kube-stager ./charts/kube-stager --set imageTag=v1.0.0`
5. Monitor upgrade process
6. Wait for new operator pod to be Ready
7. Capture post-upgrade state
8. Compare states:
   - StagingSite specs unchanged
   - StagingSite statuses preserved
   - Deployments unchanged
   - Services unchanged
   - Ingresses unchanged
9. Wait for one reconciliation cycle
10. Verify no unexpected changes
11. Make minor change to one StagingSite (e.g., imageTag)
12. Verify change is reconciled correctly
13. Create new StagingSite
14. Verify creation succeeds
15. Delete one StagingSite
16. Verify cleanup succeeds

**Expected Results**:
- Upgrade completes in < 2 minutes
- Zero downtime for managed applications
- All existing StagingSites remain functional
- No status resets or data loss
- Reconciliation continues normally
- New resources can be created
- Deletions work correctly
- No webhook errors

**Actual Results**: _______________

**Status**: [ ] Pass [ ] Fail [ ] Blocked

---

#### TC-UPGRADE-002: Config Migration (P0)
**Priority**: P0 - BLOCKING
**Test Type**: Manual
**Description**: Migrate from v0.3.0 config to v1.0.0 file-based config

**Steps**:
1. Review v0.3.0 configuration method (if used)
2. Prepare v1.0.0 config file with equivalent settings
3. Upgrade helm chart with new config values
4. Verify ConfigMap is created with config file
5. Verify operator loads config file
6. Verify all config values are applied
7. Compare behavior before and after upgrade

**Expected Results**:
- Config file is created correctly by helm
- Operator loads config file
- All config values applied
- Behavior consistent with v0.3.0

**Actual Results**: _______________

**Status**: [ ] Pass [ ] Fail [ ] Blocked

---

#### TC-UPGRADE-003: CRD Updates (P0)
**Priority**: P0 - BLOCKING
**Test Type**: Automated
**Description**: Verify CRD updates during upgrade

**Steps**:
1. Capture v0.3.0 CRD definitions:
   - stagingsites.site.operator.kube-stager.io
   - serviceconfigs.config.operator.kube-stager.io
   - mongoconfigs.config.operator.kube-stager.io
   - mysqlconfigs.config.operator.kube-stager.io
   - redisconfigs.config.operator.kube-stager.io
   - All job CRDs
   - All task CRDs
   - projectconfigs.controller-config.operator.kube-stager.io
2. Upgrade to v1.0.0
3. Capture v1.0.0 CRD definitions
4. Compare CRDs for breaking changes
5. Verify existing CRs are still valid against new CRDs
6. Check for new fields in CRDs
7. Verify storage versions

**Expected Results**:
- CRDs update successfully
- No breaking changes to existing fields
- Existing CRs remain valid
- New fields added if any
- Storage version set correctly

**Actual Results**: _______________

**Status**: [ ] Pass [ ] Fail [ ] Blocked

---

#### TC-UPGRADE-004: Webhook Reconfiguration (P0)
**Priority**: P0 - BLOCKING
**Test Type**: Manual
**Description**: Verify webhook configurations are updated correctly

**Steps**:
1. Check v0.3.0 webhook configurations:
   - MutatingWebhookConfiguration
   - ValidatingWebhookConfiguration
2. Check webhook service
3. Check webhook certificates
4. Upgrade to v1.0.0
5. Verify webhook configurations updated
6. Verify webhook service unchanged (unless port changed)
7. Verify certificates still valid
8. Test each webhook endpoint
9. Verify no webhook errors in logs

**Expected Results**:
- Webhook configurations updated correctly
- Webhook service remains accessible
- Certificates valid and working
- All webhook endpoints functional
- No transition downtime

**Actual Results**: _______________

**Status**: [ ] Pass [ ] Fail [ ] Blocked

---

## Smoke Test Checklist

Quick validation for production deployment (15-30 minutes):

### Pre-Deployment Checks
- [ ] Backup existing v0.3.0 data
- [ ] Review release notes
- [ ] Review config file changes
- [ ] Verify helm chart version is 1.0.0
- [ ] Verify image tag is v1.0.0

### Deployment
- [ ] Helm upgrade completes without errors
- [ ] New operator pod reaches Ready state (< 2 min)
- [ ] Old operator pod terminates gracefully
- [ ] No CRD errors in logs
- [ ] Config file loaded successfully

### Functional Checks
- [ ] Health endpoint responds: `curl http://<pod-ip>:8081/healthz`
- [ ] Metrics endpoint responds: `curl http://127.0.0.1:8080/metrics`
- [ ] Webhook service is accessible
- [ ] Certificate is valid and not expired
- [ ] Existing StagingSites show Complete state
- [ ] Existing Deployments are running
- [ ] Create new test StagingSite - reaches Complete (< 5 min)
- [ ] Update existing StagingSite - reconciles successfully
- [ ] Delete test StagingSite - cleanup completes

### Webhook Validation
- [ ] Create ServiceConfig - succeeds
- [ ] Create ServiceConfig with invalid name - fails with validation error
- [ ] Create StagingSite - defaults applied
- [ ] Delete ServiceConfig in use - fails with validation error

### Critical Logs Review
- [ ] No ERRORS in operator logs
- [ ] No webhook failures
- [ ] No CRD errors
- [ ] Reconciliation loops completing
- [ ] Leader election working (if enabled)

### Performance Check
- [ ] Operator memory usage < 128Mi
- [ ] Operator CPU usage reasonable
- [ ] Reconciliation time < 2 minutes per StagingSite

---

## Rollback Test Scenarios

### TC-ROLLBACK-001: Rollback to v0.3.0 (P0)
**Priority**: P0 - BLOCKING
**Test Type**: Manual
**Description**: Test rollback procedure if v1.0.0 has critical issues

**Steps**:
1. Complete upgrade to v1.0.0
2. Identify critical issue requiring rollback
3. Capture current state
4. Run rollback: `helm rollback kube-stager`
5. Wait for v0.3.0 pod to be Ready
6. Verify existing resources still work
7. Check for any data loss

**Expected Results**:
- Rollback completes successfully
- v0.3.0 operator runs correctly
- Existing StagingSites continue working
- No data corruption
- May lose v1.0.0 specific features

**Actual Results**: _______________

**Status**: [ ] Pass [ ] Fail [ ] Blocked

---

### TC-ROLLBACK-002: Config File Removal During Rollback (P1)
**Priority**: P1 - HIGH
**Test Type**: Manual
**Description**: Test if v0.3.0 handles presence of config file

**Steps**:
1. Upgrade to v1.0.0 with config file
2. Rollback to v0.3.0
3. Check if config file causes issues
4. Verify v0.3.0 operator ignores config file
5. Verify operator uses default config

**Expected Results**:
- v0.3.0 ignores config file if present
- Operator starts successfully
- Uses default configuration
- No errors related to config file

**Actual Results**: _______________

**Status**: [ ] Pass [ ] Fail [ ] Blocked

---

## Test Execution Strategy

### Phase 1: Pre-Release Testing (Unit/Integration Tests)
**Automated Tests**:
- Run existing unit tests: `make test`
- Run controller tests if available
- Run webhook tests if available

### Phase 2: Staging Environment Testing
**Environment**: Non-production Kubernetes cluster

**Day 1-2**: Config File Testing
- Execute all TC-CONFIG-* test cases
- Verify config loading and application
- Test edge cases

**Day 3**: Backward Compatibility
- Deploy v0.3.0
- Execute all TC-COMPAT-* test cases
- Execute upgrade test TC-UPGRADE-001

**Day 4**: New Functionality
- Execute all TC-NEW-* test cases
- Verify sidecar support
- Verify webhook API updates

**Day 5**: Integration Testing
- Execute all TC-INT-* test cases
- End-to-end scenarios
- Performance testing

**Day 6**: Failure Scenarios
- Execute all TC-FAIL-* test cases
- Chaos testing
- Recovery testing

**Day 7**: Rollback Testing
- Execute TC-ROLLBACK-* test cases
- Document rollback procedure

### Phase 3: Production Validation
- Deploy to production during maintenance window
- Execute Smoke Test Checklist
- Monitor for 24-48 hours
- Execute rollback if critical issues

---

## Risk Assessment

### High Risk Areas

1. **Kubernetes API Migration (v0.25 → v0.32)**
   - Risk: Breaking API changes
   - Mitigation: Extensive compatibility testing
   - Test Cases: TC-COMPAT-004

2. **Config File Loading**
   - Risk: Malformed config crashes operator
   - Mitigation: Graceful error handling
   - Test Cases: TC-CONFIG-001 through TC-CONFIG-006

3. **Webhook Functionality**
   - Risk: Webhooks break, preventing resource creation
   - Mitigation: Thorough webhook testing
   - Test Cases: TC-COMPAT-003, TC-INT-002

4. **In-Place Upgrade**
   - Risk: Data loss or service disruption
   - Mitigation: Comprehensive upgrade testing
   - Test Cases: TC-UPGRADE-001

### Medium Risk Areas

1. **Leader Election**
   - Risk: Split-brain or no reconciliation
   - Mitigation: Test multi-replica scenarios
   - Test Cases: TC-INT-004

2. **Job Configuration**
   - Risk: Jobs timeout or never clean up
   - Mitigation: Test all job config combinations
   - Test Cases: TC-INT-003

3. **Sidecar Container Support**
   - Risk: Pods fail to start with sidecars
   - Mitigation: Test on K8s 1.29+
   - Test Cases: TC-NEW-001

### Low Risk Areas

1. **Metrics Endpoint**
   - Risk: Metrics not available
   - Impact: Low (monitoring affected but not functionality)
   - Test Cases: TC-INT-005

2. **Sentry Integration**
   - Risk: Error reporting doesn't work
   - Impact: Low (debugging harder but no functional impact)
   - Test Cases: Part of TC-INT-001

---

## Test Metrics and Reporting

### Test Coverage Goals
- P0 Tests: 100% pass rate required
- P1 Tests: 95% pass rate required
- P2 Tests: 90% pass rate acceptable
- P3 Tests: Best effort

### Exit Criteria for v1.0.0 Release
1. All P0 tests passing
2. All P1 tests passing or have documented workarounds
3. No critical bugs found
4. Upgrade path validated
5. Rollback procedure documented and tested
6. Smoke test checklist validated in staging
7. Performance acceptable (no regression from v0.3.0)

### Test Report Template
```
Test Execution Summary
======================
Date: _______________
Tester: _______________
Environment: _______________

Test Results:
- P0 Tests: __ Passed, __ Failed, __ Blocked
- P1 Tests: __ Passed, __ Failed, __ Blocked
- P2 Tests: __ Passed, __ Failed, __ Blocked
- P3 Tests: __ Passed, __ Failed, __ Blocked

Critical Issues:
1. _______________
2. _______________

Blockers:
1. _______________

Recommendation: [ ] Approve Release [ ] Block Release

Notes:
_______________
```

---

## Appendix A: Test Data Files

### Sample ServiceConfig
```yaml
apiVersion: config.operator.kube-stager.io/v1
kind: ServiceConfig
metadata:
  name: test-service
  namespace: default
spec:
  shortName: "testsvc"
  deploymentPodSpec:
    containers:
    - name: nginx
      image: nginx:latest
      ports:
      - containerPort: 80
  serviceSpec:
    ports:
    - port: 80
      targetPort: 80
```

### Sample StagingSite
```yaml
apiVersion: site.operator.kube-stager.io/v1
kind: StagingSite
metadata:
  name: test-site
  namespace: default
spec:
  enabled: true
  services:
    testsvc:
      imageTag: "latest"
      replicas: 1
```

---

## Appendix B: Key Files Reference

- Main entry point: `/Users/szeber/dev/clients/szeber/kube-stager/main.go`
- Config types: `/Users/szeber/dev/clients/szeber/kube-stager/apis/controller-config/v1/projectconfig_types.go`
- Webhook implementation: `/Users/szeber/dev/clients/szeber/kube-stager/apis/site/v1/stagingsite_webhook.go`
- Helm chart: `/Users/szeber/dev/clients/szeber/kube-stager-helm/charts/kube-stager/`
- Helm values: `/Users/szeber/dev/clients/szeber/kube-stager-helm/charts/kube-stager/values.yaml`

---

## Appendix C: Automation Recommendations

### High Priority for Automation
1. **TC-CONFIG-001, 002, 003**: Config file loading tests
2. **TC-COMPAT-003**: Webhook validation tests
3. **TC-NEW-002**: CustomDefaulter webhook tests
4. **TC-INT-003**: Job controller config tests
5. **TC-FAIL-004**: CRD validation tests

### Medium Priority for Automation
1. **TC-NEW-003, 004, 005**: Config field application tests
2. **TC-INT-005, 006**: Metrics and health probe tests
3. **TC-UPGRADE-003**: CRD update tests

### Manual Testing Required
1. **TC-UPGRADE-001**: Full upgrade path (too complex for automation)
2. **TC-INT-004**: Leader election (requires multi-pod scenarios)
3. **TC-FAIL-002, 003**: Resource limits and network failures (require cluster-level changes)
4. **TC-ROLLBACK-001**: Rollback testing (requires human judgment)

### Suggested Test Framework
- **Unit Tests**: Existing Go test framework with testify
- **Integration Tests**: Kubernetes envtest (controller-runtime)
- **E2E Tests**: Ginkgo + Gomega (already present in project)
- **Webhook Tests**: kubebuilder webhook testing utilities

---

**END OF TEST PLAN**
