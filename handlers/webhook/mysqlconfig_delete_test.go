package webhook

import (
	"context"
	"net/http"
	"testing"

	"github.com/szeber/kube-stager/helpers/labels"
	"github.com/szeber/kube-stager/internal/testutil"
)

// TestMysqlConfigDeleteHandler_SitesUsingEnvironment verifies that a MysqlConfig
// cannot be deleted when StagingSites carry the label
// mysql.environments.operator.kube-stager.io/<name>=true.
// The handler denies at the label check before reaching the MatchingFields query.
func TestMysqlConfigDeleteHandler_SitesUsingEnvironment(t *testing.T) {
	const ns = "test-ns"
	const envName = "my-mysql"

	site := testutil.NewTestStagingSite("mysite", ns, nil)
	site.Labels = map[string]string{
		labels.MysqlEnvironmentsPrefix + envName: "true",
	}

	handler := &MysqlConfigDeleteHandler{
		Client: testutil.NewFakeClient(site),
	}

	resp := handler.Handle(context.Background(), makeDeleteAdmissionRequest(ns, envName))

	if resp.Allowed {
		t.Error("expected Denied when sites use the mysql environment, got Allowed")
	}
	if resp.Result == nil || resp.Result.Code != http.StatusForbidden {
		t.Errorf("expected Forbidden (403), got: %v", resp.Result)
	}
}

func TestMysqlConfigDeleteHandler_MultipleSitesUsingEnvironment(t *testing.T) {
	const ns = "test-ns"
	const envName = "my-mysql"

	site1 := testutil.NewTestStagingSite("site1", ns, nil)
	site1.Labels = map[string]string{
		labels.MysqlEnvironmentsPrefix + envName: "true",
	}
	site2 := testutil.NewTestStagingSite("site2", ns, nil)
	site2.Labels = map[string]string{
		labels.MysqlEnvironmentsPrefix + envName: "true",
	}

	handler := &MysqlConfigDeleteHandler{
		Client: testutil.NewFakeClient(site1, site2),
	}

	resp := handler.Handle(context.Background(), makeDeleteAdmissionRequest(ns, envName))

	if resp.Allowed {
		t.Error("expected Denied when multiple sites use the mysql environment, got Allowed")
	}
}

// TestMysqlConfigDeleteHandler_NoReferences tests that the label-only check passes
// when no sites reference the environment.
//
// NOTE: The handler also performs a MatchingFields query on ServiceConfigs which is
// not supported by the fake client. Tests that would reach that code path are
// omitted; those scenarios are covered by integration tests using envtest.
func TestMysqlConfigDeleteHandler_NoReferences(t *testing.T) {
	t.Skip("MatchingFields on ServiceConfig is not supported by the fake client; " +
		"covered by integration tests")
}
