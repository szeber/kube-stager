package testutil

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"testing"

	"k8s.io/client-go/rest"
)

const UnresolvableKubeconfigPath = "/dev/null/nonexistent"

// SafeTestMain prevents accidental connections to real clusters when running
// envtest-based tests, as a defense-in-depth alongside UseExistingCluster: &false.
func SafeTestMain(t *testing.T) {
	t.Setenv("USE_EXISTING_CLUSTER", "false")
	t.Setenv("KUBECONFIG", UnresolvableKubeconfigPath)
	t.Setenv("KUBERNETES_SERVICE_HOST", "")
}

func RequireLocalAPIServer(cfg *rest.Config) error {
	if cfg == nil {
		return errors.New("rest.Config is nil")
	}

	host := cfg.Host
	if !strings.Contains(host, "://") {
		host = "https://" + host
	}

	apiServerURL, err := url.Parse(host)
	if err != nil {
		return fmt.Errorf("failed to parse config host %q: %w", cfg.Host, err)
	}

	hostname := apiServerURL.Hostname()
	ip := net.ParseIP(hostname)
	if hostname == "localhost" || (ip != nil && ip.IsLoopback()) {
		return nil
	}

	return fmt.Errorf("test API server %q is not on a loopback address", cfg.Host)
}
