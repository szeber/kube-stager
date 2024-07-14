package database

import (
	"crypto/tls"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/go-redis/redis"
	configv1 "github.com/szeber/kube-stager/apis/config/v1"
	taskv1 "github.com/szeber/kube-stager/apis/task/v1"
)

func ReconcileRedis(database *taskv1.RedisDatabase, config configv1.RedisConfig, logger logr.Logger) (bool, error) {
	if database.Status.State == taskv1.Complete {
		return false, nil
	}

	var tlsConfig *tls.Config

	if config.Spec.IsTlsEnabled != nil && *config.Spec.IsTlsEnabled {
		tlsConfig = &tls.Config{}
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

	foo := client.FlushDB()

	if err := foo.Err(); nil != err {
		return false, err
	}

	database.Status.State = taskv1.Complete

	return true, nil
}
