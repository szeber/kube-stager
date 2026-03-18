package testutil

import (
	"sync"
	"time"

	"github.com/go-logr/logr"
	configv1 "github.com/szeber/kube-stager/apis/config/v1"
	taskv1 "github.com/szeber/kube-stager/apis/task/v1"
)

type MockClock struct {
	mu  sync.Mutex
	now time.Time
}

func (m *MockClock) Now() time.Time {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.now
}

func (m *MockClock) SetNow(t time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.now = t
}

type MockMysqlReconciler struct {
	mu            sync.RWMutex
	reconcileFunc func(database *taskv1.MysqlDatabase, config configv1.MysqlConfig, logger logr.Logger) (bool, error)
	deleteFunc    func(database *taskv1.MysqlDatabase, config configv1.MysqlConfig, logger logr.Logger) error
}

func (m *MockMysqlReconciler) Reconcile(database *taskv1.MysqlDatabase, config configv1.MysqlConfig, logger logr.Logger) (bool, error) {
	m.mu.RLock()
	f := m.reconcileFunc
	m.mu.RUnlock()
	if f != nil {
		return f(database, config, logger)
	}
	return false, nil
}

func (m *MockMysqlReconciler) Delete(database *taskv1.MysqlDatabase, config configv1.MysqlConfig, logger logr.Logger) error {
	m.mu.RLock()
	f := m.deleteFunc
	m.mu.RUnlock()
	if f != nil {
		return f(database, config, logger)
	}
	return nil
}

func (m *MockMysqlReconciler) SetReconcileFunc(f func(database *taskv1.MysqlDatabase, config configv1.MysqlConfig, logger logr.Logger) (bool, error)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reconcileFunc = f
}

func (m *MockMysqlReconciler) SetDeleteFunc(f func(database *taskv1.MysqlDatabase, config configv1.MysqlConfig, logger logr.Logger) error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deleteFunc = f
}

type MockMongoReconciler struct {
	mu            sync.RWMutex
	reconcileFunc func(database *taskv1.MongoDatabase, config configv1.MongoConfig, logger logr.Logger) (bool, error)
	deleteFunc    func(database *taskv1.MongoDatabase, config configv1.MongoConfig, logger logr.Logger) error
}

func (m *MockMongoReconciler) Reconcile(database *taskv1.MongoDatabase, config configv1.MongoConfig, logger logr.Logger) (bool, error) {
	m.mu.RLock()
	f := m.reconcileFunc
	m.mu.RUnlock()
	if f != nil {
		return f(database, config, logger)
	}
	return false, nil
}

func (m *MockMongoReconciler) Delete(database *taskv1.MongoDatabase, config configv1.MongoConfig, logger logr.Logger) error {
	m.mu.RLock()
	f := m.deleteFunc
	m.mu.RUnlock()
	if f != nil {
		return f(database, config, logger)
	}
	return nil
}

func (m *MockMongoReconciler) SetReconcileFunc(f func(database *taskv1.MongoDatabase, config configv1.MongoConfig, logger logr.Logger) (bool, error)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reconcileFunc = f
}

func (m *MockMongoReconciler) SetDeleteFunc(f func(database *taskv1.MongoDatabase, config configv1.MongoConfig, logger logr.Logger) error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deleteFunc = f
}

type MockRedisReconciler struct {
	mu            sync.RWMutex
	reconcileFunc func(database *taskv1.RedisDatabase, config configv1.RedisConfig, logger logr.Logger) (bool, error)
}

func (m *MockRedisReconciler) Reconcile(database *taskv1.RedisDatabase, config configv1.RedisConfig, logger logr.Logger) (bool, error) {
	m.mu.RLock()
	f := m.reconcileFunc
	m.mu.RUnlock()
	if f != nil {
		return f(database, config, logger)
	}
	return false, nil
}

func (m *MockRedisReconciler) SetReconcileFunc(f func(database *taskv1.RedisDatabase, config configv1.RedisConfig, logger logr.Logger) (bool, error)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reconcileFunc = f
}
