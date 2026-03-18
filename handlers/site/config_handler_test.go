package site

import (
	"context"
	"testing"

	configv1 "github.com/szeber/kube-stager/apis/config/v1"
	sitev1 "github.com/szeber/kube-stager/apis/site/v1"
	"github.com/szeber/kube-stager/internal/testutil"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestConfigHandler_EnsureConfigsAreUpToDate_CreatesConfigMap(t *testing.T) {
	ctx := context.Background()
	const (
		siteName  = "test-site"
		svcName   = "my-service"
		shortName = "svc"
		namespace = "default"
		cmType    = "env"
	)

	sc := testutil.NewTestServiceConfig(svcName, namespace, shortName)
	sc.Spec.ConfigMaps = map[string]configv1.Configmap{
		cmType: {"KEY": "value"},
	}

	site := testutil.NewTestStagingSite(siteName, namespace, map[string]sitev1.StagingSiteService{
		svcName: {ImageTag: "latest", Replicas: 1},
	})

	fakeClient := testutil.NewFakeClient(site, sc)
	scheme := testutil.NewTestScheme()

	handler := ConfigHandler{
		Reader: fakeClient,
		Writer: fakeClient,
		Scheme: scheme,
	}

	changed, err := handler.EnsureConfigsAreUpToDate(site, ctx)
	if err != nil {
		t.Fatalf("EnsureConfigsAreUpToDate returned unexpected error: %v", err)
	}

	if !changed {
		t.Error("expected changed=true when ConfigsAreCreated transitions from false to true")
	}

	if !site.Status.ConfigsAreCreated {
		t.Error("expected site.Status.ConfigsAreCreated to be true after handler ran")
	}

	var cmList corev1.ConfigMapList
	if err := fakeClient.List(ctx, &cmList, client.InNamespace(namespace)); err != nil {
		t.Fatalf("failed to list ConfigMaps: %v", err)
	}

	if len(cmList.Items) == 0 {
		t.Fatal("expected at least one ConfigMap to be created, got none")
	}

	found := false
	for _, cm := range cmList.Items {
		if cm.Labels["operator.kube-stager.io/site"] == siteName &&
			cm.Labels["operator.kube-stager.io/service"] == svcName &&
			cm.Labels["operator.kube-stager.io/type"] == cmType {
			found = true
			if cm.Data["KEY"] != "value" {
				t.Errorf("expected ConfigMap data KEY=value, got %v", cm.Data)
			}
			break
		}
	}
	if !found {
		t.Errorf("ConfigMap with site=%s service=%s type=%s not found among %d items", siteName, svcName, cmType, len(cmList.Items))
	}
}

func TestConfigHandler_EnsureConfigsAreUpToDate_NoChangeWhenAlreadyComplete(t *testing.T) {
	ctx := context.Background()
	const (
		siteName  = "test-site"
		svcName   = "my-service"
		shortName = "svc"
		namespace = "default"
		cmType    = "env"
	)

	sc := testutil.NewTestServiceConfig(svcName, namespace, shortName)
	sc.Spec.ConfigMaps = map[string]configv1.Configmap{
		cmType: {"KEY": "value"},
	}

	site := testutil.NewTestStagingSite(siteName, namespace, map[string]sitev1.StagingSiteService{
		svcName: {ImageTag: "latest", Replicas: 1},
	})
	// Pre-set ConfigsAreCreated so the status does not change.
	site.Status.ConfigsAreCreated = true

	fakeClient := testutil.NewFakeClient(site, sc)
	scheme := testutil.NewTestScheme()

	handler := ConfigHandler{
		Reader: fakeClient,
		Writer: fakeClient,
		Scheme: scheme,
	}

	changed, err := handler.EnsureConfigsAreUpToDate(site, ctx)
	if err != nil {
		t.Fatalf("EnsureConfigsAreUpToDate returned unexpected error: %v", err)
	}

	if changed {
		t.Error("expected changed=false when ConfigsAreCreated was already true and remains true")
	}

	if !site.Status.ConfigsAreCreated {
		t.Error("expected site.Status.ConfigsAreCreated to remain true")
	}
}

func TestConfigHandler_EnsureConfigsAreUpToDate_DeletesStaleConfigMap(t *testing.T) {
	ctx := context.Background()
	const (
		siteName  = "test-site"
		svcName   = "my-service"
		shortName = "svc"
		namespace = "default"
	)

	sc := testutil.NewTestServiceConfig(svcName, namespace, shortName)
	// No ConfigMaps defined — any pre-existing one should be deleted.

	site := testutil.NewTestStagingSite(siteName, namespace, map[string]sitev1.StagingSiteService{
		svcName: {ImageTag: "latest", Replicas: 1},
	})

	// Create a stale ConfigMap that should be cleaned up.
	stale := &corev1.ConfigMap{}
	stale.Name = "stale-cm"
	stale.Namespace = namespace
	stale.Labels = map[string]string{
		"operator.kube-stager.io/site":    siteName,
		"operator.kube-stager.io/service": svcName,
		"operator.kube-stager.io/type":    "old-type",
	}

	fakeClient := testutil.NewFakeClient(site, sc, stale)
	scheme := testutil.NewTestScheme()

	handler := ConfigHandler{
		Reader: fakeClient,
		Writer: fakeClient,
		Scheme: scheme,
	}

	_, err := handler.EnsureConfigsAreUpToDate(site, ctx)
	if err != nil {
		t.Fatalf("EnsureConfigsAreUpToDate returned unexpected error: %v", err)
	}

	var cmList corev1.ConfigMapList
	if err := fakeClient.List(ctx, &cmList, client.InNamespace(namespace), client.MatchingLabels{
		"operator.kube-stager.io/site": siteName,
	}); err != nil {
		t.Fatalf("failed to list ConfigMaps: %v", err)
	}

	if len(cmList.Items) != 0 {
		t.Errorf("expected stale ConfigMap to be deleted, but found %d items", len(cmList.Items))
	}
}

func TestConfigHandler_EnsureConfigsAreUpToDate_UpdatesConfigMap(t *testing.T) {
	ctx := context.Background()
	const (
		siteName  = "test-site"
		svcName   = "my-service"
		shortName = "svc"
		namespace = "default"
		cmType    = "env"
	)

	sc := testutil.NewTestServiceConfig(svcName, namespace, shortName)
	sc.Spec.ConfigMaps = map[string]configv1.Configmap{
		cmType: {"KEY": "new-value"},
	}

	site := testutil.NewTestStagingSite(siteName, namespace, map[string]sitev1.StagingSiteService{
		svcName: {ImageTag: "latest", Replicas: 1},
	})

	// Pre-create the ConfigMap with stale data.
	existing := &corev1.ConfigMap{}
	existing.Name = siteName + "-" + shortName + "-" + cmType
	existing.Namespace = namespace
	existing.Labels = map[string]string{
		"operator.kube-stager.io/site":    siteName,
		"operator.kube-stager.io/service": svcName,
		"operator.kube-stager.io/type":    cmType,
	}
	existing.Data = map[string]string{"KEY": "old-value"}

	fakeClient := testutil.NewFakeClient(site, sc, existing)
	scheme := testutil.NewTestScheme()

	handler := ConfigHandler{
		Reader: fakeClient,
		Writer: fakeClient,
		Scheme: scheme,
	}

	_, err := handler.EnsureConfigsAreUpToDate(site, ctx)
	if err != nil {
		t.Fatalf("EnsureConfigsAreUpToDate returned unexpected error: %v", err)
	}

	var cmList corev1.ConfigMapList
	if err := fakeClient.List(ctx, &cmList, client.InNamespace(namespace), client.MatchingLabels{
		"operator.kube-stager.io/site":    siteName,
		"operator.kube-stager.io/service": svcName,
		"operator.kube-stager.io/type":    cmType,
	}); err != nil {
		t.Fatalf("failed to list ConfigMaps: %v", err)
	}

	if len(cmList.Items) == 0 {
		t.Fatal("expected ConfigMap to still exist after update")
	}

	if cmList.Items[0].Data["KEY"] != "new-value" {
		t.Errorf("expected ConfigMap data KEY=new-value after update, got %q", cmList.Items[0].Data["KEY"])
	}
}
