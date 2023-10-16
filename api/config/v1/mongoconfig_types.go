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

// MongoConfigSpec defines the desired state of MongoConfig
type MongoConfigSpec struct {
	//+kubebuilder:validation:MinLength=1
	// The primary hostname of this mongo config
	Host1 string `json:"host1"`

	//+kubebuilder:validation:MinLength=1
	// The secondary hostname of this mongo config
	//+optional
	Host2 string `json:"host2,omitempty"`

	//+kubebuilder:validation:MinLength=0
	// The tertiary hostname of this mongo config
	//+optional
	Host3 string `json:"host3,omitempty"`

	//+kubebuilder:validation:MinLength=0
	// The admin username for the server
	Username string `json:"username"`

	//+kubebuilder:validation:MinLength=1
	// The password for the server
	Password string `json:"password"`

	//+kubebuilder:default:=27017
	// The port for the server - defaults to 27017
	//+optional
	Port uint16 `json:"port,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:printcolumn:name="Host1",type=string,JSONPath=`.spec.host1`
//+kubebuilder:printcolumn:name="Host2",type=string,JSONPath=`.spec.host2`
//+kubebuilder:printcolumn:name="Host3",type=string,JSONPath=`.spec.host3`
//+kubebuilder:printcolumn:name="Port",type=string,JSONPath=`.spec.port`
//+kubebuilder:printcolumn:name="Username",type=string,JSONPath=`.spec.username`

// MongoConfig is the Schema for the mongoconfigs API
type MongoConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec MongoConfigSpec `json:"spec,omitempty"`
}

//+kubebuilder:object:root=true

// MongoConfigList contains a list of MongoConfig
type MongoConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MongoConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MongoConfig{}, &MongoConfigList{})
}
