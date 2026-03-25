package site

import (
	"context"
	"testing"

	sitev1 "github.com/szeber/kube-stager/apis/site/v1"
	"github.com/szeber/kube-stager/internal/testutil"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestNetworkingHandler_EnsureNetworkingObjectsAreUpToDate_CreatesService(t *testing.T) {
	ctx := context.Background()
	const (
		siteName  = "test-site"
		svcName   = "my-service"
		shortName = "svc"
		namespace = "default"
	)

	sc := testutil.NewTestServiceConfigWithDefaults(svcName, namespace, shortName)

	site := testutil.NewTestStagingSite(siteName, namespace, map[string]sitev1.StagingSiteService{
		svcName: {ImageTag: "latest", Replicas: 1},
	})
	site.Status.Enabled = true

	fakeClient := testutil.NewFakeClient(site, sc)
	scheme := testutil.NewTestScheme()

	handler := NetworkingHandler{
		Reader: fakeClient,
		Writer: fakeClient,
		Scheme: scheme,
	}

	changed, err := handler.EnsureNetworkingObjectsAreUpToDate(site, ctx)
	if err != nil {
		t.Fatalf("EnsureNetworkingObjectsAreUpToDate returned unexpected error: %v", err)
	}

	if !changed {
		t.Error("expected changed=true when NetworkingObjectsAreCreated transitions from false to true")
	}

	if !site.Status.NetworkingObjectsAreCreated {
		t.Error("expected site.Status.NetworkingObjectsAreCreated to be true after handler ran")
	}

	var svcList corev1.ServiceList
	if err := fakeClient.List(ctx, &svcList, client.InNamespace(namespace), client.MatchingLabels{
		"operator.kube-stager.io/site": siteName,
	}); err != nil {
		t.Fatalf("failed to list Services: %v", err)
	}

	if len(svcList.Items) == 0 {
		t.Fatal("expected at least one k8s Service to be created, got none")
	}

	found := false
	for _, svc := range svcList.Items {
		if svc.Labels["operator.kube-stager.io/service"] == svcName {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("k8s Service with service label=%s not found", svcName)
	}
}

func TestNetworkingHandler_EnsureNetworkingObjectsAreUpToDate_NoServiceWhenServiceSpecNil(t *testing.T) {
	ctx := context.Background()
	const (
		siteName  = "test-site"
		svcName   = "my-service"
		shortName = "svc"
		namespace = "default"
	)

	// NewTestServiceConfig does not set ServiceSpec, so no k8s Service should be created.
	sc := testutil.NewTestServiceConfig(svcName, namespace, shortName)

	site := testutil.NewTestStagingSite(siteName, namespace, map[string]sitev1.StagingSiteService{
		svcName: {ImageTag: "latest", Replicas: 1},
	})
	site.Status.Enabled = true

	fakeClient := testutil.NewFakeClient(site, sc)
	scheme := testutil.NewTestScheme()

	handler := NetworkingHandler{
		Reader: fakeClient,
		Writer: fakeClient,
		Scheme: scheme,
	}

	_, err := handler.EnsureNetworkingObjectsAreUpToDate(site, ctx)
	if err != nil {
		t.Fatalf("EnsureNetworkingObjectsAreUpToDate returned unexpected error: %v", err)
	}

	var svcList corev1.ServiceList
	if err := fakeClient.List(ctx, &svcList, client.InNamespace(namespace), client.MatchingLabels{
		"operator.kube-stager.io/site": siteName,
	}); err != nil {
		t.Fatalf("failed to list Services: %v", err)
	}

	if len(svcList.Items) != 0 {
		t.Errorf("expected no k8s Services when ServiceSpec is nil, but found %d", len(svcList.Items))
	}
}

func TestNetworkingHandler_EnsureNetworkingObjectsAreUpToDate_DisabledSiteDeletesServices(t *testing.T) {
	ctx := context.Background()
	const (
		siteName  = "test-site"
		svcName   = "my-service"
		shortName = "svc"
		namespace = "default"
	)

	sc := testutil.NewTestServiceConfigWithDefaults(svcName, namespace, shortName)

	site := testutil.NewTestStagingSite(siteName, namespace, map[string]sitev1.StagingSiteService{
		svcName: {ImageTag: "latest", Replicas: 1},
	})
	// Status.Enabled=false means site is disabled; services/ingresses should be deleted.
	site.Status.Enabled = false

	// Pre-create a Service that should be deleted when the site is disabled.
	existingSvc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      siteName + "-" + shortName,
			Namespace: namespace,
			Labels: map[string]string{
				"operator.kube-stager.io/site":    siteName,
				"operator.kube-stager.io/service": svcName,
			},
		},
	}

	fakeClient := testutil.NewFakeClient(site, sc, existingSvc)
	scheme := testutil.NewTestScheme()

	handler := NetworkingHandler{
		Reader: fakeClient,
		Writer: fakeClient,
		Scheme: scheme,
	}

	_, err := handler.EnsureNetworkingObjectsAreUpToDate(site, ctx)
	if err != nil {
		t.Fatalf("EnsureNetworkingObjectsAreUpToDate returned unexpected error: %v", err)
	}

	var svcList corev1.ServiceList
	if err := fakeClient.List(ctx, &svcList, client.InNamespace(namespace), client.MatchingLabels{
		"operator.kube-stager.io/site": siteName,
	}); err != nil {
		t.Fatalf("failed to list Services: %v", err)
	}

	if len(svcList.Items) != 0 {
		t.Errorf("expected all Services to be deleted for disabled site, but found %d", len(svcList.Items))
	}
}

func TestNetworkingHandler_EnsureNetworkingObjectsAreUpToDate_DisabledSiteDeletesIngresses(t *testing.T) {
	ctx := context.Background()
	const (
		siteName  = "test-site"
		svcName   = "my-service"
		shortName = "svc"
		namespace = "default"
	)

	sc := testutil.NewTestServiceConfigWithDefaults(svcName, namespace, shortName)

	site := testutil.NewTestStagingSite(siteName, namespace, map[string]sitev1.StagingSiteService{
		svcName: {ImageTag: "latest", Replicas: 1},
	})
	site.Status.Enabled = false

	// Pre-create an Ingress that should be deleted when the site is disabled.
	existingIngress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      siteName + "-" + shortName,
			Namespace: namespace,
			Labels: map[string]string{
				"operator.kube-stager.io/site":    siteName,
				"operator.kube-stager.io/service": svcName,
			},
		},
	}

	fakeClient := testutil.NewFakeClient(site, sc, existingIngress)
	scheme := testutil.NewTestScheme()

	handler := NetworkingHandler{
		Reader: fakeClient,
		Writer: fakeClient,
		Scheme: scheme,
	}

	_, err := handler.EnsureNetworkingObjectsAreUpToDate(site, ctx)
	if err != nil {
		t.Fatalf("EnsureNetworkingObjectsAreUpToDate returned unexpected error: %v", err)
	}

	var ingressList networkingv1.IngressList
	if err := fakeClient.List(ctx, &ingressList, client.InNamespace(namespace), client.MatchingLabels{
		"operator.kube-stager.io/site": siteName,
	}); err != nil {
		t.Fatalf("failed to list Ingresses: %v", err)
	}

	if len(ingressList.Items) != 0 {
		t.Errorf("expected all Ingresses to be deleted for disabled site, but found %d", len(ingressList.Items))
	}
}

func TestNetworkingHandler_EnsureNetworkingObjectsAreUpToDate_IngressAnnotationsApplied(t *testing.T) {
	ctx := context.Background()
	const (
		siteName  = "test-site"
		svcName   = "my-service"
		shortName = "svc"
		namespace = "default"
	)

	sc := testutil.NewTestServiceConfigWithDefaults(svcName, namespace, shortName)
	pathType := networkingv1.PathTypePrefix
	sc.Spec.IngressSpec = &networkingv1.IngressSpec{
		Rules: []networkingv1.IngressRule{
			{
				Host: "example.com",
				IngressRuleValue: networkingv1.IngressRuleValue{
					HTTP: &networkingv1.HTTPIngressRuleValue{
						Paths: []networkingv1.HTTPIngressPath{
							{
								Path:     "/",
								PathType: &pathType,
								Backend: networkingv1.IngressBackend{
									Service: &networkingv1.IngressServiceBackend{
										Name: "placeholder",
										Port: networkingv1.ServiceBackendPort{Number: 80},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	sc.Spec.IngressAnnotations = map[string]string{
		"nginx.ingress.kubernetes.io/ssl-redirect":   "true",
		"nginx.ingress.kubernetes.io/rewrite-target": "/",
	}

	site := testutil.NewTestStagingSite(siteName, namespace, map[string]sitev1.StagingSiteService{
		svcName: {ImageTag: "latest", Replicas: 1},
	})
	site.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "site.operator.kube-stager.io",
		Version: "v1",
		Kind:    "StagingSite",
	})
	site.Status.Enabled = true

	fakeClient := testutil.NewFakeClient(site, sc)
	scheme := testutil.NewTestScheme()

	handler := NetworkingHandler{
		Reader: fakeClient,
		Writer: fakeClient,
		Scheme: scheme,
	}

	_, err := handler.EnsureNetworkingObjectsAreUpToDate(site, ctx)
	if err != nil {
		t.Fatalf("EnsureNetworkingObjectsAreUpToDate returned unexpected error: %v", err)
	}

	var ingressList networkingv1.IngressList
	if err := fakeClient.List(ctx, &ingressList, client.InNamespace(namespace), client.MatchingLabels{
		"operator.kube-stager.io/site": siteName,
	}); err != nil {
		t.Fatalf("failed to list Ingresses: %v", err)
	}

	if len(ingressList.Items) == 0 {
		t.Fatal("expected at least one Ingress to be created, got none")
	}

	ingress := ingressList.Items[0]
	if ingress.Annotations["nginx.ingress.kubernetes.io/ssl-redirect"] != "true" {
		t.Errorf("expected ssl-redirect annotation to be 'true', got %q", ingress.Annotations["nginx.ingress.kubernetes.io/ssl-redirect"])
	}
	if ingress.Annotations["nginx.ingress.kubernetes.io/rewrite-target"] != "/" {
		t.Errorf("expected rewrite-target annotation to be '/', got %q", ingress.Annotations["nginx.ingress.kubernetes.io/rewrite-target"])
	}
}

func TestNetworkingHandler_EnsureNetworkingObjectsAreUpToDate_NoChangeWhenAlreadyComplete(t *testing.T) {
	ctx := context.Background()
	const (
		siteName  = "test-site"
		svcName   = "my-service"
		shortName = "svc"
		namespace = "default"
	)

	sc := testutil.NewTestServiceConfigWithDefaults(svcName, namespace, shortName)

	site := testutil.NewTestStagingSite(siteName, namespace, map[string]sitev1.StagingSiteService{
		svcName: {ImageTag: "latest", Replicas: 1},
	})
	site.Status.Enabled = true
	// Pre-set NetworkingObjectsAreCreated so the handler should report no change.
	site.Status.NetworkingObjectsAreCreated = true

	fakeClient := testutil.NewFakeClient(site, sc)
	scheme := testutil.NewTestScheme()

	handler := NetworkingHandler{
		Reader: fakeClient,
		Writer: fakeClient,
		Scheme: scheme,
	}

	changed, err := handler.EnsureNetworkingObjectsAreUpToDate(site, ctx)
	if err != nil {
		t.Fatalf("EnsureNetworkingObjectsAreUpToDate returned unexpected error: %v", err)
	}

	if changed {
		t.Error("expected changed=false when NetworkingObjectsAreCreated was already true and remains true")
	}
}
