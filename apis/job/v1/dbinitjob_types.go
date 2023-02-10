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

// DbInitJobSpec defines the desired state of DbInitJob
type DbInitJobSpec struct {
	//+kubebuilder:validate:MinLength=1
	// Name of the site owning this job
	SiteName string `json:"siteName"`

	//+kubebuilder:validate:MinLength=1
	// Name of the service.
	ServiceName string `json:"serviceName"`

	//+kubebuilder:validate:MinLength=0
	// Name of the mysql environment to initialise
	//+optional
	MysqlEnvironment string `json:"mysqlEnvironment"`

	//+kubebuilder:validate:MinLength=0
	// Name of the mongo environment to initialise
	//+optional
	MongoEnvironment string `json:"mongoEnvironment"`

	//+kubebuilder:validate:MinLength=1
	// Name of the staging site used to initialise the db
	DbInitSource string `json:"dbInitSource"`

	//+kubebuilder:validate:MinLength=1
	//+kubebuilder:validation:MaxLength=63
	// Name of the database to initialise
	DatabaseName string `json:"databaseName"`

	//+kubebuilder:validate:MinLength=1
	//+kubebuilder:validation:MaxLength=16
	// Name of the user used to connect to the databases
	Username string `json:"username"`

	//+kubebuilder:validate:MinLength=1
	//+kubebuilder:validation:MaxLength=32
	// Password for the user used to connect to the databases
	Password string `json:"password"`

	// The number of seconds to use as the completion deadline
	DeadlineSeconds int64 `json:"deadlineSeconds"`
}

// DbInitJobStatus defines the observed state of DbInitJob
type DbInitJobStatus struct {
	//+kubebuilder:default:=Pending
	// State of the job
	State JobState `json:"state"`

	//+kubebuilder:default:=0
	// Number of consecutive times the related batch job failed to load
	JobNotFoundCount uint32 `json:"jobNotFoundCount"`

	// The deadline for the job's completion, after which the job will be marked as failed if it didn't run to completion yet
	DeadlineTimestamp *metav1.Time `json:"deadlineTimestamp"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Site",type=string,JSONPath=`.spec.siteName`
//+kubebuilder:printcolumn:name="Service",type=string,JSONPath=`.spec.serviceName`
//+kubebuilder:printcolumn:name="Init-Source",type=string,JSONPath=`.spec.dbInitSource`
//+kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`

// DbInitJob is the Schema for the dbinitjobs API
type DbInitJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DbInitJobSpec   `json:"spec,omitempty"`
	Status DbInitJobStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DbInitJobList contains a list of DbInitJob
type DbInitJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DbInitJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DbInitJob{}, &DbInitJobList{})
}
