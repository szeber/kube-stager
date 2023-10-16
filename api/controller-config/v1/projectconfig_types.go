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
	cfg "sigs.k8s.io/controller-runtime/pkg/config/v1alpha1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

//+kubebuilder:object:root=true

// ProjectConfig is the Schema for the projectconfigs API
type ProjectConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// ControllerManagerConfigurationSpec returns the contfigurations for controllers
	//+optional
	cfg.ControllerManagerConfigurationSpec `json:",inline"`

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
