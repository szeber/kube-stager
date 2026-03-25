// Package metrics_test uses an external test package to avoid an import cycle:
// metrics_test -> testutil -> testutil/mocks -> handlers/database -> metrics.
// The scheme and fake client setup is duplicated here for this reason.
package metrics_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	configv1 "github.com/szeber/kube-stager/apis/config/v1"
	jobv1 "github.com/szeber/kube-stager/apis/job/v1"
	sitev1 "github.com/szeber/kube-stager/apis/site/v1"
	taskv1 "github.com/szeber/kube-stager/apis/task/v1"
	"github.com/szeber/kube-stager/internal/metrics"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(s))
	utilruntime.Must(configv1.AddToScheme(s))
	utilruntime.Must(jobv1.AddToScheme(s))
	utilruntime.Must(sitev1.AddToScheme(s))
	utilruntime.Must(taskv1.AddToScheme(s))
	return s
}

func newFakeClient(objs ...client.Object) client.Client {
	return fake.NewClientBuilder().
		WithScheme(newScheme()).
		WithObjects(objs...).
		WithStatusSubresource(
			&sitev1.StagingSite{},
			&taskv1.MysqlDatabase{},
			&taskv1.MongoDatabase{},
			&taskv1.RedisDatabase{},
			&jobv1.DbInitJob{},
			&jobv1.DbMigrationJob{},
			&jobv1.Backup{},
		).
		Build()
}

func collectMetrics(t *testing.T, collector *metrics.ResourceCollector) map[string][]*dto.Metric {
	t.Helper()

	ch := make(chan prometheus.Metric, 100)
	go func() {
		collector.Collect(ch)
		close(ch)
	}()

	result := map[string][]*dto.Metric{}
	for m := range ch {
		metric := &dto.Metric{}
		if err := m.Write(metric); err != nil {
			t.Fatalf("failed to write metric: %v", err)
		}
		desc := m.Desc().String()
		for _, name := range []string{"kube_stager_staging_sites", "kube_stager_databases", "kube_stager_jobs"} {
			if strings.Contains(desc, name) {
				result[name] = append(result[name], metric)
				break
			}
		}
	}

	return result
}

func getLabelValue(metric *dto.Metric, name string) string {
	for _, lp := range metric.GetLabel() {
		if lp.GetName() == name {
			return lp.GetValue()
		}
	}
	return ""
}

func TestCollector_Describe(t *testing.T) {
	collector := metrics.NewResourceCollector(newFakeClient())
	ch := make(chan *prometheus.Desc, 10)
	go func() {
		collector.Describe(ch)
		close(ch)
	}()

	var descs []*prometheus.Desc
	for d := range ch {
		descs = append(descs, d)
	}

	if len(descs) != 3 {
		t.Errorf("expected 3 descriptors, got %d", len(descs))
	}
}

func TestCollector_EmptyCluster(t *testing.T) {
	collector := metrics.NewResourceCollector(newFakeClient())
	m := collectMetrics(t, collector)

	if len(m["kube_stager_staging_sites"]) != 0 {
		t.Errorf("expected 0 staging site metrics, got %d", len(m["kube_stager_staging_sites"]))
	}
	if len(m["kube_stager_databases"]) != 0 {
		t.Errorf("expected 0 database metrics, got %d", len(m["kube_stager_databases"]))
	}
	if len(m["kube_stager_jobs"]) != 0 {
		t.Errorf("expected 0 job metrics, got %d", len(m["kube_stager_jobs"]))
	}
}

func TestCollector_StagingSites(t *testing.T) {
	site1 := &sitev1.StagingSite{
		ObjectMeta: metav1.ObjectMeta{Name: "site1", Namespace: "default"},
		Status:     sitev1.StagingSiteStatus{State: sitev1.StateComplete, Enabled: true},
	}
	site2 := &sitev1.StagingSite{
		ObjectMeta: metav1.ObjectMeta{Name: "site2", Namespace: "default"},
		Status:     sitev1.StagingSiteStatus{State: sitev1.StateComplete, Enabled: true},
	}
	site3 := &sitev1.StagingSite{
		ObjectMeta: metav1.ObjectMeta{Name: "site3", Namespace: "default"},
		Status:     sitev1.StagingSiteStatus{State: sitev1.StateFailed, Enabled: false},
	}
	site4 := &sitev1.StagingSite{
		ObjectMeta: metav1.ObjectMeta{Name: "site4", Namespace: "other-ns"},
		Status:     sitev1.StagingSiteStatus{State: sitev1.StatePending, Enabled: true},
	}

	fakeClient := newFakeClient(site1, site2, site3, site4)
	collector := metrics.NewResourceCollector(fakeClient)
	m := collectMetrics(t, collector)

	siteMetrics := m["kube_stager_staging_sites"]
	if len(siteMetrics) != 3 {
		t.Fatalf("expected 3 site metric series, got %d", len(siteMetrics))
	}

	found := map[string]float64{}
	for _, metric := range siteMetrics {
		key := getLabelValue(metric, "namespace") + "/" + getLabelValue(metric, "state") + "/" + getLabelValue(metric, "enabled")
		found[key] = metric.GetGauge().GetValue()
	}

	if found["default/Complete/true"] != 2 {
		t.Errorf("expected 2 Complete/true in default, got %v", found["default/Complete/true"])
	}
	if found["default/Failed/false"] != 1 {
		t.Errorf("expected 1 Failed/false in default, got %v", found["default/Failed/false"])
	}
	if found["other-ns/Pending/true"] != 1 {
		t.Errorf("expected 1 Pending/true in other-ns, got %v", found["other-ns/Pending/true"])
	}
}

func TestCollector_Databases(t *testing.T) {
	mysql1 := &taskv1.MysqlDatabase{
		ObjectMeta: metav1.ObjectMeta{Name: "mysql1", Namespace: "default"},
		Status:     taskv1.TaskStatus{State: taskv1.Complete},
	}
	mysql2 := &taskv1.MysqlDatabase{
		ObjectMeta: metav1.ObjectMeta{Name: "mysql2", Namespace: "default"},
		Status:     taskv1.TaskStatus{State: taskv1.Complete},
	}
	mongo1 := &taskv1.MongoDatabase{
		ObjectMeta: metav1.ObjectMeta{Name: "mongo1", Namespace: "default"},
		Status:     taskv1.TaskStatus{State: taskv1.Pending},
	}
	redis1 := &taskv1.RedisDatabase{
		ObjectMeta: metav1.ObjectMeta{Name: "redis1", Namespace: "default"},
		Status:     taskv1.TaskStatus{State: taskv1.Failed},
	}

	fakeClient := newFakeClient(mysql1, mysql2, mongo1, redis1)
	collector := metrics.NewResourceCollector(fakeClient)
	m := collectMetrics(t, collector)

	dbMetrics := m["kube_stager_databases"]
	if len(dbMetrics) != 3 {
		t.Fatalf("expected 3 database metric series, got %d", len(dbMetrics))
	}

	found := map[string]float64{}
	for _, metric := range dbMetrics {
		key := getLabelValue(metric, "type") + "/" + getLabelValue(metric, "state")
		found[key] = metric.GetGauge().GetValue()
	}

	if found["mysql/Complete"] != 2 {
		t.Errorf("expected 2 mysql/Complete, got %v", found["mysql/Complete"])
	}
	if found["mongo/Pending"] != 1 {
		t.Errorf("expected 1 mongo/Pending, got %v", found["mongo/Pending"])
	}
	if found["redis/Failed"] != 1 {
		t.Errorf("expected 1 redis/Failed, got %v", found["redis/Failed"])
	}
}

func TestCollector_Jobs(t *testing.T) {
	initJob := &jobv1.DbInitJob{
		ObjectMeta: metav1.ObjectMeta{Name: "init1", Namespace: "default"},
		Status:     jobv1.DbInitJobStatus{State: jobv1.Running},
	}
	migrationJob := &jobv1.DbMigrationJob{
		ObjectMeta: metav1.ObjectMeta{Name: "migration1", Namespace: "default"},
		Status:     jobv1.DbMigrationJobStatus{State: jobv1.Complete},
	}
	backup1 := &jobv1.Backup{
		ObjectMeta: metav1.ObjectMeta{Name: "backup1", Namespace: "default"},
		Spec:       jobv1.BackupSpec{BackupType: jobv1.BackupTypeScheduled},
		Status:     jobv1.BackupStatus{BackupStatusDetail: jobv1.BackupStatusDetail{State: jobv1.Complete}},
	}
	backup2 := &jobv1.Backup{
		ObjectMeta: metav1.ObjectMeta{Name: "backup2", Namespace: "default"},
		Spec:       jobv1.BackupSpec{BackupType: jobv1.BackupTypeManual},
		Status:     jobv1.BackupStatus{BackupStatusDetail: jobv1.BackupStatusDetail{State: jobv1.Failed}},
	}

	fakeClient := newFakeClient(initJob, migrationJob, backup1, backup2)
	collector := metrics.NewResourceCollector(fakeClient)
	m := collectMetrics(t, collector)

	jobMetrics := m["kube_stager_jobs"]

	found := map[string]float64{}
	for _, metric := range jobMetrics {
		key := getLabelValue(metric, "kind") + "/" + getLabelValue(metric, "state")
		found[key] = metric.GetGauge().GetValue()
	}

	if found["dbinit/Running"] != 1 {
		t.Errorf("expected 1 dbinit/Running, got %v", found["dbinit/Running"])
	}
	if found["dbmigration/Complete"] != 1 {
		t.Errorf("expected 1 dbmigration/Complete, got %v", found["dbmigration/Complete"])
	}
	if found["backup/Complete"] != 1 {
		t.Errorf("expected 1 backup/Complete, got %v", found["backup/Complete"])
	}
	if found["backup/Failed"] != 1 {
		t.Errorf("expected 1 backup/Failed, got %v", found["backup/Failed"])
	}
}

func TestCollector_EmptyStateTreatedAsPending(t *testing.T) {
	site := &sitev1.StagingSite{
		ObjectMeta: metav1.ObjectMeta{Name: "site1", Namespace: "default"},
	}

	fakeClient := newFakeClient(site)
	collector := metrics.NewResourceCollector(fakeClient)
	m := collectMetrics(t, collector)

	siteMetrics := m["kube_stager_staging_sites"]
	if len(siteMetrics) != 1 {
		t.Fatalf("expected 1 site metric, got %d", len(siteMetrics))
	}

	if getLabelValue(siteMetrics[0], "state") != "Pending" {
		t.Errorf("expected empty state to be treated as Pending, got %s", getLabelValue(siteMetrics[0], "state"))
	}
}

func TestCollector_MultipleNamespaces(t *testing.T) {
	objs := []client.Object{
		&sitev1.StagingSite{
			ObjectMeta: metav1.ObjectMeta{Name: "site1", Namespace: "ns1"},
			Status:     sitev1.StagingSiteStatus{State: sitev1.StateComplete, Enabled: true},
		},
		&sitev1.StagingSite{
			ObjectMeta: metav1.ObjectMeta{Name: "site2", Namespace: "ns2"},
			Status:     sitev1.StagingSiteStatus{State: sitev1.StateComplete, Enabled: true},
		},
	}

	fakeClient := newFakeClient(objs...)
	collector := metrics.NewResourceCollector(fakeClient)
	m := collectMetrics(t, collector)

	siteMetrics := m["kube_stager_staging_sites"]
	if len(siteMetrics) != 2 {
		t.Fatalf("expected 2 site metrics (one per namespace), got %d", len(siteMetrics))
	}

	namespaces := map[string]bool{}
	for _, metric := range siteMetrics {
		namespaces[getLabelValue(metric, "namespace")] = true
	}

	if !namespaces["ns1"] || !namespaces["ns2"] {
		t.Errorf("expected metrics from both ns1 and ns2, got %v", namespaces)
	}
}

// failingReader is a client.Reader that returns an error on every List call.
type failingReader struct {
	client.Reader
}

func (f failingReader) List(_ context.Context, _ client.ObjectList, _ ...client.ListOption) error {
	return fmt.Errorf("injected list error")
}

func TestCollector_ListError_EmitsInvalidMetrics(t *testing.T) {
	collector := metrics.NewResourceCollector(failingReader{})

	ch := make(chan prometheus.Metric, 100)
	go func() {
		collector.Collect(ch)
		close(ch)
	}()

	invalidCount := 0
	for m := range ch {
		dto := &dto.Metric{}
		err := m.Write(dto)
		if err != nil {
			invalidCount++
		}
	}

	// Each of the 3 collect functions (sites, databases, jobs) should emit one InvalidMetric
	if invalidCount != 3 {
		t.Errorf("expected 3 invalid metrics from failing reader, got %d", invalidCount)
	}
}
