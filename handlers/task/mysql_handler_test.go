package task

import (
	"context"
	"testing"

	configv1 "github.com/szeber/kube-stager/apis/config/v1"
	sitev1 "github.com/szeber/kube-stager/apis/site/v1"
	taskv1 "github.com/szeber/kube-stager/apis/task/v1"
	"github.com/szeber/kube-stager/helpers/errors"
	"github.com/szeber/kube-stager/helpers/labels"
	"github.com/szeber/kube-stager/internal/testutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	mysqlTestNamespace   = "test-ns"
	mysqlTestSiteName    = "test-site"
	mysqlTestServiceName = "test-service"
	mysqlTestShortName   = "tsvc"
	mysqlTestEnvName     = "mysql-env"
)

func newMysqlHandler(objs ...client.Object) (MysqlTaskHandler, client.Client) {
	c := testutil.NewFakeClient(objs...)
	return MysqlTaskHandler{
		Reader: c,
		Writer: c,
		Scheme: testutil.NewTestScheme(),
	}, c
}

func newMysqlSiteWithEnv() *sitev1.StagingSite {
	return testutil.NewTestStagingSite(mysqlTestSiteName, mysqlTestNamespace, map[string]sitev1.StagingSiteService{
		mysqlTestServiceName: {
			ImageTag:         "latest",
			Replicas:         1,
			MysqlEnvironment: mysqlTestEnvName,
		},
	})
}

func newMysqlServiceConfig() *configv1.ServiceConfig {
	sc := testutil.NewTestServiceConfig(mysqlTestServiceName, mysqlTestNamespace, mysqlTestShortName)
	sc.Spec.DefaultMysqlEnvironment = mysqlTestEnvName
	return sc
}

// TestMysqlEnsureDatabasesAreCreated_CreatesDatabase verifies that when a site service
// has a MysqlEnvironment set and no existing database exists, a MysqlDatabase is created.
func TestMysqlEnsureDatabasesAreCreated_CreatesDatabase(t *testing.T) {
	site := newMysqlSiteWithEnv()
	sc := newMysqlServiceConfig()

	handler, c := newMysqlHandler(site, sc)
	ctx := context.Background()

	done, err := handler.EnsureDatabasesAreCreated(site, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if done {
		t.Error("expected done=false because creation was just performed")
	}

	var list taskv1.MysqlDatabaseList
	if err := c.List(ctx, &list, client.InNamespace(mysqlTestNamespace), client.MatchingLabels{labels.Site: mysqlTestSiteName}); err != nil {
		t.Fatalf("failed to list mysql databases: %v", err)
	}
	if len(list.Items) != 1 {
		t.Fatalf("expected 1 MysqlDatabase, got %d", len(list.Items))
	}
	db := list.Items[0]
	if db.Spec.EnvironmentConfig.ServiceName != mysqlTestServiceName {
		t.Errorf("ServiceName = %q, want %q", db.Spec.EnvironmentConfig.ServiceName, mysqlTestServiceName)
	}
	if db.Spec.EnvironmentConfig.Environment != mysqlTestEnvName {
		t.Errorf("Environment = %q, want %q", db.Spec.EnvironmentConfig.Environment, mysqlTestEnvName)
	}
	if db.Spec.EnvironmentConfig.SiteName != mysqlTestSiteName {
		t.Errorf("SiteName = %q, want %q", db.Spec.EnvironmentConfig.SiteName, mysqlTestSiteName)
	}
}

// TestMysqlEnsureDatabasesAreCreated_NoOpWhenNoEnv verifies that services without a
// MysqlEnvironment set do not trigger any database creation.
func TestMysqlEnsureDatabasesAreCreated_NoOpWhenNoEnv(t *testing.T) {
	site := testutil.NewTestStagingSite(mysqlTestSiteName, mysqlTestNamespace, map[string]sitev1.StagingSiteService{
		mysqlTestServiceName: {
			ImageTag: "latest",
			Replicas: 1,
			// MysqlEnvironment intentionally left empty
		},
	})
	sc := newMysqlServiceConfig()

	handler, c := newMysqlHandler(site, sc)
	ctx := context.Background()

	done, err := handler.EnsureDatabasesAreCreated(site, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !done {
		t.Error("expected done=true when there is nothing to create")
	}

	var list taskv1.MysqlDatabaseList
	if err := c.List(ctx, &list, client.InNamespace(mysqlTestNamespace), client.MatchingLabels{labels.Site: mysqlTestSiteName}); err != nil {
		t.Fatalf("failed to list mysql databases: %v", err)
	}
	if len(list.Items) != 0 {
		t.Errorf("expected 0 MysqlDatabases, got %d", len(list.Items))
	}
}

// TestMysqlEnsureDatabasesAreCreated_DeletesOrphanedDatabase verifies that when a service
// is removed from the site spec, its existing MysqlDatabase is deleted.
func TestMysqlEnsureDatabasesAreCreated_DeletesOrphanedDatabase(t *testing.T) {
	// Site has no services (the service was removed).
	site := testutil.NewTestStagingSite(mysqlTestSiteName, mysqlTestNamespace, map[string]sitev1.StagingSiteService{})
	sc := newMysqlServiceConfig()

	// Pre-existing database for the now-removed service.
	existingDB := &taskv1.MysqlDatabase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "orphaned-db",
			Namespace: mysqlTestNamespace,
			Labels: map[string]string{
				labels.Site:             mysqlTestSiteName,
				labels.Service:          mysqlTestServiceName,
				labels.MysqlEnvironment: mysqlTestEnvName,
			},
		},
		Spec: taskv1.MysqlDatabaseSpec{
			EnvironmentConfig: taskv1.EnvironmentConfig{
				ServiceName: mysqlTestServiceName,
				SiteName:    mysqlTestSiteName,
				Environment: mysqlTestEnvName,
			},
			DatabaseName: "orphaned_db",
			Username:     "user",
			Password:     "pass",
		},
	}

	handler, c := newMysqlHandler(site, sc, existingDB)
	ctx := context.Background()

	done, err := handler.EnsureDatabasesAreCreated(site, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if done {
		t.Error("expected done=false because a deletion was performed")
	}

	var list taskv1.MysqlDatabaseList
	if err := c.List(ctx, &list, client.InNamespace(mysqlTestNamespace), client.MatchingLabels{labels.Site: mysqlTestSiteName}); err != nil {
		t.Fatalf("failed to list mysql databases: %v", err)
	}
	if len(list.Items) != 0 {
		t.Errorf("expected 0 MysqlDatabases after deletion, got %d", len(list.Items))
	}
}

// TestMysqlEnsureDatabasesAreReady_AllComplete verifies that when all databases are in the
// Complete state, EnsureDatabasesAreReady returns true.
func TestMysqlEnsureDatabasesAreReady_AllComplete(t *testing.T) {
	site := newMysqlSiteWithEnv()
	sc := newMysqlServiceConfig()

	db := &taskv1.MysqlDatabase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ready-db",
			Namespace: mysqlTestNamespace,
			Labels: map[string]string{
				labels.Site: mysqlTestSiteName,
			},
		},
		Spec: taskv1.MysqlDatabaseSpec{
			EnvironmentConfig: taskv1.EnvironmentConfig{
				ServiceName: mysqlTestServiceName,
				SiteName:    mysqlTestSiteName,
				Environment: mysqlTestEnvName,
			},
			DatabaseName: "ready_db",
			Username:     "user",
			Password:     "pass",
		},
	}

	handler, c := newMysqlHandler(site, sc, db)
	ctx := context.Background()

	// Update status to Complete via the status subresource.
	db.Status.State = taskv1.Complete
	if err := c.Status().Update(ctx, db); err != nil {
		t.Fatalf("failed to set database status: %v", err)
	}

	ready, err := handler.EnsureDatabasesAreReady(site, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ready {
		t.Error("expected ready=true when all databases are Complete")
	}
}

// TestMysqlEnsureDatabasesAreReady_OnePending verifies that when at least one database is
// still Pending, EnsureDatabasesAreReady returns false without error.
func TestMysqlEnsureDatabasesAreReady_OnePending(t *testing.T) {
	site := newMysqlSiteWithEnv()
	sc := newMysqlServiceConfig()

	db := &taskv1.MysqlDatabase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pending-db",
			Namespace: mysqlTestNamespace,
			Labels: map[string]string{
				labels.Site: mysqlTestSiteName,
			},
		},
		Spec: taskv1.MysqlDatabaseSpec{
			EnvironmentConfig: taskv1.EnvironmentConfig{
				ServiceName: mysqlTestServiceName,
				SiteName:    mysqlTestSiteName,
				Environment: mysqlTestEnvName,
			},
			DatabaseName: "pending_db",
			Username:     "user",
			Password:     "pass",
		},
	}

	handler, _ := newMysqlHandler(site, sc, db)
	ctx := context.Background()

	// Status remains Pending (zero value).
	ready, err := handler.EnsureDatabasesAreReady(site, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ready {
		t.Error("expected ready=false when a database is still Pending")
	}
}

// TestMysqlEnsureDatabasesAreReady_OneFailed verifies that when a database is in the Failed
// state, EnsureDatabasesAreReady returns a DatabaseCreationError.
func TestMysqlEnsureDatabasesAreReady_OneFailed(t *testing.T) {
	site := newMysqlSiteWithEnv()
	sc := newMysqlServiceConfig()

	db := &taskv1.MysqlDatabase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "failed-db",
			Namespace: mysqlTestNamespace,
			Labels: map[string]string{
				labels.Site: mysqlTestSiteName,
			},
		},
		Spec: taskv1.MysqlDatabaseSpec{
			EnvironmentConfig: taskv1.EnvironmentConfig{
				ServiceName: mysqlTestServiceName,
				SiteName:    mysqlTestSiteName,
				Environment: mysqlTestEnvName,
			},
			DatabaseName: "failed_db",
			Username:     "user",
			Password:     "pass",
		},
	}

	handler, c := newMysqlHandler(site, sc, db)
	ctx := context.Background()

	db.Status.State = taskv1.Failed
	if err := c.Status().Update(ctx, db); err != nil {
		t.Fatalf("failed to set database status: %v", err)
	}

	ready, err := handler.EnsureDatabasesAreReady(site, ctx)
	if err == nil {
		t.Fatal("expected a DatabaseCreationError, got nil")
	}
	dbErr, ok := err.(errors.DatabaseCreationError)
	if !ok {
		t.Fatalf("expected DatabaseCreationError, got %T: %v", err, err)
	}
	if dbErr.DatabaseType != errors.DatabaseTypeMysql {
		t.Errorf("DatabaseType = %q, want %q", dbErr.DatabaseType, errors.DatabaseTypeMysql)
	}
	if ready {
		t.Error("expected ready=false on error")
	}
}
