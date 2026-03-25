package v1

import (
	"testing"
	"time"

	configv1 "github.com/szeber/kube-stager/apis/config/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetServiceStatus(t *testing.T) {
	site := &StagingSite{
		Status: StagingSiteStatus{
			Services: map[string]StagingSiteServiceStatus{
				"web": {Username: "user1", DbName: "db1"},
			},
		},
	}

	t.Run("existing service returns pointer", func(t *testing.T) {
		got := site.GetServiceStatus("web")
		if got == nil {
			t.Fatal("expected non-nil")
		}
		if got.Username != "user1" {
			t.Errorf("Username = %q, want %q", got.Username, "user1")
		}
	})

	t.Run("missing service returns nil", func(t *testing.T) {
		got := site.GetServiceStatus("missing")
		if got != nil {
			t.Errorf("expected nil, got %+v", got)
		}
	})
}

func TestGetMongoConfigForService(t *testing.T) {
	t.Run("site override takes precedence", func(t *testing.T) {
		site := StagingSite{
			Spec: StagingSiteSpec{
				Services: map[string]StagingSiteService{
					"web": {MongoEnvironment: "site-mongo"},
				},
			},
		}
		config := configv1.ServiceConfig{
			ObjectMeta: metav1.ObjectMeta{Name: "web"},
			Spec:       configv1.ServiceConfigSpec{DefaultMongoEnvironment: "default-mongo"},
		}
		got := site.GetMongoConfigForService(config)
		if got != "site-mongo" {
			t.Errorf("got %q, want %q", got, "site-mongo")
		}
	})

	t.Run("falls back to default", func(t *testing.T) {
		site := StagingSite{
			Spec: StagingSiteSpec{
				Services: map[string]StagingSiteService{
					"web": {},
				},
			},
		}
		config := configv1.ServiceConfig{
			ObjectMeta: metav1.ObjectMeta{Name: "web"},
			Spec:       configv1.ServiceConfigSpec{DefaultMongoEnvironment: "default-mongo"},
		}
		got := site.GetMongoConfigForService(config)
		if got != "default-mongo" {
			t.Errorf("got %q, want %q", got, "default-mongo")
		}
	})
}

func TestGetMysqlConfigForService(t *testing.T) {
	t.Run("site override takes precedence", func(t *testing.T) {
		site := StagingSite{
			Spec: StagingSiteSpec{
				Services: map[string]StagingSiteService{
					"web": {MysqlEnvironment: "site-mysql"},
				},
			},
		}
		config := configv1.ServiceConfig{
			ObjectMeta: metav1.ObjectMeta{Name: "web"},
			Spec:       configv1.ServiceConfigSpec{DefaultMysqlEnvironment: "default-mysql"},
		}
		if got := site.GetMysqlConfigForService(config); got != "site-mysql" {
			t.Errorf("got %q, want %q", got, "site-mysql")
		}
	})

	t.Run("falls back to default", func(t *testing.T) {
		site := StagingSite{
			Spec: StagingSiteSpec{
				Services: map[string]StagingSiteService{"web": {}},
			},
		}
		config := configv1.ServiceConfig{
			ObjectMeta: metav1.ObjectMeta{Name: "web"},
			Spec:       configv1.ServiceConfigSpec{DefaultMysqlEnvironment: "default-mysql"},
		}
		if got := site.GetMysqlConfigForService(config); got != "default-mysql" {
			t.Errorf("got %q, want %q", got, "default-mysql")
		}
	})
}

func TestGetRedisConfigForService(t *testing.T) {
	t.Run("site override takes precedence", func(t *testing.T) {
		site := StagingSite{
			Spec: StagingSiteSpec{
				Services: map[string]StagingSiteService{
					"web": {RedisEnvironment: "site-redis"},
				},
			},
		}
		config := configv1.ServiceConfig{
			ObjectMeta: metav1.ObjectMeta{Name: "web"},
			Spec:       configv1.ServiceConfigSpec{DefaultRedisEnvironment: "default-redis"},
		}
		if got := site.GetRedisConfigForService(config); got != "site-redis" {
			t.Errorf("got %q, want %q", got, "site-redis")
		}
	})

	t.Run("falls back to default", func(t *testing.T) {
		site := StagingSite{
			Spec: StagingSiteSpec{
				Services: map[string]StagingSiteService{"web": {}},
			},
		}
		config := configv1.ServiceConfig{
			ObjectMeta: metav1.ObjectMeta{Name: "web"},
			Spec:       configv1.ServiceConfigSpec{DefaultRedisEnvironment: "default-redis"},
		}
		if got := site.GetRedisConfigForService(config); got != "default-redis" {
			t.Errorf("got %q, want %q", got, "default-redis")
		}
	})
}

func TestTimeInterval_ToDuration(t *testing.T) {
	tests := []struct {
		name     string
		interval TimeInterval
		expected time.Duration
	}{
		{"days only", TimeInterval{Days: 2}, 48 * time.Hour},
		{"hours only", TimeInterval{Hours: 3}, 3 * time.Hour},
		{"minutes only", TimeInterval{Minutes: 30}, 30 * time.Minute},
		{"combo", TimeInterval{Days: 1, Hours: 2, Minutes: 30}, 26*time.Hour + 30*time.Minute},
		{"zero", TimeInterval{}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.interval.ToDuration(); got != tt.expected {
				t.Errorf("ToDuration() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetDummySite(t *testing.T) {
	site := GetDummySite("myservice", "test-ns")
	if site.Name != "dummy" {
		t.Errorf("Name = %q, want %q", site.Name, "dummy")
	}
	if site.Namespace != "test-ns" {
		t.Errorf("Namespace = %q, want %q", site.Namespace, "test-ns")
	}
	if site.Spec.DomainPrefix != "dummy" {
		t.Errorf("DomainPrefix = %q, want %q", site.Spec.DomainPrefix, "dummy")
	}
	if !site.Spec.DisableAfter.Never {
		t.Error("DisableAfter.Never should be true")
	}
	if !site.Spec.DeleteAfter.Never {
		t.Error("DeleteAfter.Never should be true")
	}
	if _, ok := site.Spec.Services["myservice"]; !ok {
		t.Error("expected myservice in Services")
	}
	svc := site.Spec.Services["myservice"]
	if svc.ImageTag != "latest" {
		t.Errorf("ImageTag = %q, want %q", svc.ImageTag, "latest")
	}
	if svc.Replicas != 1 {
		t.Errorf("Replicas = %d, want 1", svc.Replicas)
	}
}
