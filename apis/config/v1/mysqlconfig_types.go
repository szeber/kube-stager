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

// MysqlConfigSpec defines the desired state of MysqlConfig
type MysqlConfigSpec struct {
	//+kubebuilder:validation:MinLength=1
	// The hostname of this mysql config
	Host string `json:"host"`

	//+kubebuilder:validation:MinLength=1
	// The admin username for the server
	Username string `json:"username"`

	//+kubebuilder:validation:MinLength=1
	// The password for the server
	Password string `json:"password"`

	//+kubebuilder:default:=3306
	// The port for the server - defaults to 3306
	//+optional
	Port uint16 `json:"port,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:printcolumn:name="Host",type=string,JSONPath=`.spec.host`
//+kubebuilder:printcolumn:name="Port",type=string,JSONPath=`.spec.port`
//+kubebuilder:printcolumn:name="Username",type=string,JSONPath=`.spec.username`

// MysqlConfig is the Schema for the mysqlconfigs API
type MysqlConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec MysqlConfigSpec `json:"spec,omitempty"`
}

//+kubebuilder:object:root=true

// MysqlConfigList contains a list of MysqlConfig
type MysqlConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MysqlConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MysqlConfig{}, &MysqlConfigList{})
}
