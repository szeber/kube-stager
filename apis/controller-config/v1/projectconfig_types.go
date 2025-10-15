/*
Copyright 2023.

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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

//+kubebuilder:object:root=true

// ProjectConfig is the Schema for the projectconfigs API
type ProjectConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Health contains the controller health configuration.
	//+optional
	Health HealthConfig `json:"health,omitempty"`

	// Metrics contains the controller metrics configuration.
	//+optional
	Metrics MetricsConfig `json:"metrics,omitempty"`

	// Webhook contains the controller webhook configuration.
	//+optional
	Webhook WebhookConfig `json:"webhook,omitempty"`

	// LeaderElection determines whether to use leader election when starting the manager.
	//+optional
	LeaderElection bool `json:"leaderElection,omitempty"`

	// CacheNamespace if specified restricts the manager's cache to watch objects in the desired namespace.
	//+optional
	CacheNamespace string `json:"cacheNamespace,omitempty"`

	// The DSN for Sentry if sentry is used to capture errors
	//+optional
	SentryDsn string `json:"sentryDsn,omitempty"`

	// The config for the init job. The backofflimit defaults to 0 since retrying a half way complete failed init can
	//cause unexpected results if the job is not prepared for this
	//+optional
	InitJobConfig JobConfig `json:"initJobConfig,omitempty"`

	// The config for the init job
	//+optional
	MigrationJobConfig JobConfig `json:"migrationJobConfig,omitempty"`

	// The config for the init job
	//+optional
	BackupJobConfig JobConfig `json:"backupJobConfig,omitempty"`
}

// HealthConfig contains the controller health configuration.
type HealthConfig struct {
	// HealthProbeBindAddress is the TCP address that the controller should bind to
	// for serving health probes
	//+optional
	HealthProbeBindAddress string `json:"healthProbeBindAddress,omitempty"`
}

// MetricsConfig contains the controller metrics configuration.
type MetricsConfig struct {
	// BindAddress is the TCP address that the controller should bind to
	// for serving prometheus metrics
	//+optional
	BindAddress string `json:"bindAddress,omitempty"`
}

// WebhookConfig contains the controller webhook configuration.
type WebhookConfig struct {
	// Port is the port that the webhook server serves at.
	//+optional
	Port int `json:"port,omitempty"`
}

type JobConfig struct {
	// The deadline seconds for the completion of the job - it will fail if it's not complete in this amount of time
	//+kubebuilder:default:=600
	//+optional
	DeadlineSeconds int32 `json:"deadlineSeconds,omitempty"`

	// The TTL seconds for the job - The job will be cleaned up after this time
	//+kubebuilder:default:=600
	//+optional
	TtlSeconds int32 `json:"ttlSeconds,omitempty"`

	// The backofflimit for the job. The job is allowed to fail this many times
	//+kubebuilder:default:=3
	//+optional
	BackoffLimit int32 `json:"backoffLimit,omitempty"`
}

func init() {
	SchemeBuilder.Register(&ProjectConfig{})
}
