package testutil

import (
	"sync"

	configv1 "github.com/szeber/kube-stager/apis/config/v1"
	jobv1 "github.com/szeber/kube-stager/apis/job/v1"
	sitev1 "github.com/szeber/kube-stager/apis/site/v1"
	taskv1 "github.com/szeber/kube-stager/apis/task/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
)

var (
	testSchemeOnce sync.Once
	testScheme     *runtime.Scheme
)

// NewTestScheme returns a runtime.Scheme with all custom API groups and core k8s types registered.
// The scheme is created once and cached for subsequent calls.
func NewTestScheme() *runtime.Scheme {
	testSchemeOnce.Do(func() {
		testScheme = runtime.NewScheme()
		utilruntime.Must(clientgoscheme.AddToScheme(testScheme))
		utilruntime.Must(configv1.AddToScheme(testScheme))
		utilruntime.Must(jobv1.AddToScheme(testScheme))
		utilruntime.Must(sitev1.AddToScheme(testScheme))
		utilruntime.Must(taskv1.AddToScheme(testScheme))
	})

	return testScheme
}
