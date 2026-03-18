package job

import (
	"context"
	"testing"

	jobv1 "github.com/szeber/kube-stager/apis/job/v1"
	sitev1 "github.com/szeber/kube-stager/apis/site/v1"
	"github.com/szeber/kube-stager/helpers/labels"
	"github.com/szeber/kube-stager/internal/testutil"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func newDbInitJobHandler(objs ...client.Object) DbInitJobHandler {
	c := testutil.NewFakeClient(objs...)
	return DbInitJobHandler{
		Reader: c,
		Writer: c,
		Scheme: testutil.NewTestScheme(),
	}
}

func TestDbInitJobHandler_EnsureJobsAreCreated_NoServicesCreatesNoJobs(t *testing.T) {
	ctx := context.Background()
	site := testutil.NewTestStagingSite("mysite", "test-ns", nil)
	handler := newDbInitJobHandler(site)

	complete, err := handler.EnsureJobsAreCreated(site, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !complete {
		t.Error("expected complete=true when no jobs need to be created or deleted")
	}

	jobList := &jobv1.DbInitJobList{}
	err = handler.Reader.List(ctx, jobList, client.InNamespace("test-ns"))
	if err != nil {
		t.Fatalf("error listing db init jobs: %v", err)
	}
	if len(jobList.Items) != 0 {
		t.Errorf("expected 0 jobs, got %d", len(jobList.Items))
	}
}

func TestDbInitJobHandler_EnsureJobsAreCreated_ServiceWithNoDbEnvCreatesNoJob(t *testing.T) {
	ctx := context.Background()
	services := map[string]sitev1.StagingSiteService{
		"mysvc": {ImageTag: "v1", MysqlEnvironment: "", MongoEnvironment: ""},
	}
	site := testutil.NewTestStagingSite("mysite", "test-ns", services)
	svcConfig := testutil.NewTestServiceConfig("mysvc", "test-ns", "svc")
	handler := newDbInitJobHandler(site, svcConfig)

	complete, err := handler.EnsureJobsAreCreated(site, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !complete {
		t.Error("expected complete=true when no db environments are configured")
	}

	jobList := &jobv1.DbInitJobList{}
	_ = handler.Reader.List(ctx, jobList, client.InNamespace("test-ns"))
	if len(jobList.Items) != 0 {
		t.Errorf("expected 0 jobs, got %d", len(jobList.Items))
	}
}

func TestDbInitJobHandler_EnsureJobsAreCreated_ServiceWithMysqlEnvButNoDbInitPodSpec(t *testing.T) {
	ctx := context.Background()
	services := map[string]sitev1.StagingSiteService{
		"mysvc": {ImageTag: "v1", MysqlEnvironment: "mysql-env"},
	}
	site := testutil.NewTestStagingSite("mysite", "test-ns", services)
	// ServiceConfig without DbInitPodSpec
	svcConfig := testutil.NewTestServiceConfig("mysvc", "test-ns", "svc")
	handler := newDbInitJobHandler(site, svcConfig)

	complete, err := handler.EnsureJobsAreCreated(site, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !complete {
		t.Error("expected complete=true when no DbInitPodSpec is set")
	}

	jobList := &jobv1.DbInitJobList{}
	_ = handler.Reader.List(ctx, jobList, client.InNamespace("test-ns"))
	if len(jobList.Items) != 0 {
		t.Errorf("expected 0 jobs (no DbInitPodSpec), got %d", len(jobList.Items))
	}
}

func TestDbInitJobHandler_EnsureJobsAreCreated_ServiceWithMysqlEnvAndDbInitPodSpec(t *testing.T) {
	ctx := context.Background()
	services := map[string]sitev1.StagingSiteService{
		"mysvc": {ImageTag: "v1", MysqlEnvironment: "mysql-env"},
	}
	site := testutil.NewTestStagingSite("mysite", "test-ns", services)
	svcConfig := testutil.NewTestServiceConfig("mysvc", "test-ns", "svc")
	svcConfig.Spec.DbInitPodSpec = &corev1.PodSpec{
		Containers: []corev1.Container{
			{Name: "init", Image: "init:latest"},
		},
	}
	handler := newDbInitJobHandler(site, svcConfig)

	complete, err := handler.EnsureJobsAreCreated(site, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if complete {
		t.Error("expected complete=false after creating a new job")
	}

	jobList := &jobv1.DbInitJobList{}
	err = handler.Reader.List(ctx, jobList, client.InNamespace("test-ns"), client.MatchingLabels{labels.Site: "mysite"})
	if err != nil {
		t.Fatalf("error listing db init jobs: %v", err)
	}
	if len(jobList.Items) != 1 {
		t.Fatalf("expected 1 db init job, got %d", len(jobList.Items))
	}

	createdJob := jobList.Items[0]
	if createdJob.Spec.SiteName != "mysite" {
		t.Errorf("SiteName = %q, want %q", createdJob.Spec.SiteName, "mysite")
	}
	if createdJob.Spec.ServiceName != "mysvc" {
		t.Errorf("ServiceName = %q, want %q", createdJob.Spec.ServiceName, "mysvc")
	}
	if createdJob.Spec.MysqlEnvironment != "mysql-env" {
		t.Errorf("MysqlEnvironment = %q, want %q", createdJob.Spec.MysqlEnvironment, "mysql-env")
	}
}

func TestDbInitJobHandler_EnsureJobsAreCreated_ServiceWithMongoEnvAndDbInitPodSpec(t *testing.T) {
	ctx := context.Background()
	services := map[string]sitev1.StagingSiteService{
		"mysvc": {ImageTag: "v1", MongoEnvironment: "mongo-env"},
	}
	site := testutil.NewTestStagingSite("mysite", "test-ns", services)
	svcConfig := testutil.NewTestServiceConfig("mysvc", "test-ns", "svc")
	svcConfig.Spec.DbInitPodSpec = &corev1.PodSpec{
		Containers: []corev1.Container{
			{Name: "init", Image: "init:latest"},
		},
	}
	handler := newDbInitJobHandler(site, svcConfig)

	complete, err := handler.EnsureJobsAreCreated(site, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if complete {
		t.Error("expected complete=false after creating a new job")
	}

	jobList := &jobv1.DbInitJobList{}
	err = handler.Reader.List(ctx, jobList, client.InNamespace("test-ns"), client.MatchingLabels{labels.Site: "mysite"})
	if err != nil {
		t.Fatalf("error listing db init jobs: %v", err)
	}
	if len(jobList.Items) != 1 {
		t.Fatalf("expected 1 db init job, got %d", len(jobList.Items))
	}

	createdJob := jobList.Items[0]
	if createdJob.Spec.MongoEnvironment != "mongo-env" {
		t.Errorf("MongoEnvironment = %q, want %q", createdJob.Spec.MongoEnvironment, "mongo-env")
	}
}

func TestDbInitJobHandler_EnsureJobsAreCreated_ExistingJobMeansComplete(t *testing.T) {
	ctx := context.Background()
	services := map[string]sitev1.StagingSiteService{
		"mysvc": {ImageTag: "v1", MysqlEnvironment: "mysql-env"},
	}
	site := testutil.NewTestStagingSite("mysite", "test-ns", services)
	svcConfig := testutil.NewTestServiceConfig("mysvc", "test-ns", "svc")
	svcConfig.Spec.DbInitPodSpec = &corev1.PodSpec{
		Containers: []corev1.Container{
			{Name: "init", Image: "init:latest"},
		},
	}

	// Pre-existing job for this service
	existingJob := testutil.NewTestDbInitJob("mysite-svc", "test-ns", "mysite", "mysvc")
	existingJob.Labels = map[string]string{labels.Site: "mysite"}
	handler := newDbInitJobHandler(site, svcConfig, existingJob)

	complete, err := handler.EnsureJobsAreCreated(site, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !complete {
		t.Error("expected complete=true when job already exists")
	}
}

func TestDbInitJobHandler_EnsureJobsAreCreated_StaleJobIsDeleted(t *testing.T) {
	ctx := context.Background()
	// Site has no services needing db init
	site := testutil.NewTestStagingSite("mysite", "test-ns", nil)

	// But there is a stale job
	staleJob := testutil.NewTestDbInitJob("mysite-svc", "test-ns", "mysite", "oldsvc")
	staleJob.Labels = map[string]string{labels.Site: "mysite"}
	handler := newDbInitJobHandler(site, staleJob)

	complete, err := handler.EnsureJobsAreCreated(site, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if complete {
		t.Error("expected complete=false when a stale job was deleted")
	}

	jobList := &jobv1.DbInitJobList{}
	_ = handler.Reader.List(ctx, jobList, client.InNamespace("test-ns"), client.MatchingLabels{labels.Site: "mysite"})
	if len(jobList.Items) != 0 {
		t.Errorf("expected 0 jobs after deletion, got %d", len(jobList.Items))
	}
}

func TestDbInitJobHandler_EnsureJobsAreComplete_NoJobsReturnsTrue(t *testing.T) {
	ctx := context.Background()
	site := testutil.NewTestStagingSite("mysite", "test-ns", nil)
	handler := newDbInitJobHandler(site)

	complete, err := handler.EnsureJobsAreComplete(site, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !complete {
		t.Error("expected complete=true when there are no jobs")
	}
}

func TestDbInitJobHandler_EnsureJobsAreComplete_AllCompleteReturnsTrue(t *testing.T) {
	ctx := context.Background()
	site := testutil.NewTestStagingSite("mysite", "test-ns", nil)
	job1 := testutil.NewTestDbInitJob("mysite-svc1", "test-ns", "mysite", "svc1")
	job1.Labels = map[string]string{labels.Site: "mysite"}
	job1.Status.State = jobv1.Complete
	job2 := testutil.NewTestDbInitJob("mysite-svc2", "test-ns", "mysite", "svc2")
	job2.Labels = map[string]string{labels.Site: "mysite"}
	job2.Status.State = jobv1.Complete
	handler := newDbInitJobHandler(site, job1, job2)

	complete, err := handler.EnsureJobsAreComplete(site, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !complete {
		t.Error("expected complete=true when all jobs are Complete")
	}
}

func TestDbInitJobHandler_EnsureJobsAreComplete_PendingJobReturnsFalse(t *testing.T) {
	ctx := context.Background()
	site := testutil.NewTestStagingSite("mysite", "test-ns", nil)
	job1 := testutil.NewTestDbInitJob("mysite-svc1", "test-ns", "mysite", "svc1")
	job1.Labels = map[string]string{labels.Site: "mysite"}
	job1.Status.State = jobv1.Pending
	handler := newDbInitJobHandler(site, job1)

	complete, err := handler.EnsureJobsAreComplete(site, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if complete {
		t.Error("expected complete=false when a job is Pending")
	}
}

func TestDbInitJobHandler_EnsureJobsAreComplete_FailedJobReturnsError(t *testing.T) {
	ctx := context.Background()
	site := testutil.NewTestStagingSite("mysite", "test-ns", nil)
	job1 := testutil.NewTestDbInitJob("mysite-svc1", "test-ns", "mysite", "svc1")
	job1.Labels = map[string]string{labels.Site: "mysite"}
	job1.Status.State = jobv1.Failed
	handler := newDbInitJobHandler(site, job1)

	complete, err := handler.EnsureJobsAreComplete(site, ctx)
	if err == nil {
		t.Error("expected error when a job is Failed")
	}
	if complete {
		t.Error("expected complete=false when a job is Failed")
	}
}

func TestDbInitJobHandler_EnsureJobsAreComplete_MixedCompleteAndPendingReturnsFalse(t *testing.T) {
	ctx := context.Background()
	site := testutil.NewTestStagingSite("mysite", "test-ns", nil)
	job1 := testutil.NewTestDbInitJob("mysite-svc1", "test-ns", "mysite", "svc1")
	job1.Labels = map[string]string{labels.Site: "mysite"}
	job1.Status.State = jobv1.Complete
	job2 := testutil.NewTestDbInitJob("mysite-svc2", "test-ns", "mysite", "svc2")
	job2.Labels = map[string]string{labels.Site: "mysite"}
	job2.Status.State = jobv1.Pending
	handler := newDbInitJobHandler(site, job1, job2)

	complete, err := handler.EnsureJobsAreComplete(site, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if complete {
		t.Error("expected complete=false when at least one job is not Complete")
	}
}
