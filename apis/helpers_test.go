package api

import (
	"strings"
	"testing"

	configv1 "github.com/szeber/kube-stager/apis/config/v1"
	sitev1 "github.com/szeber/kube-stager/apis/site/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func makeSiteAndService(siteName, dbName, username, shortName string) (*sitev1.StagingSite, *configv1.ServiceConfig) {
	site := &sitev1.StagingSite{
		ObjectMeta: metav1.ObjectMeta{Name: siteName, Namespace: "test-ns"},
		Spec: sitev1.StagingSiteSpec{
			DbName:   dbName,
			Username: username,
		},
	}
	svc := &configv1.ServiceConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "myservice"},
		Spec:       configv1.ServiceConfigSpec{ShortName: shortName},
	}
	return site, svc
}

func TestMakeDatabaseName(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		site, svc := makeSiteAndService("mysite", "mydb", "user", "web")
		got := MakeDatabaseName(site, svc)
		if got != "mydb_web" {
			t.Errorf("MakeDatabaseName() = %q, want %q", got, "mydb_web")
		}
	})

	t.Run("long names shortened", func(t *testing.T) {
		site, svc := makeSiteAndService("mysite", strings.Repeat("a", 60), "user", "web")
		got := MakeDatabaseName(site, svc)
		if len(got) > 63 {
			t.Errorf("MakeDatabaseName() length = %d, want <= 63", len(got))
		}
	})

	t.Run("special chars sanitized", func(t *testing.T) {
		site, svc := makeSiteAndService("mysite", "my-db", "user", "web")
		got := MakeDatabaseName(site, svc)
		if strings.Contains(got, "-") {
			t.Errorf("MakeDatabaseName() = %q, should not contain hyphens", got)
		}
	})
}

func TestMakeUsername(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		site, svc := makeSiteAndService("mysite", "mydb", "usr", "web")
		got := MakeUsername(site, svc)
		if got != "usr_web" {
			t.Errorf("MakeUsername() = %q, want %q", got, "usr_web")
		}
	})

	t.Run("long names hashed to 10 chars", func(t *testing.T) {
		site, svc := makeSiteAndService("mysite", "mydb", "verylongusername", "longshort")
		got := MakeUsername(site, svc)
		if len(got) > 16 {
			t.Errorf("MakeUsername() length = %d, want <= 16", len(got))
		}
	})
}

func TestMakeConfigmapName(t *testing.T) {
	site, svc := makeSiteAndService("mysite", "mydb", "user", "web")
	got := MakeConfigmapName(site, svc, "env")
	if !strings.Contains(got, "web") || !strings.Contains(got, "env") {
		t.Errorf("MakeConfigmapName() = %q, expected to contain shortname and type", got)
	}
}

func TestMakeServiceName(t *testing.T) {
	site, svc := makeSiteAndService("mysite", "mydb", "user", "web")
	got := MakeServiceName(site, svc)
	if got != "mysite-web" {
		t.Errorf("MakeServiceName() = %q, want %q", got, "mysite-web")
	}
}

func TestMakeIngressName(t *testing.T) {
	site, svc := makeSiteAndService("mysite", "mydb", "user", "web")
	got := MakeIngressName(site, svc)
	if got != "mysite-web" {
		t.Errorf("MakeIngressName() = %q, want %q", got, "mysite-web")
	}
}

func TestMakeDeploymentName(t *testing.T) {
	site, svc := makeSiteAndService("mysite", "mydb", "user", "web")
	got := MakeDeploymentName(site, svc)
	if got != "mysite-web" {
		t.Errorf("MakeDeploymentName() = %q, want %q", got, "mysite-web")
	}
}

func TestMakeServiceUrl(t *testing.T) {
	site := &sitev1.StagingSite{
		ObjectMeta: metav1.ObjectMeta{Name: "mysite", Namespace: "test-ns"},
	}
	got := MakeServiceUrl(site, "web")
	expected := "mysite-web.test-ns.svc.cluster.local"
	if got != expected {
		t.Errorf("MakeServiceUrl() = %q, want %q", got, expected)
	}
}
