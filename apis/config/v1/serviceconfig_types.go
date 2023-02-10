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
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ServiceConfigSpec defines the desired state of ServiceConfig
type ServiceConfigSpec struct {
	//+kubebuilder:validation:MinLength:=1
	//+kubebuilder:validation:MaxLength:=9
	//+kubebuilder:validation:Pattern:=[a-z][-0-9a-z]*
	// A short identifier for the service. Must be all lowercase and match dns name rules unique in the namespace, ideal length is about 3 characters. This will be used to
	// suffix the created kube objects
	ShortName string `json:"shortName"`

	// The data for any configmaps to create
	//+optional
	ConfigMaps map[string]Configmap `json:"configMaps,omitempty"`

	// Any additional custom template values. May be overridden in the site config
	//+optional
	CustomTemplateValues map[string]string `json:"customTemplateValues"`

	// The spec for the deployment to create
	DeploymentPodSpec corev1.PodSpec `json:"deploymentPodSpec"`

	// The spec for the db init job. If not set, no db initialisation will be run
	//+optional
	DbInitPodSpec *corev1.PodSpec `json:"dbInitPodSpec"`

	// The spec for the migration job. If not set, no db migration will be run
	//+optional
	MigrationJobPodSpec *corev1.PodSpec `json:"migrationJobPodSpec"`

	// The spec for the backup job. If not set, no backup will be run
	//+optional
	BackupPodSpec *corev1.PodSpec `json:"backupPodSpec,omitempty"`

	// The spec for the service created for the deployment of this service. If not set, no service will be created
	//+optional
	ServiceSpec *corev1.ServiceSpec `json:"serviceSpec,omitempty"`

	// The spec for the ingress for this service. If not set, no ingress will be created
	//+optional
	IngressSpec *networkingv1.IngressSpec `json:"ingressSpec,omitempty"`

	// Annotations for the ingress object
	//+optional
	IngressAnnotations map[string]string `json:"ingressAnnotations"`

	// Name of the default mongo environment if one is not specified on the site level
	//+optional
	DefaultMongoEnvironment string `json:"defaultMongoEnvironment"`

	// Name of the default mysql environment if one is not specified on the site level
	//+optional
	DefaultMysqlEnvironment string `json:"defaultMysqlEnvironment"`

	// Name of the default redis environment if one is not specified on the site level
	//+optional
	DefaultRedisEnvironment string `json:"defaultRedisEnvironment"`
}

type Configmap map[string]string

//+kubebuilder:object:root=true
//+kubebuilder:printcolumn:name="Short-Name",type=string,JSONPath=`.spec.shortName`
//+kubebuilder:printcolumn:name="Mysql",type=string,JSONPath=`.spec.defaultMongoEnvironment`
//+kubebuilder:printcolumn:name="Mongo",type=string,JSONPath=`.spec.defaultMysqlEnvironment`
//+kubebuilder:printcolumn:name="Redis",type=string,JSONPath=`.spec.defaultRedisEnvironment`

// ServiceConfig is the Schema for the serviceconfigs API
type ServiceConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ServiceConfigSpec `json:"spec,omitempty"`
}

//+kubebuilder:object:root=true

// ServiceConfigList contains a list of ServiceConfig
type ServiceConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ServiceConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ServiceConfig{}, &ServiceConfigList{})
}
