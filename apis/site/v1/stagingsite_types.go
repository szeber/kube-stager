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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// StagingSiteSpec defines the desired state of StagingSite
type StagingSiteSpec struct {
	//+kubebuilder:validation:MinLength=0
	// The domain prefix for the staging environment. Defaults to the name of the staging site
	//+optional
	DomainPrefix string `json:"domainPrefix,omitempty"`

	//+kubebuilder:validation:Pattern=[_a-zA-Z0-9]*
	//+kubebuilder:validation:MinLength=0
	//+kubebuilder:validation:MaxLength=63
	// The name of the databases to create (applies both to mysql and mongo). Defaults to the name of the staging site
	//+optional
	DbName string `json:"dbName,omitempty"`

	//+kubebuilder:validation:Pattern=[_a-zA-Z0-9]*
	//+kubebuilder:validation:MinLength=0
	//+kubebuilder:validation:MaxLength=16
	// The username to use for authentication (applies both to mysql and mongo). Defaults to the db name
	//+optional
	Username string `json:"username,omitempty"`

	//+kubebuilder:validation:Pattern=[_a-zA-Z0-9]*
	//+kubebuilder:validation:MinLength=0
	//+kubebuilder:validation:MaxLength=32
	// The password to use for authentication (applies both to mysql and mongo). Defaults to a randomly generated password
	//+optional
	Password string `json:"password,omitempty"`

	//+kubebuilder:default:=true
	// Whether to enable the staging site (run it's pods). Defaults to TRUE
	//+optional
	Enabled bool `json:"enabled"`

	// The period after which the staging site will be automatically disabled (data left intact). Defaults to 2 days
	//+optional
	DisableAfter TimeInterval `json:"disableAfter,omitempty"`

	// The period after which the staging site will be automatically deleted (including it's database). Defaults to 7 days
	//+optional
	DeleteAfter TimeInterval `json:"deleteAfter,omitempty"`

	//+kubebuilder:default:=false
	// Whether to perform a database backup before deleting the site. Defaults to FALSE
	//+optional
	BackupBeforeDelete bool `json:"backupBeforeDelete,omitempty"`

	//+kubebuilder:validation:Min=-1
	//+kubebuilder:validation:Max=23
	// The hour for the daily backup window in UTC 24 hour time (0-23).
	//+optional
	DailyBackupWindowHour *int32 `json:"dailyBackupWindowHour,omitempty"`

	// The services used by the staging site
	//+optional
	Services map[string]StagingSiteService `json:"services,omitempty"`

	//+kubebuilder:default:=false
	// If set to TRUE, all services will be included and deployed with this staging site. Defaults to FALSE
	//+optional
	IncludeAllServices bool `json:"includeAllServices,omitempty"`
}

type StagingSiteService struct {
	//+kubebuilder:default:=latest
	// Tag of the image to deploy the file from. Will default to latest if not set.
	//+optional
	ImageTag string `json:"imageTag"`

	//+kubebuilder:validation:Min=1
	//+kubebuilder:validation:Max=3
	//+kubebuilder:default:1
	// The replica count for this services deployment
	//+optional
	Replicas int32 `json:"replicas"`

	// Override resource requirements for the containers in the service pods
	//+optional
	ResourceOverrides map[string]corev1.ResourceRequirements `json:"resourceOverrides,omitempty"`

	//+kubebuilder:validation:MinLength=1
	// Name of the mysql environment to use for this service
	MysqlEnvironment string `json:"mysqlEnvironment,omitempty"`

	//+kubebuilder:validation:MinLength=1
	// Name of the mongodb environment to use for this service
	MongoEnvironment string `json:"mongoEnvironment,omitempty"`

	//+kubebuilder:validation:MinLength=1
	// Name of the redis environment to use for this service
	RedisEnvironment string `json:"redisEnvironment,omitempty"`

	//+kubebuilder:default:=false
	// Whether to include the service in backups. Defaults to FALSE
	//+optional
	IncludeInBackups bool `json:"includeInBackups,omitempty"`

	//+kubebuilder:validation:MinLength=0
	// The name of the environment to initialise the database from. Defaults to "master"
	//+optional
	DbInitSourceEnvironmentName string `json:"dumpSourceEnvironmentName,omitempty"`

	// Any extra environment variables to set for the staging site.
	//+optional
	ExtraEnvs map[string]string `json:"extraEnvs,omitempty"`

	// Any additional custom template value overrides
	//+optional
	CustomTemplateValues map[string]string `json:"customTemplateValues,omitempty"`
}

type TimeInterval struct {
	// If TRUE the time range will never apply
	Never bool `json:"never,omitempty"`
	// Number of days. All amounts are additive, so 1 day 25 hours 90 minutes == 2 days 2 hour 30 minutes
	Days int `json:"days,omitempty"`
	// Number of hours. All amounts are additive, so 1 day 25 hours 90 minutes == 2 days 2 hour 30 minutes
	Hours int `json:"hours,omitempty"`
	// Number of minutes. All amounts are additive, so 1 day 25 hours 90 minutes == 2 days 2 hour 30 minutes
	Minutes int `json:"minutes,omitempty"`
}

// StagingSiteStatus defines the observed state of StagingSite
type StagingSiteStatus struct {
	// Whether the database creation is complete
	DatabaseCreationComplete bool `json:"databaseCreationComplete"`

	// Whether the database initialisation is complete
	DatabaseInitialisationComplete bool `json:"databaseInitialisationComplete"`

	// Whether the database migrations have finished running everywhere
	DatabaseMigrationsComplete bool `json:"databaseMigrationsComplete"`

	// Whether configuration type objects are created/updated (configmaps, secrets)
	ConfigsAreCreated bool `json:"configsAreCreated"`

	// Whether networking type objects are created/updated (services, ingresses)
	NetworkingObjectsAreCreated bool `json:"networkingObjectsAreCreated"`

	// Whether the workload objects are created and up to date (deployments)
	WorkloadsAreCreated bool `json:"workloadsAreCreated"`

	// The timestamp of the last applied configuration
	LastAppliedConfiguration *metav1.Time `json:"lastAppliedConfiguration,omitempty"`

	// The timestamp when the site will be automatically disabled at
	//+optional
	DisableAt *metav1.Time `json:"disableAt,omitempty"`

	// The timestamp when the site will be automatically deleted at
	//+optional
	DeleteAt *metav1.Time `json:"deleteAt,omitempty"`

	// Whether the site is enabled or not. The automatic disabling sets this flag to false, but doesn't touch the spec one
	Enabled bool `json:"enabled"`

	// The global state of the site
	State StagingSiteState `json:"state"`

	// The combined health of the workloads related to this instance
	WorkloadHealth WorkloadHealth `json:"workloadHealth"`

	// The error message associated with the Failed status
	ErrorMessage string `json:"errorMessage"`

	// The status for the services
	Services map[string]StagingSiteServiceStatus `json:"services,omitempty"`

	// The time of the latest successful backup of the whole site
	//+optional
	LastBackupTime *metav1.Time `json:"lastBackupTime,omitempty"`

	// The time the next backup is scheduled for
	//+optional
	NextBackupTime *metav1.Time `json:"nextBackupTime,omitempty"`
}

type StagingSiteServiceStatus struct {
	// The username to use for database connections
	Username string `json:"username,omitempty"`

	// The database name to use for database connections
	DbName string `json:"databaseName"`

	// The database number to use for redis connections
	RedisDatabaseNumber uint32 `json:"redisDatabaseNumber"`

	// The status subentity of the created deployment
	DeploymentStatus appsv1.DeploymentStatus `json:"deploymentStatus,omitempty"`
}

// +kubebuilder:validation:Enum=Pending;Complete;Failed
type StagingSiteState string

// +kubebuilder:validation:Enum=Healthy;Unhealthy;Incomplete
type WorkloadHealth string

const (
	StatePending             StagingSiteState = "Pending"
	StateComplete            StagingSiteState = "Complete"
	StateFailed              StagingSiteState = "Failed"
	WorkloadHealthHealthy    WorkloadHealth   = "Healthy"
	WorkloadHealthUnhealthy  WorkloadHealth   = "Unhealthy"
	WorkloadHealthIncomplete WorkloadHealth   = "Incomplete"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion
//+kubebuilder:printcolumn:name="DB",type=string,JSONPath=`.spec.dbName`
//+kubebuilder:printcolumn:name="Username",type=string,JSONPath=`.spec.username`
//+kubebuilder:printcolumn:name="Init-Source",type=string,JSONPath=`.spec.dumpSourceEnvironmentName`
//+kubebuilder:printcolumn:name="Enabled",type=boolean,JSONPath=`.spec.enabled`
//+kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`
//+kubebuilder:printcolumn:name="Workload-Health",type=string,JSONPath=`.status.workloadHealth`
//+kubebuilder:printcolumn:name="Next-Backup",type=string,JSONPath=`.status.nextBackupTime`
//+kubebuilder:printcolumn:name="Last-Successful-Backup",type=date,JSONPath=`.status.lastBackupTime`

// StagingSite is the Schema for the stagingsites API
type StagingSite struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StagingSiteSpec   `json:"spec,omitempty"`
	Status StagingSiteStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// StagingSiteList contains a list of StagingSite
type StagingSiteList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []StagingSite `json:"items"`
}

func init() {
	SchemeBuilder.Register(&StagingSite{}, &StagingSiteList{})
}
