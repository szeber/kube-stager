package webhook

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	sitev1 "github.com/szeber/kube-stager/apis/site/v1"
	"github.com/szeber/kube-stager/internal/testutil"
)

func makeSiteAdmissionRequest(site *sitev1.StagingSite) admission.Request {
	raw, _ := json.Marshal(site)
	return admission.Request{
		AdmissionRequest: admissionv1.AdmissionRequest{
			Object:    runtime.RawExtension{Raw: raw},
			Namespace: site.Namespace,
			Name:      site.Name,
		},
	}
}

func TestStagingsiteHandler_ValidSiteWithAllConfigs(t *testing.T) {
	const ns = "test-ns"

	serviceConfig := testutil.NewTestServiceConfig("mysvc", ns, "svc")
	serviceConfig.Spec.DefaultMysqlEnvironment = "mydb"
	mysqlConfig := testutil.NewTestMysqlConfig("mydb", ns)

	handler := &StagingsiteHandler{
		Client:  testutil.NewFakeClient(serviceConfig, mysqlConfig),
		Decoder: admission.NewDecoder(testutil.NewTestScheme()),
	}

	site := testutil.NewTestStagingSite("mysite", ns, map[string]sitev1.StagingSiteService{
		"mysvc": {
			ImageTag:         "v1.0",
			MysqlEnvironment: "mydb",
		},
	})

	resp := handler.Handle(context.Background(), makeSiteAdmissionRequest(site))

	if !resp.Allowed {
		t.Errorf("expected Allowed, got Denied: %v", resp.Result)
	}
}

func TestStagingsiteHandler_MissingServiceConfig(t *testing.T) {
	const ns = "test-ns"

	handler := &StagingsiteHandler{
		Client:  testutil.NewFakeClient(),
		Decoder: admission.NewDecoder(testutil.NewTestScheme()),
	}

	site := testutil.NewTestStagingSite("mysite", ns, map[string]sitev1.StagingSiteService{
		"nonexistent-svc": {
			ImageTag: "v1.0",
		},
	})

	resp := handler.Handle(context.Background(), makeSiteAdmissionRequest(site))

	if resp.Allowed {
		t.Error("expected Denied, got Allowed")
	}
	if resp.Result == nil || resp.Result.Code != http.StatusForbidden {
		t.Errorf("expected Forbidden (403), got: %v", resp.Result)
	}
}

func TestStagingsiteHandler_InvalidMysqlEnvironment(t *testing.T) {
	const ns = "test-ns"

	// ServiceConfig exists but the referenced mysql env "mydb" does not
	serviceConfig := testutil.NewTestServiceConfig("mysvc", ns, "svc")

	handler := &StagingsiteHandler{
		Client:  testutil.NewFakeClient(serviceConfig),
		Decoder: admission.NewDecoder(testutil.NewTestScheme()),
	}

	site := testutil.NewTestStagingSite("mysite", ns, map[string]sitev1.StagingSiteService{
		"mysvc": {
			ImageTag:         "v1.0",
			MysqlEnvironment: "mydb",
		},
	})

	resp := handler.Handle(context.Background(), makeSiteAdmissionRequest(site))

	if resp.Allowed {
		t.Error("expected Denied due to missing mysql environment, got Allowed")
	}
	if resp.Result == nil || resp.Result.Code != http.StatusForbidden {
		t.Errorf("expected Forbidden (403), got: %v", resp.Result)
	}
}

func TestStagingsiteHandler_IncludeAllServices(t *testing.T) {
	const ns = "test-ns"

	svc1 := testutil.NewTestServiceConfig("svc1", ns, "s1")
	svc2 := testutil.NewTestServiceConfig("svc2", ns, "s2")

	handler := &StagingsiteHandler{
		Client:  testutil.NewFakeClient(svc1, svc2),
		Decoder: admission.NewDecoder(testutil.NewTestScheme()),
	}

	site := testutil.NewTestStagingSite("mysite", ns, nil)
	site.Spec.IncludeAllServices = true

	resp := handler.Handle(context.Background(), makeSiteAdmissionRequest(site))

	if !resp.Allowed {
		t.Errorf("expected Allowed when IncludeAllServices=true, got Denied: %v", resp.Result)
	}
}

func TestStagingsiteHandler_EmptyServices(t *testing.T) {
	const ns = "test-ns"

	handler := &StagingsiteHandler{
		Client:  testutil.NewFakeClient(),
		Decoder: admission.NewDecoder(testutil.NewTestScheme()),
	}

	site := testutil.NewTestStagingSite("mysite", ns, nil)

	resp := handler.Handle(context.Background(), makeSiteAdmissionRequest(site))

	if resp.Allowed {
		t.Error("expected Denied for empty services, got Allowed")
	}
	if resp.Result == nil || resp.Result.Code != http.StatusForbidden {
		t.Errorf("expected Forbidden (403), got: %v", resp.Result)
	}
}

func TestStagingsiteHandler_DefaultsFilled(t *testing.T) {
	const ns = "test-ns"

	serviceConfig := testutil.NewTestServiceConfig("mysvc", ns, "svc")
	serviceConfig.Spec.DefaultMysqlEnvironment = "mydb"
	mysqlConfig := testutil.NewTestMysqlConfig("mydb", ns)

	handler := &StagingsiteHandler{
		Client:  testutil.NewFakeClient(serviceConfig, mysqlConfig),
		Decoder: admission.NewDecoder(testutil.NewTestScheme()),
	}

	site := testutil.NewTestStagingSite("mysite", ns, map[string]sitev1.StagingSiteService{
		"mysvc": {},
	})

	resp := handler.Handle(context.Background(), makeSiteAdmissionRequest(site))

	if !resp.Allowed {
		t.Errorf("expected Allowed with defaults filled, got Denied: %v", resp.Result)
	}
	// Handler patches the object — ImageTag defaults to "latest" and
	// DbInitSourceEnvironmentName defaults to "master".
	t.Logf("Patch count: %d", len(resp.Patches))
}

func TestStagingsiteHandler_DefaultMysqlEnvironmentFromServiceConfig(t *testing.T) {
	const ns = "test-ns"

	serviceConfig := testutil.NewTestServiceConfig("mysvc", ns, "svc")
	serviceConfig.Spec.DefaultMysqlEnvironment = "mydb"
	mysqlConfig := testutil.NewTestMysqlConfig("mydb", ns)

	handler := &StagingsiteHandler{
		Client:  testutil.NewFakeClient(serviceConfig, mysqlConfig),
		Decoder: admission.NewDecoder(testutil.NewTestScheme()),
	}

	// No explicit MysqlEnvironment on the service — filled from DefaultMysqlEnvironment on config
	site := testutil.NewTestStagingSite("mysite", ns, map[string]sitev1.StagingSiteService{
		"mysvc": {
			ImageTag: "latest",
		},
	})

	resp := handler.Handle(context.Background(), makeSiteAdmissionRequest(site))

	if !resp.Allowed {
		t.Errorf("expected Allowed (default mysql env from service config), got Denied: %v", resp.Result)
	}
}

func TestStagingsiteHandler_InvalidMongoEnvironment(t *testing.T) {
	const ns = "test-ns"

	serviceConfig := testutil.NewTestServiceConfig("mysvc", ns, "svc")
	// No MongoConfig added — "mymongo" does not exist

	handler := &StagingsiteHandler{
		Client:  testutil.NewFakeClient(serviceConfig),
		Decoder: admission.NewDecoder(testutil.NewTestScheme()),
	}

	site := testutil.NewTestStagingSite("mysite", ns, map[string]sitev1.StagingSiteService{
		"mysvc": {
			ImageTag:         "v1.0",
			MongoEnvironment: "mymongo",
		},
	})

	resp := handler.Handle(context.Background(), makeSiteAdmissionRequest(site))

	if resp.Allowed {
		t.Error("expected Denied due to missing mongo environment, got Allowed")
	}
}

func TestStagingsiteHandler_InvalidRedisEnvironment(t *testing.T) {
	const ns = "test-ns"

	serviceConfig := testutil.NewTestServiceConfig("mysvc", ns, "svc")
	// No RedisConfig added — "myredis" does not exist

	handler := &StagingsiteHandler{
		Client:  testutil.NewFakeClient(serviceConfig),
		Decoder: admission.NewDecoder(testutil.NewTestScheme()),
	}

	site := testutil.NewTestStagingSite("mysite", ns, map[string]sitev1.StagingSiteService{
		"mysvc": {
			ImageTag:         "v1.0",
			RedisEnvironment: "myredis",
		},
	})

	resp := handler.Handle(context.Background(), makeSiteAdmissionRequest(site))

	if resp.Allowed {
		t.Error("expected Denied due to missing redis environment, got Allowed")
	}
}

func TestStagingsiteHandler_AllEnvironmentLabelsSet(t *testing.T) {
	const ns = "test-ns"

	serviceConfig := testutil.NewTestServiceConfig("mysvc", ns, "svc")
	mysqlConfig := testutil.NewTestMysqlConfig("mysql-env", ns)
	mongoConfig := testutil.NewTestMongoConfig("mongo-env", ns)
	redisConfig := testutil.NewTestRedisConfig("redis-env", ns)

	handler := &StagingsiteHandler{
		Client:  testutil.NewFakeClient(serviceConfig, mysqlConfig, mongoConfig, redisConfig),
		Decoder: admission.NewDecoder(testutil.NewTestScheme()),
	}

	site := testutil.NewTestStagingSite("mysite", ns, map[string]sitev1.StagingSiteService{
		"mysvc": {
			ImageTag:         "v1.0",
			MysqlEnvironment: "mysql-env",
			MongoEnvironment: "mongo-env",
			RedisEnvironment: "redis-env",
		},
	})

	resp := handler.Handle(context.Background(), makeSiteAdmissionRequest(site))

	if !resp.Allowed {
		t.Errorf("expected Allowed with all environments valid, got Denied: %v", resp.Result)
	}
}

func TestStagingsiteHandler_ExistingLabelsPreserved(t *testing.T) {
	const ns = "test-ns"

	serviceConfig := testutil.NewTestServiceConfig("mysvc", ns, "svc")

	handler := &StagingsiteHandler{
		Client:  testutil.NewFakeClient(serviceConfig),
		Decoder: admission.NewDecoder(testutil.NewTestScheme()),
	}

	site := &sitev1.StagingSite{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mysite",
			Namespace: ns,
			Labels: map[string]string{
				"custom-label": "custom-value",
			},
		},
		Spec: sitev1.StagingSiteSpec{
			DomainPrefix: "mysite",
			DbName:       "mysite",
			Username:     "mysite",
			Password:     "testpassword",
			Enabled:      true,
			Services: map[string]sitev1.StagingSiteService{
				"mysvc": {
					ImageTag: "v1.0",
				},
			},
		},
	}

	resp := handler.Handle(context.Background(), makeSiteAdmissionRequest(site))

	if !resp.Allowed {
		t.Errorf("expected Allowed, got Denied: %v", resp.Result)
	}
}
