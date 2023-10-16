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
	sitev1 "github.com/szeber/kube-stager/api/site/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ImportExportDataSpec defines the desired state of ImportExportData
type ImportExportDataSpec struct {
	// The spec of the site
	SiteSpec sitev1.StagingSiteSpec `json:"siteSpec"`

	SiteStatus sitev1.StagingSiteStatus `json:"siteStatus"`
}

// ImportExportDataStatus defines the observed state of ImportExportData
type ImportExportDataStatus struct {
	// Whether the import has completed (or was never an import)
	//+optional
	IsImportComplete bool `json:"isImportComplete,omitempty"`
	// The timestamp when the import was started at
	//+optional
	ImportStartedAt *metav1.Time `json:"importStartedAt,omitempty"`
}

func (r ImportExportDataStatus) NeedsProcessing(startedAtThresholdTime time.Time) bool {
	return !r.IsImportComplete && (nil == r.ImportStartedAt || r.ImportStartedAt.Time.Before(startedAtThresholdTime))
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ImportExportData is the Schema for the importexportdata API
type ImportExportData struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ImportExportDataSpec   `json:"spec,omitempty"`
	Status ImportExportDataStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ImportExportDataList contains a list of ImportExportData
type ImportExportDataList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ImportExportData `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ImportExportData{}, &ImportExportDataList{})
}
