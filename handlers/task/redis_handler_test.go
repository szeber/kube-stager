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
	redisTestNamespace   = "redis-ns"
	redisTestSiteName    = "redis-site"
	redisTestServiceName = "redis-service"
	redisTestShortName   = "rsvc"
	redisTestEnvName     = "redis-env"
)

func newRedisHandler(objs ...client.Object) (RedisTaskHandler, client.Client) {
	c := testutil.NewFakeClient(objs...)
	return RedisTaskHandler{
		Reader: c,
		Writer: c,
		Scheme: testutil.NewTestScheme(),
	}, c
}

func newRedisSiteWithEnv() *sitev1.StagingSite {
	site := testutil.NewTestStagingSite(redisTestSiteName, redisTestNamespace, map[string]sitev1.StagingSiteService{
		redisTestServiceName: {
			ImageTag:         "latest",
			Replicas:         1,
			RedisEnvironment: redisTestEnvName,
		},
	})
	site.Status.Services = map[string]sitev1.StagingSiteServiceStatus{}
	return site
}

func newRedisServiceConfig() *configv1.ServiceConfig {
	sc := testutil.NewTestServiceConfig(redisTestServiceName, redisTestNamespace, redisTestShortName)
	sc.Spec.DefaultRedisEnvironment = redisTestEnvName
	return sc
}

func newRedisConfig() *configv1.RedisConfig {
	rc := testutil.NewTestRedisConfig(redisTestEnvName, redisTestNamespace)
	rc.Spec.AvailableDatabaseCount = 16
	return rc
}

// TestRedisEnsureDatabasesAreCreated_CreatesDatabase verifies that when a site service
// has a RedisEnvironment set and no existing database exists, a RedisDatabase is created.
func TestRedisEnsureDatabasesAreCreated_CreatesDatabase(t *testing.T) {
	site := newRedisSiteWithEnv()
	sc := newRedisServiceConfig()
	rc := newRedisConfig()

	handler, c := newRedisHandler(site, sc, rc)
	ctx := context.Background()

	done, err := handler.EnsureDatabasesAreCreated(site, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if done {
		t.Error("expected done=false because creation was just performed")
	}

	var list taskv1.RedisDatabaseList
	if err := c.List(ctx, &list, client.InNamespace(redisTestNamespace), client.MatchingLabels{labels.Site: redisTestSiteName}); err != nil {
		t.Fatalf("failed to list redis databases: %v", err)
	}
	if len(list.Items) != 1 {
		t.Fatalf("expected 1 RedisDatabase, got %d", len(list.Items))
	}
	db := list.Items[0]
	if db.Spec.EnvironmentConfig.ServiceName != redisTestServiceName {
		t.Errorf("ServiceName = %q, want %q", db.Spec.EnvironmentConfig.ServiceName, redisTestServiceName)
	}
	if db.Spec.EnvironmentConfig.Environment != redisTestEnvName {
		t.Errorf("Environment = %q, want %q", db.Spec.EnvironmentConfig.Environment, redisTestEnvName)
	}
	if db.Spec.EnvironmentConfig.SiteName != redisTestSiteName {
		t.Errorf("SiteName = %q, want %q", db.Spec.EnvironmentConfig.SiteName, redisTestSiteName)
	}
}

// TestRedisEnsureDatabasesAreCreated_NoOpWhenNoEnv verifies that services without a
// RedisEnvironment set do not trigger any database creation.
func TestRedisEnsureDatabasesAreCreated_NoOpWhenNoEnv(t *testing.T) {
	site := testutil.NewTestStagingSite(redisTestSiteName, redisTestNamespace, map[string]sitev1.StagingSiteService{
		redisTestServiceName: {
			ImageTag: "latest",
			Replicas: 1,
			// RedisEnvironment intentionally left empty
		},
	})
	sc := newRedisServiceConfig()
	rc := newRedisConfig()

	handler, c := newRedisHandler(site, sc, rc)
	ctx := context.Background()

	done, err := handler.EnsureDatabasesAreCreated(site, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !done {
		t.Error("expected done=true when there is nothing to create")
	}

	var list taskv1.RedisDatabaseList
	if err := c.List(ctx, &list, client.InNamespace(redisTestNamespace), client.MatchingLabels{labels.Site: redisTestSiteName}); err != nil {
		t.Fatalf("failed to list redis databases: %v", err)
	}
	if len(list.Items) != 0 {
		t.Errorf("expected 0 RedisDatabases, got %d", len(list.Items))
	}
}

// TestRedisEnsureDatabasesAreCreated_DeletesOrphanedDatabase verifies that when a service
// is removed from the site spec, its existing RedisDatabase is deleted.
func TestRedisEnsureDatabasesAreCreated_DeletesOrphanedDatabase(t *testing.T) {
	// Site has no services (the service was removed).
	site := testutil.NewTestStagingSite(redisTestSiteName, redisTestNamespace, map[string]sitev1.StagingSiteService{})
	sc := newRedisServiceConfig()
	rc := newRedisConfig()

	// Pre-existing database for the now-removed service.
	existingDB := &taskv1.RedisDatabase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "orphaned-redis-db",
			Namespace: redisTestNamespace,
			Labels: map[string]string{
				labels.Site:             redisTestSiteName,
				labels.Service:          redisTestServiceName,
				labels.RedisEnvironment: redisTestEnvName,
			},
		},
		Spec: taskv1.RedisDatabaseSpec{
			EnvironmentConfig: taskv1.EnvironmentConfig{
				ServiceName: redisTestServiceName,
				SiteName:    redisTestSiteName,
				Environment: redisTestEnvName,
			},
			DatabaseNumber: 0,
		},
	}

	handler, c := newRedisHandler(site, sc, rc, existingDB)
	ctx := context.Background()

	done, err := handler.EnsureDatabasesAreCreated(site, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if done {
		t.Error("expected done=false because a deletion was performed")
	}

	var list taskv1.RedisDatabaseList
	if err := c.List(ctx, &list, client.InNamespace(redisTestNamespace), client.MatchingLabels{labels.Site: redisTestSiteName}); err != nil {
		t.Fatalf("failed to list redis databases: %v", err)
	}
	if len(list.Items) != 0 {
		t.Errorf("expected 0 RedisDatabases after deletion, got %d", len(list.Items))
	}
}

// TestRedisEnsureDatabasesAreReady_AllComplete verifies that when all databases are in the
// Complete state, EnsureDatabasesAreReady returns true.
func TestRedisEnsureDatabasesAreReady_AllComplete(t *testing.T) {
	site := newRedisSiteWithEnv()
	sc := newRedisServiceConfig()
	rc := newRedisConfig()

	db := &taskv1.RedisDatabase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ready-redis-db",
			Namespace: redisTestNamespace,
			Labels: map[string]string{
				labels.Site: redisTestSiteName,
			},
		},
		Spec: taskv1.RedisDatabaseSpec{
			EnvironmentConfig: taskv1.EnvironmentConfig{
				ServiceName: redisTestServiceName,
				SiteName:    redisTestSiteName,
				Environment: redisTestEnvName,
			},
			DatabaseNumber: 0,
		},
	}

	handler, c := newRedisHandler(site, sc, rc, db)
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

// TestRedisEnsureDatabasesAreReady_OnePending verifies that when at least one database is
// still Pending, EnsureDatabasesAreReady returns false without error.
func TestRedisEnsureDatabasesAreReady_OnePending(t *testing.T) {
	site := newRedisSiteWithEnv()
	sc := newRedisServiceConfig()
	rc := newRedisConfig()

	db := &taskv1.RedisDatabase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pending-redis-db",
			Namespace: redisTestNamespace,
			Labels: map[string]string{
				labels.Site: redisTestSiteName,
			},
		},
		Spec: taskv1.RedisDatabaseSpec{
			EnvironmentConfig: taskv1.EnvironmentConfig{
				ServiceName: redisTestServiceName,
				SiteName:    redisTestSiteName,
				Environment: redisTestEnvName,
			},
			DatabaseNumber: 0,
		},
	}

	handler, _ := newRedisHandler(site, sc, rc, db)
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

// TestRedisEnsureDatabasesAreReady_OneFailed verifies that when a database is in the Failed
// state, EnsureDatabasesAreReady returns a DatabaseCreationError.
func TestRedisEnsureDatabasesAreReady_OneFailed(t *testing.T) {
	site := newRedisSiteWithEnv()
	sc := newRedisServiceConfig()
	rc := newRedisConfig()

	db := &taskv1.RedisDatabase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "failed-redis-db",
			Namespace: redisTestNamespace,
			Labels: map[string]string{
				labels.Site: redisTestSiteName,
			},
		},
		Spec: taskv1.RedisDatabaseSpec{
			EnvironmentConfig: taskv1.EnvironmentConfig{
				ServiceName: redisTestServiceName,
				SiteName:    redisTestSiteName,
				Environment: redisTestEnvName,
			},
			DatabaseNumber: 0,
		},
	}

	handler, c := newRedisHandler(site, sc, rc, db)
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
	if dbErr.DatabaseType != errors.DatabaseTypeRedis {
		t.Errorf("DatabaseType = %q, want %q", dbErr.DatabaseType, errors.DatabaseTypeRedis)
	}
	if ready {
		t.Error("expected ready=false on error")
	}
}
