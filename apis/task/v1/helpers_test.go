package v1

import (
	"testing"

	configv1 "github.com/szeber/kube-stager/apis/config/v1"
	sitev1 "github.com/szeber/kube-stager/apis/site/v1"
	"github.com/szeber/kube-stager/helpers/labels"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func makeTestSiteAndConfig() (*sitev1.StagingSite, *configv1.ServiceConfig) {
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
				"myservice": {ImageTag: "v1.0"},
			},
		},
	}
	config := &configv1.ServiceConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "myservice",
			Namespace: "test-ns",
		},
		Spec: configv1.ServiceConfigSpec{ShortName: "svc"},
	}
	return site, config
}

func TestMysqlDatabase_PopulateFomSite(t *testing.T) {
	t.Run("populates all fields", func(t *testing.T) {
		site, config := makeTestSiteAndConfig()
		db := &MysqlDatabase{}
		err := db.PopulateFomSite(site, config, "prod")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if db.Namespace != "test-ns" {
			t.Errorf("Namespace = %q, want %q", db.Namespace, "test-ns")
		}
		if db.Spec.EnvironmentConfig.SiteName != "test-site" {
			t.Errorf("SiteName = %q, want %q", db.Spec.EnvironmentConfig.SiteName, "test-site")
		}
		if db.Spec.EnvironmentConfig.Environment != "prod" {
			t.Errorf("Environment = %q, want %q", db.Spec.EnvironmentConfig.Environment, "prod")
		}
		if db.Spec.Password != "testpass" {
			t.Errorf("Password = %q, want %q", db.Spec.Password, "testpass")
		}
		if db.Labels[labels.MysqlEnvironment] != "prod" {
			t.Errorf("Label MysqlEnvironment = %q, want %q", db.Labels[labels.MysqlEnvironment], "prod")
		}
	})

	t.Run("nil config returns error", func(t *testing.T) {
		site, _ := makeTestSiteAndConfig()
		db := &MysqlDatabase{}
		err := db.PopulateFomSite(site, nil, "prod")
		if err == nil {
			t.Error("expected error for nil config")
		}
	})
}

func TestMysqlDatabase_Matches(t *testing.T) {
	site, config := makeTestSiteAndConfig()
	db1 := &MysqlDatabase{}
	_ = db1.PopulateFomSite(site, config, "prod")
	db2 := &MysqlDatabase{}
	_ = db2.PopulateFomSite(site, config, "prod")

	t.Run("identical returns true", func(t *testing.T) {
		if !db1.Matches(*db2) {
			t.Error("expected Matches() to return true for identical databases")
		}
	})

	t.Run("different spec returns false", func(t *testing.T) {
		db3 := &MysqlDatabase{}
		_ = db3.PopulateFomSite(site, config, "staging")
		if db1.Matches(*db3) {
			t.Error("expected Matches() to return false for different environment")
		}
	})
}

func TestMysqlDatabase_UpdateFromExpected(t *testing.T) {
	site, config := makeTestSiteAndConfig()
	db1 := &MysqlDatabase{}
	_ = db1.PopulateFomSite(site, config, "prod")
	db2 := &MysqlDatabase{}
	_ = db2.PopulateFomSite(site, config, "staging")

	db1.UpdateFromExpected(*db2)
	if db1.Spec.EnvironmentConfig.Environment != "staging" {
		t.Errorf("Environment = %q, want %q after update", db1.Spec.EnvironmentConfig.Environment, "staging")
	}
}

func TestMongoDatabase_PopulateFomSite(t *testing.T) {
	t.Run("populates all fields", func(t *testing.T) {
		site, config := makeTestSiteAndConfig()
		db := &MongoDatabase{}
		err := db.PopulateFomSite(site, config, "prod")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if db.Namespace != "test-ns" {
			t.Errorf("Namespace = %q, want %q", db.Namespace, "test-ns")
		}
		if db.Spec.EnvironmentConfig.SiteName != "test-site" {
			t.Errorf("SiteName = %q, want %q", db.Spec.EnvironmentConfig.SiteName, "test-site")
		}
		if db.Labels[labels.MongoEnvironment] != "prod" {
			t.Errorf("Label MongoEnvironment = %q, want %q", db.Labels[labels.MongoEnvironment], "prod")
		}
	})

	t.Run("nil config returns error", func(t *testing.T) {
		site, _ := makeTestSiteAndConfig()
		db := &MongoDatabase{}
		if err := db.PopulateFomSite(site, nil, "prod"); err == nil {
			t.Error("expected error for nil config")
		}
	})
}

func TestMongoDatabase_Matches(t *testing.T) {
	site, config := makeTestSiteAndConfig()
	db1 := &MongoDatabase{}
	_ = db1.PopulateFomSite(site, config, "prod")
	db2 := &MongoDatabase{}
	_ = db2.PopulateFomSite(site, config, "prod")

	if !db1.Matches(*db2) {
		t.Error("expected Matches() to return true for identical databases")
	}

	db3 := &MongoDatabase{}
	_ = db3.PopulateFomSite(site, config, "staging")
	if db1.Matches(*db3) {
		t.Error("expected Matches() to return false for different environment")
	}
}

func TestMongoDatabase_UpdateFromExpected(t *testing.T) {
	site, config := makeTestSiteAndConfig()
	db1 := &MongoDatabase{}
	_ = db1.PopulateFomSite(site, config, "prod")
	db2 := &MongoDatabase{}
	_ = db2.PopulateFomSite(site, config, "staging")
	db1.UpdateFromExpected(*db2)
	if db1.Spec.EnvironmentConfig.Environment != "staging" {
		t.Errorf("Environment = %q, want %q", db1.Spec.EnvironmentConfig.Environment, "staging")
	}
}

func TestRedisDatabase_PopulateFomSite(t *testing.T) {
	t.Run("populates all fields", func(t *testing.T) {
		site, config := makeTestSiteAndConfig()
		db := &RedisDatabase{}
		err := db.PopulateFomSite(site, config, "prod")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if db.Namespace != "test-ns" {
			t.Errorf("Namespace = %q, want %q", db.Namespace, "test-ns")
		}
		if db.Labels[labels.RedisEnvironment] != "prod" {
			t.Errorf("Label RedisEnvironment = %q, want %q", db.Labels[labels.RedisEnvironment], "prod")
		}
	})

	t.Run("nil config returns error", func(t *testing.T) {
		site, _ := makeTestSiteAndConfig()
		db := &RedisDatabase{}
		if err := db.PopulateFomSite(site, nil, "prod"); err == nil {
			t.Error("expected error for nil config")
		}
	})
}

func TestRedisDatabase_Matches(t *testing.T) {
	site, config := makeTestSiteAndConfig()
	db1 := &RedisDatabase{}
	_ = db1.PopulateFomSite(site, config, "prod")
	db2 := &RedisDatabase{}
	_ = db2.PopulateFomSite(site, config, "prod")
	if !db1.Matches(*db2) {
		t.Error("expected Matches() to return true")
	}

	db3 := &RedisDatabase{}
	_ = db3.PopulateFomSite(site, config, "staging")
	if db1.Matches(*db3) {
		t.Error("expected Matches() to return false for different env")
	}
}

func TestRedisDatabase_UpdateFromExpected(t *testing.T) {
	site, config := makeTestSiteAndConfig()
	db1 := &RedisDatabase{}
	_ = db1.PopulateFomSite(site, config, "prod")
	db2 := &RedisDatabase{}
	_ = db2.PopulateFomSite(site, config, "staging")
	db1.UpdateFromExpected(*db2)
	if db1.Spec.EnvironmentConfig.Environment != "staging" {
		t.Errorf("Environment = %q, want %q", db1.Spec.EnvironmentConfig.Environment, "staging")
	}
}

func TestEnvironmentConfig_Getters(t *testing.T) {
	ec := EnvironmentConfig{
		ServiceName: "svc",
		SiteName:    "site",
		Environment: "env",
	}
	if ec.GetSiteName() != "site" {
		t.Errorf("GetSiteName() = %q, want %q", ec.GetSiteName(), "site")
	}
	if ec.GetServiceName() != "svc" {
		t.Errorf("GetServiceName() = %q, want %q", ec.GetServiceName(), "svc")
	}
	if ec.GetEnvironment() != "env" {
		t.Errorf("GetEnvironment() = %q, want %q", ec.GetEnvironment(), "env")
	}
}
