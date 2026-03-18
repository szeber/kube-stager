package database

import (
	"github.com/go-logr/logr"
	configv1 "github.com/szeber/kube-stager/apis/config/v1"
	taskv1 "github.com/szeber/kube-stager/apis/task/v1"
)

type MysqlReconciler interface {
	Reconcile(database *taskv1.MysqlDatabase, config configv1.MysqlConfig, logger logr.Logger) (bool, error)
	Delete(database *taskv1.MysqlDatabase, config configv1.MysqlConfig, logger logr.Logger) error
}

type MongoReconciler interface {
	Reconcile(database *taskv1.MongoDatabase, config configv1.MongoConfig, logger logr.Logger) (bool, error)
	Delete(database *taskv1.MongoDatabase, config configv1.MongoConfig, logger logr.Logger) error
}

// RedisReconciler omits Delete because Redis databases are ephemeral and the
// controller has no finalizer/cleanup logic.
type RedisReconciler interface {
	Reconcile(database *taskv1.RedisDatabase, config configv1.RedisConfig, logger logr.Logger) (bool, error)
}

// DefaultMysqlReconciler provides the production implementation using real MySQL connections.
type DefaultMysqlReconciler struct{}

func (DefaultMysqlReconciler) Reconcile(database *taskv1.MysqlDatabase, config configv1.MysqlConfig, logger logr.Logger) (bool, error) {
	return ReconcileMysqlDatabase(database, config, logger)
}

func (DefaultMysqlReconciler) Delete(database *taskv1.MysqlDatabase, config configv1.MysqlConfig, logger logr.Logger) error {
	return DeleteMysqlDatabase(database, config, logger)
}

// DefaultMongoReconciler provides the production implementation using real MongoDB connections.
type DefaultMongoReconciler struct{}

func (DefaultMongoReconciler) Reconcile(database *taskv1.MongoDatabase, config configv1.MongoConfig, logger logr.Logger) (bool, error) {
	return ReconcileMongoDatabase(database, config, logger)
}

func (DefaultMongoReconciler) Delete(database *taskv1.MongoDatabase, config configv1.MongoConfig, logger logr.Logger) error {
	return DeleteMongoDatabase(database, config, logger)
}

// DefaultRedisReconciler provides the production implementation using real Redis connections.
type DefaultRedisReconciler struct{}

func (DefaultRedisReconciler) Reconcile(database *taskv1.RedisDatabase, config configv1.RedisConfig, logger logr.Logger) (bool, error) {
	return ReconcileRedis(database, config, logger)
}
