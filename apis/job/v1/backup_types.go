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

// BackupSpec defines the desired state of Backup
type BackupSpec struct {
	//+kubebuilder:validate:MinLength=1
	// Name of the site owning this job
	SiteName string `json:"siteName"`

	//+kubebuilder:default:=Manual
	// Type of the backup. Scheduled backups should happen daily if the backup is enabled for a site, Final backup should happen just before deleting the databases
	//+optional
	BackupType BackupType `json:"backupType"`
}

// BackupStatus defines the observed state of Backup
type BackupStatus struct {
	BackupStatusDetail `json:",inline"`

	// Service level statuses
	Services map[string]BackupStatusDetail `json:"services,omitempty"`
}

type BackupStatusDetail struct {
	//+kubebuilder:default:=Pending
	// State of the job
	State JobState `json:"state"`

	// Time the backup job was started at
	//+optional
	JobStartedAt *metav1.Time `json:"jobStartedAt,omitempty"`

	// Time the backup job successfully completed at
	//+optional
	JobFinishedAt *metav1.Time `json:"jobFinishedAt,omitempty"`
}

// +kubebuilder:validation:Enum=Manual;Scheduled;Final
type BackupType string

const (
	BackupTypeManual    BackupType = "Manual"
	BackupTypeScheduled BackupType = "Scheduled"
	BackupTypeFinal     BackupType = "Final"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Site",type=string,JSONPath=`.spec.siteName`
//+kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.spec.backupType`
//+kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`
//+kubebuilder:printcolumn:name="Started",type=date,JSONPath=`.status.jobStartedAt`
//+kubebuilder:printcolumn:name="Finished",type=date,JSONPath=`.status.jobFinishedAt`

// Backup is the Schema for the backups API
type Backup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BackupSpec   `json:"spec,omitempty"`
	Status BackupStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// BackupList contains a list of Backup
type BackupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Backup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Backup{}, &BackupList{})
}
