package pod

import (
	"testing"

	configv1 "github.com/szeber/kube-stager/apis/config/v1"
	sitev1 "github.com/szeber/kube-stager/apis/site/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func makeSiteAndConfig(serviceName string) (*sitev1.StagingSite, *configv1.ServiceConfig) {
	site := &sitev1.StagingSite{
		ObjectMeta: metav1.ObjectMeta{Name: "test-site"},
		Spec: sitev1.StagingSiteSpec{
			Services: map[string]sitev1.StagingSiteService{
				serviceName: {
					ExtraEnvs: map[string]string{
						"ENV_B": "val_b",
						"ENV_A": "val_a",
					},
					ResourceOverrides: map[string]corev1.ResourceRequirements{
						"nginx": {
							Limits: corev1.ResourceList{
								corev1.ResourceCPU: resource.MustParse("500m"),
							},
						},
					},
				},
			},
		},
	}
	config := &configv1.ServiceConfig{
		ObjectMeta: metav1.ObjectMeta{Name: serviceName},
	}
	return site, config
}

func TestSetExtraEnvVarsOnPodSpec(t *testing.T) {
	t.Run("adds env vars to all containers sorted alphabetically", func(t *testing.T) {
		site, config := makeSiteAndConfig("web")
		spec := corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "nginx", Env: []corev1.EnvVar{{Name: "EXISTING", Value: "val"}}},
				{Name: "sidecar"},
			},
		}
		got := SetExtraEnvVarsOnPodSpec(spec, site, config)
		for _, c := range got.Containers {
			if len(c.Env) < 2 {
				t.Errorf("container %s: expected at least 2 env vars, got %d", c.Name, len(c.Env))
				continue
			}
			// Check sorted
			for i := 1; i < len(c.Env); i++ {
				if c.Env[i-1].Name > c.Env[i].Name {
					t.Errorf("container %s: env vars not sorted: %s > %s", c.Name, c.Env[i-1].Name, c.Env[i].Name)
				}
			}
		}
	})

	t.Run("no-op when empty", func(t *testing.T) {
		site := &sitev1.StagingSite{
			Spec: sitev1.StagingSiteSpec{
				Services: map[string]sitev1.StagingSiteService{
					"web": {},
				},
			},
		}
		config := &configv1.ServiceConfig{ObjectMeta: metav1.ObjectMeta{Name: "web"}}
		spec := corev1.PodSpec{
			Containers: []corev1.Container{{Name: "nginx"}},
		}
		got := SetExtraEnvVarsOnPodSpec(spec, site, config)
		if len(got.Containers[0].Env) != 0 {
			t.Errorf("expected 0 env vars, got %d", len(got.Containers[0].Env))
		}
	})
}

func TestUpdatePodSpecWithOverrides(t *testing.T) {
	t.Run("overrides matching containers", func(t *testing.T) {
		site, config := makeSiteAndConfig("web")
		spec := corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "nginx"},
				{Name: "sidecar"},
			},
		}
		got := UpdatePodSpecWithOverrides(spec, site, config)
		if got.Containers[0].Resources.Limits.Cpu().String() != "500m" {
			t.Errorf("nginx CPU limit = %s, want 500m", got.Containers[0].Resources.Limits.Cpu().String())
		}
		if len(got.Containers[1].Resources.Limits) != 0 {
			t.Error("sidecar should not have resource overrides")
		}
	})

	t.Run("no-op when empty overrides", func(t *testing.T) {
		site := &sitev1.StagingSite{
			Spec: sitev1.StagingSiteSpec{
				Services: map[string]sitev1.StagingSiteService{
					"web": {},
				},
			},
		}
		config := &configv1.ServiceConfig{ObjectMeta: metav1.ObjectMeta{Name: "web"}}
		spec := corev1.PodSpec{
			Containers: []corev1.Container{{Name: "nginx"}},
		}
		got := UpdatePodSpecWithOverrides(spec, site, config)
		if len(got.Containers[0].Resources.Limits) != 0 {
			t.Error("should have no overrides")
		}
	})
}
