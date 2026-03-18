package webhook

import (
	"context"
	"net/http"
	"testing"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	sitev1 "github.com/szeber/kube-stager/apis/site/v1"
	"github.com/szeber/kube-stager/helpers/labels"
	"github.com/szeber/kube-stager/internal/testutil"
)

func makeDeleteAdmissionRequest(namespace, name string) admission.Request {
	return admission.Request{
		AdmissionRequest: admissionv1.AdmissionRequest{
			Namespace: namespace,
			Name:      name,
		},
	}
}

func TestServiceConfigDeleteHandler_NoSitesUsingService(t *testing.T) {
	const ns = "test-ns"
	const serviceName = "mysvc"

	handler := &ServiceConfigDeleteHandler{
		Client: testutil.NewFakeClient(),
	}

	resp := handler.Handle(context.Background(), makeDeleteAdmissionRequest(ns, serviceName))

	if !resp.Allowed {
		t.Errorf("expected Allowed when no sites use the service, got Denied: %v", resp.Result)
	}
}

func TestServiceConfigDeleteHandler_SitesUsingService(t *testing.T) {
	const ns = "test-ns"
	const serviceName = "mysvc"

	site := testutil.NewTestStagingSite("mysite", ns, nil)
	site.Labels = map[string]string{
		labels.ServicesPrefix + serviceName: "true",
	}

	handler := &ServiceConfigDeleteHandler{
		Client: testutil.NewFakeClient(site),
	}

	resp := handler.Handle(context.Background(), makeDeleteAdmissionRequest(ns, serviceName))

	if resp.Allowed {
		t.Error("expected Denied when sites are using the service, got Allowed")
	}
	if resp.Result == nil || resp.Result.Code != http.StatusForbidden {
		t.Errorf("expected Forbidden (403), got: %v", resp.Result)
	}
}

func TestServiceConfigDeleteHandler_SiteInDifferentNamespaceNotBlocking(t *testing.T) {
	const ns = "test-ns"
	const otherNs = "other-ns"
	const serviceName = "mysvc"

	site := &sitev1.StagingSite{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mysite",
			Namespace: otherNs,
			Labels: map[string]string{
				labels.ServicesPrefix + serviceName: "true",
			},
		},
	}

	handler := &ServiceConfigDeleteHandler{
		Client: testutil.NewFakeClient(site),
	}

	resp := handler.Handle(context.Background(), makeDeleteAdmissionRequest(ns, serviceName))

	if !resp.Allowed {
		t.Errorf("expected Allowed when matching site is in a different namespace, got Denied: %v", resp.Result)
	}
}

func TestServiceConfigDeleteHandler_MultipleSitesUsingService(t *testing.T) {
	const ns = "test-ns"
	const serviceName = "mysvc"

	site1 := testutil.NewTestStagingSite("site1", ns, nil)
	site1.Labels = map[string]string{
		labels.ServicesPrefix + serviceName: "true",
	}
	site2 := testutil.NewTestStagingSite("site2", ns, nil)
	site2.Labels = map[string]string{
		labels.ServicesPrefix + serviceName: "true",
	}

	handler := &ServiceConfigDeleteHandler{
		Client: testutil.NewFakeClient(site1, site2),
	}

	resp := handler.Handle(context.Background(), makeDeleteAdmissionRequest(ns, serviceName))

	if resp.Allowed {
		t.Error("expected Denied when multiple sites use the service, got Allowed")
	}
}
