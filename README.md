# kube-stager

A Kubernetes operator for managing staging and test environments with automated database provisioning and lifecycle management.

## Description

kube-stager is a Kubernetes operator that automates the creation and management of staging/test sites. It handles:
- Automatic database provisioning (MySQL, MongoDB, Redis)
- Database initialization and migration jobs
- Backup creation and restoration
- Resource lifecycle management
- Service configuration and validation

The operator is designed for teams that need to quickly spin up isolated test environments with their own databases and configurations.

## Requirements

- Kubernetes 1.29+ (for sidecar container support)
- Go 1.23+ (for development)
- Helm 3+ (for deployment)

## Getting Started

The recommended way to install kube-stager is via Helm chart. See [kube-stager-helm](https://github.com/szeber/kube-stager-helm) for installation instructions.

### Configuration

The operator supports configuration via a ConfigMap. Key settings include:

- `leaderElection`: Enable/disable leader election (default: true for v1.0.0+)
- `sentryDsn`: Optional Sentry DSN for error tracking
- `initJobConfig`, `migrationJobConfig`, `backupJobConfig`: Job timeout and retry settings

For Redis databases with TLS:
- Set `isTlsEnabled: true` in RedisConfig
- Optionally set `verifyTlsServerCertificate: false` for self-signed certificates

All configuration values are validated at startup. Invalid configurations will cause the operator to exit with a descriptive error message.

### Running on the cluster
1. Install Instances of Custom Resources:

```sh
kubectl apply -f config/samples/
```

2. Build and push your image to the location specified by `IMG`:
	
```sh
make docker-build docker-push IMG=ghcr.io/szeber/kube-stager:tag
```
	
3. Deploy the controller to the cluster with the image specified by `IMG`:

```sh
make deploy IMG=ghcr.io/szeber/kube-stager:tag
```

### Uninstall CRDs
To delete the CRDs from the cluster:

```sh
make uninstall
```

### Undeploy controller
UnDeploy the controller to the cluster:

```sh
make undeploy
```

## Contributing
// TODO(user): Add detailed information on how you would like others to contribute to this project

### How it works
This project aims to follow the Kubernetes [Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)

It uses [Controllers](https://kubernetes.io/docs/concepts/architecture/controller/) 
which provides a reconcile function responsible for synchronizing resources untile the desired state is reached on the cluster 

### Test It Out
1. Install the CRDs into the cluster:

```sh
make install
```

2. Run your controller (this will run in the foreground, so switch to a new terminal if you want to leave it running):

```sh
make run
```

**NOTE:** You can also run this in one step by running: `make install run`

### Modifying the API definitions
If you are editing the API definitions, generate the manifests such as CRs or CRDs using:

```sh
make manifests
```

**NOTE:** Run `make --help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## Monitoring

Optional Prometheus integration is available via ServiceMonitor (requires Prometheus Operator):
- Set `monitoring.enabled: true` in Helm values
- Metrics exposed on port 8443 via kube-rbac-proxy
- Default scrape interval: 30s

## Version History

See [CHANGELOG.md](CHANGELOG.md) for detailed version history.

## License

Copyright 2023-2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

