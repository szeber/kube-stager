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
	mongoTestNamespace   = "mongo-ns"
	mongoTestSiteName    = "mongo-site"
	mongoTestServiceName = "mongo-service"
	mongoTestShortName   = "msvc"
	mongoTestEnvName     = "mongo-env"
)

func newMongoHandler(objs ...client.Object) (MongoTaskHandler, client.Client) {
	c := testutil.NewFakeClient(objs...)
	return MongoTaskHandler{
		Reader: c,
		Writer: c,
		Scheme: testutil.NewTestScheme(),
	}, c
}

func newMongoSiteWithEnv() *sitev1.StagingSite {
	return testutil.NewTestStagingSite(mongoTestSiteName, mongoTestNamespace, map[string]sitev1.StagingSiteService{
		mongoTestServiceName: {
			ImageTag:         "latest",
			Replicas:         1,
			MongoEnvironment: mongoTestEnvName,
		},
	})
}

func newMongoServiceConfig() *configv1.ServiceConfig {
	sc := testutil.NewTestServiceConfig(mongoTestServiceName, mongoTestNamespace, mongoTestShortName)
	sc.Spec.DefaultMongoEnvironment = mongoTestEnvName
	return sc
}

// TestMongoEnsureDatabasesAreCreated_CreatesDatabase verifies that when a site service
// has a MongoEnvironment set and no existing database exists, a MongoDatabase is created.
func TestMongoEnsureDatabasesAreCreated_CreatesDatabase(t *testing.T) {
	site := newMongoSiteWithEnv()
	sc := newMongoServiceConfig()

	handler, c := newMongoHandler(site, sc)
	ctx := context.Background()

	done, err := handler.EnsureDatabasesAreCreated(site, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if done {
		t.Error("expected done=false because creation was just performed")
	}

	var list taskv1.MongoDatabaseList
	if err := c.List(ctx, &list, client.InNamespace(mongoTestNamespace), client.MatchingLabels{labels.Site: mongoTestSiteName}); err != nil {
		t.Fatalf("failed to list mongo databases: %v", err)
	}
	if len(list.Items) != 1 {
		t.Fatalf("expected 1 MongoDatabase, got %d", len(list.Items))
	}
	db := list.Items[0]
	if db.Spec.EnvironmentConfig.ServiceName != mongoTestServiceName {
		t.Errorf("ServiceName = %q, want %q", db.Spec.EnvironmentConfig.ServiceName, mongoTestServiceName)
	}
	if db.Spec.EnvironmentConfig.Environment != mongoTestEnvName {
		t.Errorf("Environment = %q, want %q", db.Spec.EnvironmentConfig.Environment, mongoTestEnvName)
	}
	if db.Spec.EnvironmentConfig.SiteName != mongoTestSiteName {
		t.Errorf("SiteName = %q, want %q", db.Spec.EnvironmentConfig.SiteName, mongoTestSiteName)
	}
}

// TestMongoEnsureDatabasesAreCreated_NoOpWhenNoEnv verifies that services without a
// MongoEnvironment set do not trigger any database creation.
func TestMongoEnsureDatabasesAreCreated_NoOpWhenNoEnv(t *testing.T) {
	site := testutil.NewTestStagingSite(mongoTestSiteName, mongoTestNamespace, map[string]sitev1.StagingSiteService{
		mongoTestServiceName: {
			ImageTag: "latest",
			Replicas: 1,
			// MongoEnvironment intentionally left empty
		},
	})
	sc := newMongoServiceConfig()

	handler, c := newMongoHandler(site, sc)
	ctx := context.Background()

	done, err := handler.EnsureDatabasesAreCreated(site, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !done {
		t.Error("expected done=true when there is nothing to create")
	}

	var list taskv1.MongoDatabaseList
	if err := c.List(ctx, &list, client.InNamespace(mongoTestNamespace), client.MatchingLabels{labels.Site: mongoTestSiteName}); err != nil {
		t.Fatalf("failed to list mongo databases: %v", err)
	}
	if len(list.Items) != 0 {
		t.Errorf("expected 0 MongoDatabases, got %d", len(list.Items))
	}
}

// TestMongoEnsureDatabasesAreCreated_DeletesOrphanedDatabase verifies that when a service
// is removed from the site spec, its existing MongoDatabase is deleted.
func TestMongoEnsureDatabasesAreCreated_DeletesOrphanedDatabase(t *testing.T) {
	// Site has no services (the service was removed).
	site := testutil.NewTestStagingSite(mongoTestSiteName, mongoTestNamespace, map[string]sitev1.StagingSiteService{})
	sc := newMongoServiceConfig()

	// Pre-existing database for the now-removed service.
	existingDB := &taskv1.MongoDatabase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "orphaned-mongo-db",
			Namespace: mongoTestNamespace,
			Labels: map[string]string{
				labels.Site:             mongoTestSiteName,
				labels.Service:          mongoTestServiceName,
				labels.MongoEnvironment: mongoTestEnvName,
			},
		},
		Spec: taskv1.MongoDatabaseSpec{
			EnvironmentConfig: taskv1.EnvironmentConfig{
				ServiceName: mongoTestServiceName,
				SiteName:    mongoTestSiteName,
				Environment: mongoTestEnvName,
			},
			DatabaseName: "orphaned_mongo_db",
			Username:     "user",
			Password:     "pass",
		},
	}

	handler, c := newMongoHandler(site, sc, existingDB)
	ctx := context.Background()

	done, err := handler.EnsureDatabasesAreCreated(site, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if done {
		t.Error("expected done=false because a deletion was performed")
	}

	var list taskv1.MongoDatabaseList
	if err := c.List(ctx, &list, client.InNamespace(mongoTestNamespace), client.MatchingLabels{labels.Site: mongoTestSiteName}); err != nil {
		t.Fatalf("failed to list mongo databases: %v", err)
	}
	if len(list.Items) != 0 {
		t.Errorf("expected 0 MongoDatabases after deletion, got %d", len(list.Items))
	}
}

// TestMongoEnsureDatabasesAreReady_AllComplete verifies that when all databases are in the
// Complete state, EnsureDatabasesAreReady returns true.
func TestMongoEnsureDatabasesAreReady_AllComplete(t *testing.T) {
	site := newMongoSiteWithEnv()
	sc := newMongoServiceConfig()

	db := &taskv1.MongoDatabase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ready-mongo-db",
			Namespace: mongoTestNamespace,
			Labels: map[string]string{
				labels.Site: mongoTestSiteName,
			},
		},
		Spec: taskv1.MongoDatabaseSpec{
			EnvironmentConfig: taskv1.EnvironmentConfig{
				ServiceName: mongoTestServiceName,
				SiteName:    mongoTestSiteName,
				Environment: mongoTestEnvName,
			},
			DatabaseName: "ready_mongo_db",
			Username:     "user",
			Password:     "pass",
		},
	}

	handler, c := newMongoHandler(site, sc, db)
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

// TestMongoEnsureDatabasesAreReady_OnePending verifies that when at least one database is
// still Pending, EnsureDatabasesAreReady returns false without error.
func TestMongoEnsureDatabasesAreReady_OnePending(t *testing.T) {
	site := newMongoSiteWithEnv()
	sc := newMongoServiceConfig()

	db := &taskv1.MongoDatabase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pending-mongo-db",
			Namespace: mongoTestNamespace,
			Labels: map[string]string{
				labels.Site: mongoTestSiteName,
			},
		},
		Spec: taskv1.MongoDatabaseSpec{
			EnvironmentConfig: taskv1.EnvironmentConfig{
				ServiceName: mongoTestServiceName,
				SiteName:    mongoTestSiteName,
				Environment: mongoTestEnvName,
			},
			DatabaseName: "pending_mongo_db",
			Username:     "user",
			Password:     "pass",
		},
	}

	handler, _ := newMongoHandler(site, sc, db)
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

// TestMongoEnsureDatabasesAreReady_OneFailed verifies that when a database is in the Failed
// state, EnsureDatabasesAreReady returns a DatabaseCreationError.
func TestMongoEnsureDatabasesAreReady_OneFailed(t *testing.T) {
	site := newMongoSiteWithEnv()
	sc := newMongoServiceConfig()

	db := &taskv1.MongoDatabase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "failed-mongo-db",
			Namespace: mongoTestNamespace,
			Labels: map[string]string{
				labels.Site: mongoTestSiteName,
			},
		},
		Spec: taskv1.MongoDatabaseSpec{
			EnvironmentConfig: taskv1.EnvironmentConfig{
				ServiceName: mongoTestServiceName,
				SiteName:    mongoTestSiteName,
				Environment: mongoTestEnvName,
			},
			DatabaseName: "failed_mongo_db",
			Username:     "user",
			Password:     "pass",
		},
	}

	handler, c := newMongoHandler(site, sc, db)
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
	if dbErr.DatabaseType != errors.DatabaseTypeMongo {
		t.Errorf("DatabaseType = %q, want %q", dbErr.DatabaseType, errors.DatabaseTypeMongo)
	}
	if ready {
		t.Error("expected ready=false on error")
	}
}
