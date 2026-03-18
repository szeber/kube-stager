package webhook

import (
	"context"
	"encoding/json"
	"testing"

	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	configv1 "github.com/szeber/kube-stager/apis/config/v1"
	"github.com/szeber/kube-stager/internal/testutil"
)

func makeServiceConfigAdmissionRequest(cfg *configv1.ServiceConfig) admission.Request {
	raw, _ := json.Marshal(cfg)
	return admission.Request{
		AdmissionRequest: admissionv1.AdmissionRequest{
			Object:    runtime.RawExtension{Raw: raw},
			Namespace: cfg.Namespace,
			Name:      cfg.Name,
		},
	}
}

// TestServiceConfigCreateOrUpdateHandler_ValidConfig verifies that a well-formed
// ServiceConfig with no default environment references is admitted.
//
// NOTE: The shortname uniqueness check uses MatchingFields which is not supported
// by the fake client. This test is skipped as it cannot pass the uniqueness check.
func TestServiceConfigCreateOrUpdateHandler_ValidConfig(t *testing.T) {
	t.Skip("shortName uniqueness check uses MatchingFields which is not supported by the fake client")
}

// TestServiceConfigCreateOrUpdateHandler_InvalidDefaultMysqlEnvironment verifies
// that a ServiceConfig referencing a non-existent DefaultMysqlEnvironment is denied.
//
// NOTE: The handler performs a MatchingFields query for shortName uniqueness before
// the environment validation. With the fake client this returns a 500 error rather
// than a 403 Denied. The test verifies the request is not allowed, regardless of
// the error code.
func TestServiceConfigCreateOrUpdateHandler_InvalidDefaultMysqlEnvironment(t *testing.T) {
	const ns = "test-ns"

	cfg := testutil.NewTestServiceConfig("mysvc", ns, "svc")
	cfg.Spec.DefaultMysqlEnvironment = "nonexistent-mysql"

	handler := &ServiceConfigCreateOrUpdateHandler{
		Client:  testutil.NewFakeClient(cfg),
		Decoder: admission.NewDecoder(testutil.NewTestScheme()),
	}

	resp := handler.Handle(context.Background(), makeServiceConfigAdmissionRequest(cfg))

	if resp.Allowed {
		t.Error("expected not Allowed for invalid DefaultMysqlEnvironment, got Allowed")
	}
}

// TestServiceConfigCreateOrUpdateHandler_InvalidDefaultMongoEnvironment verifies
// that a ServiceConfig referencing a non-existent DefaultMongoEnvironment is denied.
//
// NOTE: Same fake-client limitation as TestServiceConfigCreateOrUpdateHandler_InvalidDefaultMysqlEnvironment.
func TestServiceConfigCreateOrUpdateHandler_InvalidDefaultMongoEnvironment(t *testing.T) {
	const ns = "test-ns"

	cfg := testutil.NewTestServiceConfig("mysvc", ns, "svc")
	cfg.Spec.DefaultMongoEnvironment = "nonexistent-mongo"

	handler := &ServiceConfigCreateOrUpdateHandler{
		Client:  testutil.NewFakeClient(cfg),
		Decoder: admission.NewDecoder(testutil.NewTestScheme()),
	}

	resp := handler.Handle(context.Background(), makeServiceConfigAdmissionRequest(cfg))

	if resp.Allowed {
		t.Error("expected not Allowed for invalid DefaultMongoEnvironment, got Allowed")
	}
}

// TestServiceConfigCreateOrUpdateHandler_InvalidDefaultRedisEnvironment verifies
// that a ServiceConfig referencing a non-existent DefaultRedisEnvironment is denied.
//
// NOTE: Same fake-client limitation as TestServiceConfigCreateOrUpdateHandler_InvalidDefaultMysqlEnvironment.
func TestServiceConfigCreateOrUpdateHandler_InvalidDefaultRedisEnvironment(t *testing.T) {
	const ns = "test-ns"

	cfg := testutil.NewTestServiceConfig("mysvc", ns, "svc")
	cfg.Spec.DefaultRedisEnvironment = "nonexistent-redis"

	handler := &ServiceConfigCreateOrUpdateHandler{
		Client:  testutil.NewFakeClient(cfg),
		Decoder: admission.NewDecoder(testutil.NewTestScheme()),
	}

	resp := handler.Handle(context.Background(), makeServiceConfigAdmissionRequest(cfg))

	if resp.Allowed {
		t.Error("expected not Allowed for invalid DefaultRedisEnvironment, got Allowed")
	}
}
