package testutil

import (
	configv1 "github.com/szeber/kube-stager/apis/config/v1"
	jobv1 "github.com/szeber/kube-stager/apis/job/v1"
	sitev1 "github.com/szeber/kube-stager/apis/site/v1"
	taskv1 "github.com/szeber/kube-stager/apis/task/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
)

// NewTestScheme creates a runtime.Scheme with all custom API groups and core k8s types registered.
func NewTestScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(s))
	utilruntime.Must(configv1.AddToScheme(s))
	utilruntime.Must(jobv1.AddToScheme(s))
	utilruntime.Must(sitev1.AddToScheme(s))
	utilruntime.Must(taskv1.AddToScheme(s))

	return s
}
