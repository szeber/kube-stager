package testutil

import "testing"

// SafeTestMain prevents accidental connections to real clusters when running
// envtest-based tests, as a defense-in-depth alongside UseExistingCluster: &false.
func SafeTestMain(t *testing.T) {
	t.Setenv("USE_EXISTING_CLUSTER", "false")
	t.Setenv("KUBECONFIG", "")
	t.Setenv("KUBERNETES_SERVICE_HOST", "")
}
