package v1

import (
	"testing"

	configv1 "github.com/szeber/kube-stager/apis/config/v1"
	sitev1 "github.com/szeber/kube-stager/apis/site/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func makeJobTestSiteAndConfig() (*sitev1.StagingSite, *configv1.ServiceConfig) {
	site := &sitev1.StagingSite{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-site",
			Namespace: "test-ns",
		},
		Spec: sitev1.StagingSiteSpec{
			DbName:   "testdb",
			Username: "testuser",
			Password: "testpass",
			Services: map[string]sitev1.StagingSiteService{
				"myservice": {
					ImageTag:                    "v1.0",
					DbInitSourceEnvironmentName: "master",
				},
			},
		},
	}
	config := &configv1.ServiceConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "myservice",
			Namespace: "test-ns",
		},
		Spec: configv1.ServiceConfigSpec{
			ShortName: "svc",
			MigrationJobPodSpec: &corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "migrate", Image: "migrate:latest"},
				},
			},
		},
	}
	return site, config
}

func TestDbInitJob_PopulateFomSite(t *testing.T) {
	t.Run("populates all fields", func(t *testing.T) {
		site, config := makeJobTestSiteAndConfig()
		job := &DbInitJob{}
		err := job.PopulateFomSite(site, config, "mysql-env", "mongo-env")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if job.Namespace != "test-ns" {
			t.Errorf("Namespace = %q, want %q", job.Namespace, "test-ns")
		}
		if job.Spec.SiteName != "test-site" {
			t.Errorf("SiteName = %q, want %q", job.Spec.SiteName, "test-site")
		}
		if job.Spec.MysqlEnvironment != "mysql-env" {
			t.Errorf("MysqlEnvironment = %q, want %q", job.Spec.MysqlEnvironment, "mysql-env")
		}
		if job.Spec.MongoEnvironment != "mongo-env" {
			t.Errorf("MongoEnvironment = %q, want %q", job.Spec.MongoEnvironment, "mongo-env")
		}
		if job.Spec.DbInitSource != "master" {
			t.Errorf("DbInitSource = %q, want %q", job.Spec.DbInitSource, "master")
		}
		if job.Spec.Password != "testpass" {
			t.Errorf("Password = %q, want %q", job.Spec.Password, "testpass")
		}
		if job.Spec.DeadlineSeconds != 600 {
			t.Errorf("DeadlineSeconds = %d, want 600", job.Spec.DeadlineSeconds)
		}
	})

	t.Run("service not in site returns error", func(t *testing.T) {
		site, _ := makeJobTestSiteAndConfig()
		config := &configv1.ServiceConfig{
			ObjectMeta: metav1.ObjectMeta{Name: "notinsite"},
			Spec:       configv1.ServiceConfigSpec{ShortName: "nope"},
		}
		job := &DbInitJob{}
		err := job.PopulateFomSite(site, config, "", "")
		if err == nil {
			t.Error("expected error for missing service in site")
		}
	})
}

func TestDbMigrationJob_PopulateFomSite(t *testing.T) {
	t.Run("populates all fields", func(t *testing.T) {
		site, config := makeJobTestSiteAndConfig()
		job := &DbMigrationJob{}
		err := job.PopulateFomSite(site, config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if job.Spec.ImageTag != "v1.0" {
			t.Errorf("ImageTag = %q, want %q", job.Spec.ImageTag, "v1.0")
		}
		if job.Spec.SiteName != "test-site" {
			t.Errorf("SiteName = %q, want %q", job.Spec.SiteName, "test-site")
		}
	})

	t.Run("nil MigrationJobPodSpec returns error", func(t *testing.T) {
		site, config := makeJobTestSiteAndConfig()
		config.Spec.MigrationJobPodSpec = nil
		job := &DbMigrationJob{}
		err := job.PopulateFomSite(site, config)
		if err == nil {
			t.Error("expected error for nil MigrationJobPodSpec")
		}
	})
}

func TestDbMigrationJob_Matches(t *testing.T) {
	t.Run("matching returns true", func(t *testing.T) {
		j1 := &DbMigrationJob{Spec: DbMigrationJobSpec{SiteName: "s", ServiceName: "svc", ImageTag: "v1"}}
		j2 := &DbMigrationJob{Spec: DbMigrationJobSpec{SiteName: "s", ServiceName: "svc", ImageTag: "v1"}}
		if !j1.Matches(j2) {
			t.Error("expected Matches() to return true")
		}
	})

	t.Run("different ImageTag returns false", func(t *testing.T) {
		j1 := &DbMigrationJob{Spec: DbMigrationJobSpec{SiteName: "s", ServiceName: "svc", ImageTag: "v1"}}
		j2 := &DbMigrationJob{Spec: DbMigrationJobSpec{SiteName: "s", ServiceName: "svc", ImageTag: "v2"}}
		if j1.Matches(j2) {
			t.Error("expected Matches() to return false")
		}
	})
}

func TestDbMigrationJob_UpdateFrom(t *testing.T) {
	j1 := &DbMigrationJob{Spec: DbMigrationJobSpec{SiteName: "s", ImageTag: "v1"}}
	j2 := &DbMigrationJob{Spec: DbMigrationJobSpec{SiteName: "s", ImageTag: "v2"}}
	j1.UpdateFrom(j2)
	if j1.Spec.ImageTag != "v2" {
		t.Errorf("ImageTag = %q, want %q", j1.Spec.ImageTag, "v2")
	}
}

func TestJobState_IsFinal(t *testing.T) {
	tests := []struct {
		state    JobState
		expected bool
	}{
		{Failed, true},
		{Complete, true},
		{Pending, false},
		{Running, false},
	}
	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			if got := tt.state.IsFinal(); got != tt.expected {
				t.Errorf("IsFinal() = %v, want %v", got, tt.expected)
			}
		})
	}
}
