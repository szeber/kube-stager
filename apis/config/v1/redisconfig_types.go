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

// RedisConfigSpec defines the desired state of RedisConfig
type RedisConfigSpec struct {
	//+kubebuilder:validation:MinLength=1
	// The hostname of this mysql config
	Host string `json:"host"`

	//+kubebuilder:default:=16
	// The number of available databases on this redis server. Defaults to 16.
	//+optional
	AvailableDatabaseCount uint32 `json:"availableDatabaseCount,omitempty"`

	//+kubebuilder:default:=6379
	// The port for the server - defaults to 6379
	//+optional
	Port uint16 `json:"port,omitempty"`

	//+kubebuilder:default:=false
	// Whether TLS is enabled on the server
	IsTlsEnabled *bool `json:"isTlsEnabled,omitempty"`

	// Whether to verify the server's certificate
	//+optional
	//+kubebuilder:default:=true
	VerifyTlsServerCertificate *bool `json:"verifyTlsServerCertificate,omitempty"`

	// The password to connect to the server
	//+optional
	Password string `json:"password,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:printcolumn:name="Host",type=string,JSONPath=`.spec.host`
//+kubebuilder:printcolumn:name="Port",type=string,JSONPath=`.spec.port`
//+kubebuilder:printcolumn:name="Available-Database-Count",type=integer,JSONPath=`.spec.availableDatabaseCount`
//+kubebuilder:printcolumn:name="Is-TLS-Enabled",type=boolean,JSONPath=`.spec.isTlsEnabled`

// RedisConfig is the Schema for the redisconfigs API
type RedisConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec RedisConfigSpec `json:"spec,omitempty"`
}

//+kubebuilder:object:root=true

// RedisConfigList contains a list of RedisConfig
type RedisConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RedisConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RedisConfig{}, &RedisConfigList{})
}
