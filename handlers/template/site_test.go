package template

import (
	"context"
	"testing"

	configv1 "github.com/szeber/kube-stager/apis/config/v1"
	sitev1 "github.com/szeber/kube-stager/apis/site/v1"
	"github.com/szeber/kube-stager/internal/testutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewSite(t *testing.T) {
	site := sitev1.StagingSite{
		ObjectMeta: metav1.ObjectMeta{Name: "mysite", Namespace: "test-ns"},
		Spec: sitev1.StagingSiteSpec{
			DomainPrefix: "mysite",
			Services: map[string]sitev1.StagingSiteService{
				"web": {ImageTag: "v1.0"},
			},
		},
	}
	config := configv1.ServiceConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "web"},
	}
	handler := NewSite(site, config)
	if handler.site.Name != "mysite" {
		t.Errorf("site.Name = %q, want %q", handler.site.Name, "mysite")
	}
	if handler.siteServiceSpec.ImageTag != "v1.0" {
		t.Errorf("siteServiceSpec.ImageTag = %q, want %q", handler.siteServiceSpec.ImageTag, "v1.0")
	}
}

func TestLoadConfigs(t *testing.T) {
	mysqlCfg := testutil.NewTestMysqlConfig("mysql1", "test-ns")
	mongoCfg := testutil.NewTestMongoConfig("mongo1", "test-ns")
	redisCfg := testutil.NewTestRedisConfig("redis1", "test-ns")
	svcCfg := testutil.NewTestServiceConfig("web", "test-ns", "web")
	c := testutil.NewFakeClient(mysqlCfg, mongoCfg, redisCfg, svcCfg)

	site := sitev1.StagingSite{
		ObjectMeta: metav1.ObjectMeta{Name: "mysite", Namespace: "test-ns"},
		Spec: sitev1.StagingSiteSpec{
			Services: map[string]sitev1.StagingSiteService{"web": {}},
		},
	}
	config := configv1.ServiceConfig{ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "test-ns"}}
	handler := NewSite(site, config)

	err := LoadConfigs(&handler, context.Background(), c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(handler.GetMysql()) != 1 {
		t.Errorf("mysql configs = %d, want 1", len(handler.GetMysql()))
	}
	if len(handler.GetMongo()) != 1 {
		t.Errorf("mongo configs = %d, want 1", len(handler.GetMongo()))
	}
	if len(handler.GetRedis()) != 1 {
		t.Errorf("redis configs = %d, want 1", len(handler.GetRedis()))
	}
}

func TestListConfigsInNamespace(t *testing.T) {
	t.Run("ListMongoConfigsInNamespace", func(t *testing.T) {
		mongoCfg := testutil.NewTestMongoConfig("mongo1", "test-ns")
		c := testutil.NewFakeClient(mongoCfg)
		result, err := ListMongoConfigsInNamespace("test-ns", context.Background(), c)
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		if len(result) != 1 {
			t.Errorf("got %d, want 1", len(result))
		}
	})

	t.Run("ListMysqlConfigsInNamespace", func(t *testing.T) {
		mysqlCfg := testutil.NewTestMysqlConfig("mysql1", "test-ns")
		c := testutil.NewFakeClient(mysqlCfg)
		result, err := ListMysqlConfigsInNamespace("test-ns", context.Background(), c)
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		if len(result) != 1 {
			t.Errorf("got %d, want 1", len(result))
		}
	})

	t.Run("ListRedisConfigsInNamespace", func(t *testing.T) {
		redisCfg := testutil.NewTestRedisConfig("redis1", "test-ns")
		c := testutil.NewFakeClient(redisCfg)
		result, err := ListRedisConfigsInNamespace("test-ns", context.Background(), c)
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		if len(result) != 1 {
			t.Errorf("got %d, want 1", len(result))
		}
	})
}

func TestLoadServiceConfigs(t *testing.T) {
	svcCfg := testutil.NewTestServiceConfig("web", "test-ns", "web")
	c := testutil.NewFakeClient(svcCfg)

	site := sitev1.StagingSite{
		ObjectMeta: metav1.ObjectMeta{Name: "mysite", Namespace: "test-ns"},
	}
	config := configv1.ServiceConfig{ObjectMeta: metav1.ObjectMeta{Name: "web"}}
	handler := NewSite(site, config)
	err := LoadServiceConfigs(&handler, context.Background(), c)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(handler.serviceConfigs) != 1 {
		t.Errorf("serviceConfigs = %d, want 1", len(handler.serviceConfigs))
	}
}

func TestSettersAndGetters(t *testing.T) {
	site := sitev1.StagingSite{ObjectMeta: metav1.ObjectMeta{Name: "s", Namespace: "ns"}}
	config := configv1.ServiceConfig{ObjectMeta: metav1.ObjectMeta{Name: "c"}}
	handler := NewSite(site, config)

	mysqlMap := map[string]configv1.MysqlConfig{"m": {}}
	handler.SetMysql(mysqlMap)
	if len(handler.GetMysql()) != 1 {
		t.Error("SetMysql/GetMysql round-trip failed")
	}

	mongoMap := map[string]configv1.MongoConfig{"m": {}}
	handler.SetMongo(mongoMap)
	if len(handler.GetMongo()) != 1 {
		t.Error("SetMongo/GetMongo round-trip failed")
	}

	redisMap := map[string]configv1.RedisConfig{"r": {}}
	handler.SetRedis(redisMap)
	if len(handler.GetRedis()) != 1 {
		t.Error("SetRedis/GetRedis round-trip failed")
	}

	svcMap := map[string]configv1.ServiceConfig{"s": {}}
	handler.SetServiceConfigs(svcMap)
	if len(handler.serviceConfigs) != 1 {
		t.Error("SetServiceConfigs failed")
	}

	handler.SetServiceConfig("s2", configv1.ServiceConfig{})
	if len(handler.serviceConfigs) != 2 {
		t.Error("SetServiceConfig failed")
	}
}

func TestGetTemplateValues(t *testing.T) {
	site := sitev1.StagingSite{
		ObjectMeta: metav1.ObjectMeta{Name: "mysite", Namespace: "test-ns"},
		Spec: sitev1.StagingSiteSpec{
			DomainPrefix: "myprefix",
			Password:     "testpass",
			Services: map[string]sitev1.StagingSiteService{
				"web": {
					ImageTag:                    "v1.0",
					MysqlEnvironment:            "mysql1",
					DbInitSourceEnvironmentName: "master",
				},
			},
		},
		Status: sitev1.StagingSiteStatus{
			Services: map[string]sitev1.StagingSiteServiceStatus{
				"web": {Username: "testuser", DbName: "testdb"},
			},
		},
	}
	config := configv1.ServiceConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "web"},
		Spec: configv1.ServiceConfigSpec{
			ShortName: "web",
			ConfigMaps: map[string]configv1.Configmap{
				"env": {"KEY": "VALUE"},
			},
			CustomTemplateValues: map[string]string{
				"custom1": "val1",
			},
		},
	}
	handler := NewSite(site, config)
	handler.SetMysql(map[string]configv1.MysqlConfig{
		"mysql1": {
			ObjectMeta: metav1.ObjectMeta{Name: "mysql1"},
			Spec:       configv1.MysqlConfigSpec{Host: "mysql.example.com", Port: 3306},
		},
	})

	values := handler.GetTemplateValues()

	checks := map[string]string{
		"site.name":           "mysite",
		"site.domainPrefix":   "myprefix",
		"site.imageTag":       "v1.0",
		"database.username":   "testuser",
		"database.name":       "testdb",
		"database.password":   "testpass",
		"database.initSource": "master",
		"database.mysql.host": "mysql.example.com",
		"database.mysql.port": "3306",
		"site.custom.custom1": "val1",
	}

	for key, expected := range checks {
		if got, ok := values[key]; !ok {
			t.Errorf("missing key %q", key)
		} else if got != expected {
			t.Errorf("values[%q] = %q, want %q", key, got, expected)
		}
	}

	if _, ok := values["site.configmap.env"]; !ok {
		t.Error("missing site.configmap.env key")
	}
}
