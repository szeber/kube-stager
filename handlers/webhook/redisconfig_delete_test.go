package webhook

import (
	"context"
	"net/http"
	"testing"

	"github.com/szeber/kube-stager/helpers/labels"
	appmetrics "github.com/szeber/kube-stager/internal/metrics"
	"github.com/szeber/kube-stager/internal/metricstest"
	"github.com/szeber/kube-stager/internal/testutil"
)

// TestRedisConfigDeleteHandler_SitesUsingEnvironment verifies that a RedisConfig
// cannot be deleted when StagingSites carry the label
// redis.environments.operator.kube-stager.io/<name>=true.
// The handler denies at the label check before reaching the MatchingFields query.
func TestRedisConfigDeleteHandler_SitesUsingEnvironment(t *testing.T) {
	const ns = "test-ns"
	const envName = "my-redis"

	site := testutil.NewTestStagingSite("mysite", ns, nil)
	site.Labels = map[string]string{
		labels.RedisEnvironmentsPrefix + envName: "true",
	}

	handler := &RedisConfigDeleteHandler{
		Client: testutil.NewFakeClient(site),
	}

	before := metricstest.GetCounterValue(appmetrics.WebhookDenied, "redisconfig_delete", "resource_in_use")
	resp := handler.Handle(context.Background(), makeDeleteAdmissionRequest(ns, envName))

	if resp.Allowed {
		t.Error("expected Denied when sites use the redis environment, got Allowed")
	}
	if resp.Result == nil || resp.Result.Code != http.StatusForbidden {
		t.Errorf("expected Forbidden (403), got: %v", resp.Result)
	}
	after := metricstest.GetCounterValue(appmetrics.WebhookDenied, "redisconfig_delete", "resource_in_use")
	if after-before != 1 {
		t.Errorf("expected webhook_denied_total to increment by 1, got delta %v", after-before)
	}
}

func TestRedisConfigDeleteHandler_MultipleSitesUsingEnvironment(t *testing.T) {
	const ns = "test-ns"
	const envName = "my-redis"

	site1 := testutil.NewTestStagingSite("site1", ns, nil)
	site1.Labels = map[string]string{
		labels.RedisEnvironmentsPrefix + envName: "true",
	}
	site2 := testutil.NewTestStagingSite("site2", ns, nil)
	site2.Labels = map[string]string{
		labels.RedisEnvironmentsPrefix + envName: "true",
	}

	handler := &RedisConfigDeleteHandler{
		Client: testutil.NewFakeClient(site1, site2),
	}

	resp := handler.Handle(context.Background(), makeDeleteAdmissionRequest(ns, envName))

	if resp.Allowed {
		t.Error("expected Denied when multiple sites use the redis environment, got Allowed")
	}
}

// TestRedisConfigDeleteHandler_NoReferences tests that the label-only check passes
// when no sites reference the environment.
//
// NOTE: The handler also performs a MatchingFields query on ServiceConfigs which is
// not supported by the fake client. Tests that would reach that code path are
// omitted; those scenarios are covered by integration tests using envtest.
func TestRedisConfigDeleteHandler_NoReferences(t *testing.T) {
	t.Skip("MatchingFields on ServiceConfig is not supported by the fake client; " +
		"covered by integration tests")
}
