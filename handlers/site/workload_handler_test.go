package site

import (
	"context"
	"testing"

	sitev1 "github.com/szeber/kube-stager/apis/site/v1"
	"github.com/szeber/kube-stager/internal/testutil"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestWorkloadHandler_EnsureWorkloadObjectsAreUpToDate_CreatesDeployment(t *testing.T) {
	ctx := context.Background()
	const (
		siteName  = "test-site"
		svcName   = "my-service"
		shortName = "svc"
		namespace = "default"
	)

	sc := testutil.NewTestServiceConfig(svcName, namespace, shortName)

	site := testutil.NewTestStagingSite(siteName, namespace, map[string]sitev1.StagingSiteService{
		svcName: {ImageTag: "latest", Replicas: 1},
	})
	site.Status.Enabled = true

	fakeClient := testutil.NewFakeClient(site, sc)
	scheme := testutil.NewTestScheme()

	handler := WorkloadHandler{
		Reader: fakeClient,
		Writer: fakeClient,
		Scheme: scheme,
	}

	changed, err := handler.EnsureWorkloadObjectsAreUpToDate(site, ctx)
	if err != nil {
		t.Fatalf("EnsureWorkloadObjectsAreUpToDate returned unexpected error: %v", err)
	}

	if !changed {
		t.Error("expected changed=true when WorkloadsAreCreated transitions from false to true")
	}

	if !site.Status.WorkloadsAreCreated {
		t.Error("expected site.Status.WorkloadsAreCreated to be true after handler ran")
	}

	var depList appsv1.DeploymentList
	if err := fakeClient.List(ctx, &depList, client.InNamespace(namespace), client.MatchingLabels{
		"operator.kube-stager.io/site": siteName,
	}); err != nil {
		t.Fatalf("failed to list Deployments: %v", err)
	}

	if len(depList.Items) == 0 {
		t.Fatal("expected at least one Deployment to be created, got none")
	}

	found := false
	for _, dep := range depList.Items {
		if dep.Labels["operator.kube-stager.io/service"] == svcName {
			found = true
			if dep.Namespace != namespace {
				t.Errorf("expected Deployment namespace=%s, got %s", namespace, dep.Namespace)
			}
			break
		}
	}
	if !found {
		t.Errorf("Deployment with service label=%s not found", svcName)
	}
}

func TestWorkloadHandler_EnsureWorkloadObjectsAreUpToDate_SetsReplicaCount(t *testing.T) {
	ctx := context.Background()
	const (
		siteName  = "test-site"
		svcName   = "my-service"
		shortName = "svc"
		namespace = "default"
	)

	sc := testutil.NewTestServiceConfig(svcName, namespace, shortName)

	site := testutil.NewTestStagingSite(siteName, namespace, map[string]sitev1.StagingSiteService{
		svcName: {ImageTag: "latest", Replicas: 2},
	})
	site.Status.Enabled = true

	fakeClient := testutil.NewFakeClient(site, sc)
	scheme := testutil.NewTestScheme()

	handler := WorkloadHandler{
		Reader: fakeClient,
		Writer: fakeClient,
		Scheme: scheme,
	}

	_, err := handler.EnsureWorkloadObjectsAreUpToDate(site, ctx)
	if err != nil {
		t.Fatalf("EnsureWorkloadObjectsAreUpToDate returned unexpected error: %v", err)
	}

	var depList appsv1.DeploymentList
	if err := fakeClient.List(ctx, &depList, client.InNamespace(namespace), client.MatchingLabels{
		"operator.kube-stager.io/site":    siteName,
		"operator.kube-stager.io/service": svcName,
	}); err != nil {
		t.Fatalf("failed to list Deployments: %v", err)
	}

	if len(depList.Items) == 0 {
		t.Fatal("expected a Deployment to be created")
	}

	dep := depList.Items[0]
	if dep.Spec.Replicas == nil {
		t.Fatal("expected Deployment.Spec.Replicas to be set, got nil")
	}
	if *dep.Spec.Replicas != 2 {
		t.Errorf("expected Deployment replicas=2, got %d", *dep.Spec.Replicas)
	}
}

func TestWorkloadHandler_EnsureWorkloadObjectsAreUpToDate_DisabledSiteDeletesDeployments(t *testing.T) {
	ctx := context.Background()
	const (
		siteName  = "test-site"
		svcName   = "my-service"
		shortName = "svc"
		namespace = "default"
	)

	sc := testutil.NewTestServiceConfig(svcName, namespace, shortName)

	site := testutil.NewTestStagingSite(siteName, namespace, map[string]sitev1.StagingSiteService{
		svcName: {ImageTag: "latest", Replicas: 1},
	})
	// Status.Enabled=false means site is disabled; deployments should be deleted.
	site.Status.Enabled = false

	// Pre-create a Deployment that should be removed when the site is disabled.
	replicas := int32(1)
	existingDep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      siteName + "-" + shortName,
			Namespace: namespace,
			Labels: map[string]string{
				"operator.kube-stager.io/site":    siteName,
				"operator.kube-stager.io/service": svcName,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"operator.kube-stager.io/site":    siteName,
					"operator.kube-stager.io/service": svcName,
				},
			},
		},
	}

	fakeClient := testutil.NewFakeClient(site, sc, existingDep)
	scheme := testutil.NewTestScheme()

	handler := WorkloadHandler{
		Reader: fakeClient,
		Writer: fakeClient,
		Scheme: scheme,
	}

	_, err := handler.EnsureWorkloadObjectsAreUpToDate(site, ctx)
	if err != nil {
		t.Fatalf("EnsureWorkloadObjectsAreUpToDate returned unexpected error: %v", err)
	}

	var depList appsv1.DeploymentList
	if err := fakeClient.List(ctx, &depList, client.InNamespace(namespace), client.MatchingLabels{
		"operator.kube-stager.io/site": siteName,
	}); err != nil {
		t.Fatalf("failed to list Deployments: %v", err)
	}

	if len(depList.Items) != 0 {
		t.Errorf("expected all Deployments to be deleted for disabled site, but found %d", len(depList.Items))
	}
}

func TestWorkloadHandler_EnsureWorkloadObjectsAreUpToDate_DisabledSiteDoesNotCreateDeployments(t *testing.T) {
	ctx := context.Background()
	const (
		siteName  = "test-site"
		svcName   = "my-service"
		shortName = "svc"
		namespace = "default"
	)

	sc := testutil.NewTestServiceConfig(svcName, namespace, shortName)

	site := testutil.NewTestStagingSite(siteName, namespace, map[string]sitev1.StagingSiteService{
		svcName: {ImageTag: "latest", Replicas: 1},
	})
	site.Status.Enabled = false

	fakeClient := testutil.NewFakeClient(site, sc)
	scheme := testutil.NewTestScheme()

	handler := WorkloadHandler{
		Reader: fakeClient,
		Writer: fakeClient,
		Scheme: scheme,
	}

	_, err := handler.EnsureWorkloadObjectsAreUpToDate(site, ctx)
	if err != nil {
		t.Fatalf("EnsureWorkloadObjectsAreUpToDate returned unexpected error: %v", err)
	}

	var depList appsv1.DeploymentList
	if err := fakeClient.List(ctx, &depList, client.InNamespace(namespace), client.MatchingLabels{
		"operator.kube-stager.io/site": siteName,
	}); err != nil {
		t.Fatalf("failed to list Deployments: %v", err)
	}

	if len(depList.Items) != 0 {
		t.Errorf("expected no Deployments created for a disabled site, but found %d", len(depList.Items))
	}
}

func TestWorkloadHandler_EnsureWorkloadObjectsAreUpToDate_SetsWorkloadHealth(t *testing.T) {
	ctx := context.Background()
	const (
		siteName  = "test-site"
		svcName   = "my-service"
		shortName = "svc"
		namespace = "default"
	)

	sc := testutil.NewTestServiceConfig(svcName, namespace, shortName)

	site := testutil.NewTestStagingSite(siteName, namespace, map[string]sitev1.StagingSiteService{
		svcName: {ImageTag: "latest", Replicas: 1},
	})
	site.Status.Enabled = true

	fakeClient := testutil.NewFakeClient(site, sc)
	scheme := testutil.NewTestScheme()

	handler := WorkloadHandler{
		Reader: fakeClient,
		Writer: fakeClient,
		Scheme: scheme,
	}

	_, err := handler.EnsureWorkloadObjectsAreUpToDate(site, ctx)
	if err != nil {
		t.Fatalf("EnsureWorkloadObjectsAreUpToDate returned unexpected error: %v", err)
	}

	if site.Status.WorkloadHealth == "" {
		t.Error("expected site.Status.WorkloadHealth to be set after handler ran")
	}
}

func TestWorkloadHandler_EnsureWorkloadObjectsAreUpToDate_DefaultReplicaCountIsOne(t *testing.T) {
	ctx := context.Background()
	const (
		siteName  = "test-site"
		svcName   = "my-service"
		shortName = "svc"
		namespace = "default"
	)

	sc := testutil.NewTestServiceConfig(svcName, namespace, shortName)

	// Replicas=0 should default to 1.
	site := testutil.NewTestStagingSite(siteName, namespace, map[string]sitev1.StagingSiteService{
		svcName: {ImageTag: "latest", Replicas: 0},
	})
	site.Status.Enabled = true

	fakeClient := testutil.NewFakeClient(site, sc)
	scheme := testutil.NewTestScheme()

	handler := WorkloadHandler{
		Reader: fakeClient,
		Writer: fakeClient,
		Scheme: scheme,
	}

	_, err := handler.EnsureWorkloadObjectsAreUpToDate(site, ctx)
	if err != nil {
		t.Fatalf("EnsureWorkloadObjectsAreUpToDate returned unexpected error: %v", err)
	}

	var depList appsv1.DeploymentList
	if err := fakeClient.List(ctx, &depList, client.InNamespace(namespace), client.MatchingLabels{
		"operator.kube-stager.io/site":    siteName,
		"operator.kube-stager.io/service": svcName,
	}); err != nil {
		t.Fatalf("failed to list Deployments: %v", err)
	}

	if len(depList.Items) == 0 {
		t.Fatal("expected a Deployment to be created")
	}

	dep := depList.Items[0]
	if dep.Spec.Replicas == nil {
		t.Fatal("expected Deployment.Spec.Replicas to be set, got nil")
	}
	if *dep.Spec.Replicas != 1 {
		t.Errorf("expected default Deployment replicas=1 when site replicas=0, got %d", *dep.Spec.Replicas)
	}
}
