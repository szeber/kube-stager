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

func newDbMigrationJobHandler(objs ...client.Object) DbMigrationJobHandler {
	c := testutil.NewFakeClient(objs...)
	return DbMigrationJobHandler{
		Reader: c,
		Writer: c,
		Scheme: testutil.NewTestScheme(),
	}
}

func TestDbMigrationJobHandler_EnsureJobsAreCreated_SiteNotEnabledCreatesNoJobs(t *testing.T) {
	ctx := context.Background()
	services := map[string]sitev1.StagingSiteService{
		"mysvc": {ImageTag: "v1", MysqlEnvironment: "mysql-env"},
	}
	site := testutil.NewTestStagingSite("mysite", "test-ns", services)
	site.Status.Enabled = false
	svcConfig := testutil.NewTestServiceConfig("mysvc", "test-ns", "svc")
	svcConfig.Spec.MigrationJobPodSpec = &corev1.PodSpec{
		Containers: []corev1.Container{
			{Name: "migrate", Image: "migrate:latest"},
		},
	}
	handler := newDbMigrationJobHandler(site, svcConfig)

	complete, err := handler.EnsureJobsAreCreated(site, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !complete {
		t.Error("expected complete=true when site is not enabled (no jobs to create)")
	}

	jobList := &jobv1.DbMigrationJobList{}
	_ = handler.Reader.List(ctx, jobList, client.InNamespace("test-ns"))
	if len(jobList.Items) != 0 {
		t.Errorf("expected 0 jobs when site is not enabled, got %d", len(jobList.Items))
	}
}

func TestDbMigrationJobHandler_EnsureJobsAreCreated_NoServicesCreatesNoJobs(t *testing.T) {
	ctx := context.Background()
	site := testutil.NewTestStagingSite("mysite", "test-ns", nil)
	site.Status.Enabled = true
	handler := newDbMigrationJobHandler(site)

	complete, err := handler.EnsureJobsAreCreated(site, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !complete {
		t.Error("expected complete=true when there are no services")
	}

	jobList := &jobv1.DbMigrationJobList{}
	_ = handler.Reader.List(ctx, jobList, client.InNamespace("test-ns"))
	if len(jobList.Items) != 0 {
		t.Errorf("expected 0 jobs, got %d", len(jobList.Items))
	}
}

func TestDbMigrationJobHandler_EnsureJobsAreCreated_ServiceWithNoDbEnvCreatesNoJob(t *testing.T) {
	ctx := context.Background()
	services := map[string]sitev1.StagingSiteService{
		"mysvc": {ImageTag: "v1", MysqlEnvironment: "", MongoEnvironment: ""},
	}
	site := testutil.NewTestStagingSite("mysite", "test-ns", services)
	site.Status.Enabled = true
	svcConfig := testutil.NewTestServiceConfig("mysvc", "test-ns", "svc")
	svcConfig.Spec.MigrationJobPodSpec = &corev1.PodSpec{
		Containers: []corev1.Container{
			{Name: "migrate", Image: "migrate:latest"},
		},
	}
	handler := newDbMigrationJobHandler(site, svcConfig)

	complete, err := handler.EnsureJobsAreCreated(site, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !complete {
		t.Error("expected complete=true when no db environments are configured")
	}

	jobList := &jobv1.DbMigrationJobList{}
	_ = handler.Reader.List(ctx, jobList, client.InNamespace("test-ns"))
	if len(jobList.Items) != 0 {
		t.Errorf("expected 0 jobs, got %d", len(jobList.Items))
	}
}

func TestDbMigrationJobHandler_EnsureJobsAreCreated_ServiceWithMysqlEnvButNoMigrationPodSpec(t *testing.T) {
	ctx := context.Background()
	services := map[string]sitev1.StagingSiteService{
		"mysvc": {ImageTag: "v1", MysqlEnvironment: "mysql-env"},
	}
	site := testutil.NewTestStagingSite("mysite", "test-ns", services)
	site.Status.Enabled = true
	// ServiceConfig without MigrationJobPodSpec
	svcConfig := testutil.NewTestServiceConfig("mysvc", "test-ns", "svc")
	handler := newDbMigrationJobHandler(site, svcConfig)

	complete, err := handler.EnsureJobsAreCreated(site, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !complete {
		t.Error("expected complete=true when no MigrationJobPodSpec is set")
	}

	jobList := &jobv1.DbMigrationJobList{}
	_ = handler.Reader.List(ctx, jobList, client.InNamespace("test-ns"))
	if len(jobList.Items) != 0 {
		t.Errorf("expected 0 jobs (no MigrationJobPodSpec), got %d", len(jobList.Items))
	}
}

func TestDbMigrationJobHandler_EnsureJobsAreCreated_CreatesJobForEnabledSiteWithMysqlEnv(t *testing.T) {
	ctx := context.Background()
	services := map[string]sitev1.StagingSiteService{
		"mysvc": {ImageTag: "v1.2", MysqlEnvironment: "mysql-env"},
	}
	site := testutil.NewTestStagingSite("mysite", "test-ns", services)
	site.Status.Enabled = true
	svcConfig := testutil.NewTestServiceConfig("mysvc", "test-ns", "svc")
	svcConfig.Spec.MigrationJobPodSpec = &corev1.PodSpec{
		Containers: []corev1.Container{
			{Name: "migrate", Image: "migrate:latest"},
		},
	}
	handler := newDbMigrationJobHandler(site, svcConfig)

	complete, err := handler.EnsureJobsAreCreated(site, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if complete {
		t.Error("expected complete=false after creating a new job")
	}

	jobList := &jobv1.DbMigrationJobList{}
	err = handler.Reader.List(ctx, jobList, client.InNamespace("test-ns"), client.MatchingLabels{labels.Site: "mysite"})
	if err != nil {
		t.Fatalf("error listing db migration jobs: %v", err)
	}
	if len(jobList.Items) != 1 {
		t.Fatalf("expected 1 db migration job, got %d", len(jobList.Items))
	}

	createdJob := jobList.Items[0]
	if createdJob.Spec.SiteName != "mysite" {
		t.Errorf("SiteName = %q, want %q", createdJob.Spec.SiteName, "mysite")
	}
	if createdJob.Spec.ServiceName != "mysvc" {
		t.Errorf("ServiceName = %q, want %q", createdJob.Spec.ServiceName, "mysvc")
	}
	if createdJob.Spec.ImageTag != "v1.2" {
		t.Errorf("ImageTag = %q, want %q", createdJob.Spec.ImageTag, "v1.2")
	}
}

func TestDbMigrationJobHandler_EnsureJobsAreCreated_CreatesJobForEnabledSiteWithMongoEnv(t *testing.T) {
	ctx := context.Background()
	services := map[string]sitev1.StagingSiteService{
		"mysvc": {ImageTag: "v2.0", MongoEnvironment: "mongo-env"},
	}
	site := testutil.NewTestStagingSite("mysite", "test-ns", services)
	site.Status.Enabled = true
	svcConfig := testutil.NewTestServiceConfig("mysvc", "test-ns", "svc")
	svcConfig.Spec.MigrationJobPodSpec = &corev1.PodSpec{
		Containers: []corev1.Container{
			{Name: "migrate", Image: "migrate:latest"},
		},
	}
	handler := newDbMigrationJobHandler(site, svcConfig)

	complete, err := handler.EnsureJobsAreCreated(site, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if complete {
		t.Error("expected complete=false after creating a new job")
	}

	jobList := &jobv1.DbMigrationJobList{}
	err = handler.Reader.List(ctx, jobList, client.InNamespace("test-ns"), client.MatchingLabels{labels.Site: "mysite"})
	if err != nil {
		t.Fatalf("error listing db migration jobs: %v", err)
	}
	if len(jobList.Items) != 1 {
		t.Fatalf("expected 1 db migration job, got %d", len(jobList.Items))
	}
}

func TestDbMigrationJobHandler_EnsureJobsAreCreated_ExistingMatchingJobMeansComplete(t *testing.T) {
	ctx := context.Background()
	services := map[string]sitev1.StagingSiteService{
		"mysvc": {ImageTag: "v1.0", MysqlEnvironment: "mysql-env"},
	}
	site := testutil.NewTestStagingSite("mysite", "test-ns", services)
	site.Status.Enabled = true
	svcConfig := testutil.NewTestServiceConfig("mysvc", "test-ns", "svc")
	svcConfig.Spec.MigrationJobPodSpec = &corev1.PodSpec{
		Containers: []corev1.Container{
			{Name: "migrate", Image: "migrate:latest"},
		},
	}

	// Pre-existing job that matches expected state
	existingJob := testutil.NewTestDbMigrationJob("mysite-svc", "test-ns", "mysite", "mysvc", "v1.0")
	existingJob.Labels = map[string]string{labels.Site: "mysite"}
	handler := newDbMigrationJobHandler(site, svcConfig, existingJob)

	complete, err := handler.EnsureJobsAreCreated(site, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !complete {
		t.Error("expected complete=true when job already exists and matches")
	}
}

func TestDbMigrationJobHandler_EnsureJobsAreCreated_ExistingJobWithDifferentTagIsUpdated(t *testing.T) {
	ctx := context.Background()
	services := map[string]sitev1.StagingSiteService{
		"mysvc": {ImageTag: "v2.0", MysqlEnvironment: "mysql-env"},
	}
	site := testutil.NewTestStagingSite("mysite", "test-ns", services)
	site.Status.Enabled = true
	svcConfig := testutil.NewTestServiceConfig("mysvc", "test-ns", "svc")
	svcConfig.Spec.MigrationJobPodSpec = &corev1.PodSpec{
		Containers: []corev1.Container{
			{Name: "migrate", Image: "migrate:latest"},
		},
	}

	// Pre-existing job with an old image tag
	existingJob := testutil.NewTestDbMigrationJob("mysite-svc", "test-ns", "mysite", "mysvc", "v1.0")
	existingJob.Labels = map[string]string{labels.Site: "mysite"}
	handler := newDbMigrationJobHandler(site, svcConfig, existingJob)

	complete, err := handler.EnsureJobsAreCreated(site, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// After update, the job exists but was updated (no create/delete needed beyond update)
	if !complete {
		t.Error("expected complete=true after updating an existing job (no net creates or deletes)")
	}

	jobList := &jobv1.DbMigrationJobList{}
	err = handler.Reader.List(ctx, jobList, client.InNamespace("test-ns"), client.MatchingLabels{labels.Site: "mysite"})
	if err != nil {
		t.Fatalf("error listing db migration jobs: %v", err)
	}
	if len(jobList.Items) != 1 {
		t.Fatalf("expected 1 db migration job, got %d", len(jobList.Items))
	}
	if jobList.Items[0].Spec.ImageTag != "v2.0" {
		t.Errorf("ImageTag after update = %q, want %q", jobList.Items[0].Spec.ImageTag, "v2.0")
	}
}

func TestDbMigrationJobHandler_EnsureJobsAreCreated_SiteDisabledDeletesExistingJobs(t *testing.T) {
	ctx := context.Background()
	site := testutil.NewTestStagingSite("mysite", "test-ns", nil)
	site.Status.Enabled = false

	// Stale job from when site was enabled
	staleJob := testutil.NewTestDbMigrationJob("mysite-svc", "test-ns", "mysite", "mysvc", "v1.0")
	staleJob.Labels = map[string]string{labels.Site: "mysite"}
	handler := newDbMigrationJobHandler(site, staleJob)

	complete, err := handler.EnsureJobsAreCreated(site, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if complete {
		t.Error("expected complete=false when stale jobs are deleted")
	}

	jobList := &jobv1.DbMigrationJobList{}
	_ = handler.Reader.List(ctx, jobList, client.InNamespace("test-ns"), client.MatchingLabels{labels.Site: "mysite"})
	if len(jobList.Items) != 0 {
		t.Errorf("expected 0 jobs after deletion, got %d", len(jobList.Items))
	}
}

func TestDbMigrationJobHandler_EnsureJobsAreComplete_NoJobsReturnsTrue(t *testing.T) {
	ctx := context.Background()
	site := testutil.NewTestStagingSite("mysite", "test-ns", nil)
	handler := newDbMigrationJobHandler(site)

	complete, err := handler.EnsureJobsAreComplete(site, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !complete {
		t.Error("expected complete=true when there are no jobs")
	}
}

func TestDbMigrationJobHandler_EnsureJobsAreComplete_AllCompleteReturnsTrue(t *testing.T) {
	ctx := context.Background()
	site := testutil.NewTestStagingSite("mysite", "test-ns", nil)
	job1 := testutil.NewTestDbMigrationJob("mysite-svc1", "test-ns", "mysite", "svc1", "v1")
	job1.Labels = map[string]string{labels.Site: "mysite"}
	job1.Status.State = jobv1.Complete
	job2 := testutil.NewTestDbMigrationJob("mysite-svc2", "test-ns", "mysite", "svc2", "v1")
	job2.Labels = map[string]string{labels.Site: "mysite"}
	job2.Status.State = jobv1.Complete
	handler := newDbMigrationJobHandler(site, job1, job2)

	complete, err := handler.EnsureJobsAreComplete(site, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !complete {
		t.Error("expected complete=true when all jobs are Complete")
	}
}

func TestDbMigrationJobHandler_EnsureJobsAreComplete_PendingJobReturnsFalse(t *testing.T) {
	ctx := context.Background()
	site := testutil.NewTestStagingSite("mysite", "test-ns", nil)
	job1 := testutil.NewTestDbMigrationJob("mysite-svc1", "test-ns", "mysite", "svc1", "v1")
	job1.Labels = map[string]string{labels.Site: "mysite"}
	job1.Status.State = jobv1.Pending
	handler := newDbMigrationJobHandler(site, job1)

	complete, err := handler.EnsureJobsAreComplete(site, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if complete {
		t.Error("expected complete=false when a job is Pending")
	}
}

func TestDbMigrationJobHandler_EnsureJobsAreComplete_FailedJobReturnsError(t *testing.T) {
	ctx := context.Background()
	site := testutil.NewTestStagingSite("mysite", "test-ns", nil)
	job1 := testutil.NewTestDbMigrationJob("mysite-svc1", "test-ns", "mysite", "svc1", "v1")
	job1.Labels = map[string]string{labels.Site: "mysite"}
	job1.Status.State = jobv1.Failed
	handler := newDbMigrationJobHandler(site, job1)

	complete, err := handler.EnsureJobsAreComplete(site, ctx)
	if err == nil {
		t.Error("expected error when a job is Failed")
	}
	if complete {
		t.Error("expected complete=false when a job is Failed")
	}
}

func TestDbMigrationJobHandler_EnsureJobsAreComplete_MixedCompleteAndPendingReturnsFalse(t *testing.T) {
	ctx := context.Background()
	site := testutil.NewTestStagingSite("mysite", "test-ns", nil)
	job1 := testutil.NewTestDbMigrationJob("mysite-svc1", "test-ns", "mysite", "svc1", "v1")
	job1.Labels = map[string]string{labels.Site: "mysite"}
	job1.Status.State = jobv1.Complete
	job2 := testutil.NewTestDbMigrationJob("mysite-svc2", "test-ns", "mysite", "svc2", "v1")
	job2.Labels = map[string]string{labels.Site: "mysite"}
	job2.Status.State = jobv1.Running
	handler := newDbMigrationJobHandler(site, job1, job2)

	complete, err := handler.EnsureJobsAreComplete(site, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if complete {
		t.Error("expected complete=false when at least one job is not Complete")
	}
}
