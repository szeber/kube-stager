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

// RedisDatabaseSpec defines the desired state of RedisDatabase
type RedisDatabaseSpec struct {
	EnvironmentConfig EnvironmentConfig `json:"environmentConfig"`

	//+kubebuilder:validation:Minimum=0
	// Name of the database
	DatabaseNumber uint32 `json:"databaseNumber"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Site",type=string,JSONPath=`.spec.environmentConfig.siteName`
//+kubebuilder:printcolumn:name="Service",type=string,JSONPath=`.spec.environmentConfig.serviceName`
//+kubebuilder:printcolumn:name="Environment",type=string,JSONPath=`.spec.environmentConfig.environment`
//+kubebuilder:printcolumn:name="Database",type=integer,JSONPath=`.spec.databaseNumber`
//+kubebuilder:printcolumn:name="Username",type=string,JSONPath=`.spec.username`
//+kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`

// RedisDatabase is the Schema for the redisdatabases API
type RedisDatabase struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RedisDatabaseSpec `json:"spec,omitempty"`
	Status TaskStatus        `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// RedisDatabaseList contains a list of RedisDatabase
type RedisDatabaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RedisDatabase `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RedisDatabase{}, &RedisDatabaseList{})
}
