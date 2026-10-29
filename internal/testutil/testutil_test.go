package testutil

import (
	"os"
	"testing"

	"k8s.io/client-go/rest"
)

func TestSafeTestMainSetsIsolatingEnvironment(t *testing.T) {
	t.Setenv("KUBERNETES_SERVICE_HOST", "10.0.0.1")

	SafeTestMain(t)

	if got := os.Getenv("KUBECONFIG"); got != UnresolvableKubeconfigPath {
		t.Errorf("KUBECONFIG = %q, want %q", got, UnresolvableKubeconfigPath)
	}
	if got := os.Getenv("USE_EXISTING_CLUSTER"); got != "false" {
		t.Errorf("USE_EXISTING_CLUSTER = %q, want false", got)
	}
	if got := os.Getenv("KUBERNETES_SERVICE_HOST"); got != "" {
		t.Errorf("KUBERNETES_SERVICE_HOST = %q, want empty", got)
	}
}

func TestRequireLocalAPIServer(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *rest.Config
		wantErr bool
	}{
		{"nil config", nil, true},
		{"localhost", &rest.Config{Host: "https://localhost:6443"}, false},
		{"127.0.0.1", &rest.Config{Host: "https://127.0.0.1:6443"}, false},
		{"other loopback address", &rest.Config{Host: "https://127.0.0.2:6443"}, false},
		{"ipv6 loopback", &rest.Config{Host: "https://[::1]:6443"}, false},
		{"bare 127.0.0.1", &rest.Config{Host: "127.0.0.1:6443"}, false},
		{"remote host", &rest.Config{Host: "https://10.0.0.1:6443"}, true},
		{"remote hostname", &rest.Config{Host: "https://my-cluster.example.com:6443"}, true},
		{"hostname starting with localhost", &rest.Config{Host: "https://localhost.example.com:6443"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RequireLocalAPIServer(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("RequireLocalAPIServer() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
