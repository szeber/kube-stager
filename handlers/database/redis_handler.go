package database

import (
	"crypto/tls"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/go-redis/redis"
	"github.com/prometheus/client_golang/prometheus"
	configv1 "github.com/szeber/kube-stager/apis/config/v1"
	taskv1 "github.com/szeber/kube-stager/apis/task/v1"
	appmetrics "github.com/szeber/kube-stager/internal/metrics"
)

func ReconcileRedis(database *taskv1.RedisDatabase, config configv1.RedisConfig, logger logr.Logger) (bool, error) {
	if database.Status.State == taskv1.Complete {
		return false, nil
	}

	timer := prometheus.NewTimer(appmetrics.DatabaseOperationDuration.WithLabelValues("redis", "reconcile"))
	defer timer.ObserveDuration()

	var tlsConfig *tls.Config

	if config.Spec.IsTlsEnabled != nil && *config.Spec.IsTlsEnabled {
		tlsConfig = &tls.Config{}
		if config.Spec.VerifyTlsServerCertificate != nil && !*config.Spec.VerifyTlsServerCertificate {
			logger.Info("Disabling TLS server certificate verification")
			tlsConfig.InsecureSkipVerify = true
		}
	}

	logger.Info(fmt.Sprintf("Flushing redis database %d on connection %s", database.Spec.DatabaseNumber, config.Name))
	client := redis.NewClient(
		&redis.Options{
			Addr:      config.Spec.Host + ":" + fmt.Sprint(config.Spec.Port),
			DB:        int(database.Spec.DatabaseNumber),
			Password:  config.Spec.Password,
			TLSConfig: tlsConfig,
		},
	)
	defer func() { _ = client.Close() }()

	foo := client.FlushDB()

	if err := foo.Err(); err != nil {
		appmetrics.DatabaseOperations.WithLabelValues("redis", "reconcile", "error").Inc()
		return false, err
	}

	appmetrics.DatabaseOperations.WithLabelValues("redis", "reconcile", "success").Inc()

	database.Status.State = taskv1.Complete

	return true, nil
}
