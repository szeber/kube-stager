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

// DbMigrationJobSpec defines the desired state of DbMigrationJob
type DbMigrationJobSpec struct {
	//+kubebuilder:validate:MinLength=1
	// Name of the site owning this job
	SiteName string `json:"siteName"`

	//+kubebuilder:validate:MinLength=1
	// Name of the service.
	ServiceName string `json:"serviceName"`

	//+kubebuilder:validate:MinLength=1
	// The tag for the images to use
	ImageTag string `json:"imageTag"`

	// The number of seconds to use as the completion deadline
	DeadlineSeconds int64 `json:"deadlineSeconds"`
}

// DbMigrationJobStatus defines the observed state of DbMigrationJob
type DbMigrationJobStatus struct {
	//+kubebuilder:default:=Pending
	// State of the job
	State JobState `json:"state"`

	//+kubebuilder:default:=0
	// Number of consecutive times the related batch job failed to load
	JobNotFoundCount uint32 `json:"jobNotFoundCount"`

	// Name of the image that the last migration was executed
	LastMigratedImageTag string `json:"lastMigratedImageTag"`

	// The deadline for the job's completion, after which the job will be marked as failed if it didn't run to completion yet
	DeadlineTimestamp *metav1.Time `json:"deadlineTimestamp"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Site",type=string,JSONPath=`.spec.siteName`
//+kubebuilder:printcolumn:name="Service",type=string,JSONPath=`.spec.serviceName`
//+kubebuilder:printcolumn:name="Image-Tag",type=string,JSONPath=`.spec.imageTag`
//+kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`

// DbMigrationJob is the Schema for the dbmigrationjobs API
type DbMigrationJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DbMigrationJobSpec   `json:"spec,omitempty"`
	Status DbMigrationJobStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DbMigrationJobList contains a list of DbMigrationJob
type DbMigrationJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DbMigrationJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DbMigrationJob{}, &DbMigrationJobList{})
}
