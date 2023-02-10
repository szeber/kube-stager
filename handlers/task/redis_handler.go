package task

import (
	"context"
	"fmt"
	configv1 "github.com/szeber/kube-stager/apis/config/v1"
	sitev1 "github.com/szeber/kube-stager/apis/site/v1"
	taskv1 "github.com/szeber/kube-stager/apis/task/v1"
	"github.com/szeber/kube-stager/helpers/errors"
	"github.com/szeber/kube-stager/helpers/labels"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type RedisTaskHandler struct {
	Reader client.Reader
	Writer client.Writer
	Scheme *runtime.Scheme
}

func (r RedisTaskHandler) EnsureDatabasesAreCreated(site *sitev1.StagingSite, ctx context.Context) (bool, error) {
	logger := log.FromContext(ctx)

	logger.V(0).Info("Retrieving redis database list")

	var list taskv1.RedisDatabaseList
	err := r.Reader.List(ctx, &list, client.InNamespace(site.Namespace), client.MatchingLabels{labels.Site: site.Name})
	if nil != err {
		return false, err
	}

	logger.V(1).Info("Retrieved list.", "count", len(list.Items))
	logger.V(1).Info("Getting changes required to reconcile the redis databases")

	databasesToDelete := make(map[string]taskv1.RedisDatabase)
	databasesToUpdate := make(map[string]taskv1.RedisDatabase)
	databasesToCreate := make(map[string]taskv1.RedisDatabase)

	for name, service := range site.Spec.Services {
		if service.RedisEnvironment == "" {
			continue
		}
		var serviceConfig configv1.ServiceConfig
		err = r.Reader.Get(ctx, client.ObjectKey{Namespace: site.Namespace, Name: name}, &serviceConfig)
		if nil != err {
			return false, err
		}
		databasesToCreate[name], err = r.getPopulatedDatabase(site, &serviceConfig, service.RedisEnvironment)
		if nil != err {
			return false, err
		}
	}

	for _, database := range list.Items {
		serviceName := database.Spec.EnvironmentConfig.ServiceName

		if expectedDatabase, ok := databasesToCreate[serviceName]; ok {
			if !expectedDatabase.Matches(database) {
				database.UpdateFromExpected(expectedDatabase)
				expectedDatabase.Spec.DatabaseNumber, err = r.getFirstFreeDatabaseInEnvironment(
					ctx,
					site.Namespace,
					database.Spec.EnvironmentConfig,
				)
				if nil != err {
					return false, err
				}
				databasesToUpdate[serviceName] = database
			}
			delete(databasesToCreate, serviceName)
		} else {
			databasesToDelete[serviceName] = database
		}
	}

	isComplete := len(databasesToDelete) == 0 && len(databasesToCreate) == 0 && len(databasesToUpdate) == 0

	for serviceName, database := range databasesToDelete {
		logger.V(1).Info("Deleting redis for service " + serviceName)
		if err = r.Writer.Delete(ctx, &database); nil != err {
			return isComplete, err
		}
	}
	for serviceName, database := range databasesToCreate {
		logger.V(1).Info("Creating redis for service " + serviceName)
		if database.Spec.DatabaseNumber, err = r.getFirstFreeDatabaseInEnvironment(
			ctx,
			site.Namespace,
			database.Spec.EnvironmentConfig,
		); nil != err {
			return isComplete, err
		}
		if err = r.Writer.Create(ctx, &database); nil != err {
			return isComplete, err
		}
		service := site.Status.Services[serviceName]
		service.RedisDatabaseNumber = database.Spec.DatabaseNumber
		site.Status.Services[serviceName] = service
	}
	for serviceName, database := range databasesToUpdate {
		logger.V(1).Info("Updating redis for service " + serviceName)
		if err = r.Writer.Update(ctx, &database); nil != err {
			return isComplete, err
		}
		service := site.Status.Services[serviceName]
		service.RedisDatabaseNumber = database.Spec.DatabaseNumber
		site.Status.Services[serviceName] = service

	}

	logger.V(0).Info("Redis databases created/updated")

	return isComplete, nil
}

func (r RedisTaskHandler) EnsureDatabasesAreReady(site *sitev1.StagingSite, ctx context.Context) (bool, error) {
	logger := log.FromContext(ctx)

	logger.V(0).Info("Retrieving redis database list")

	var list taskv1.RedisDatabaseList
	err := r.Reader.List(ctx, &list, client.InNamespace(site.Namespace), client.MatchingLabels{labels.Site: site.Name})
	if nil != err {
		return false, err
	}
	logger.V(1).Info(fmt.Sprintf("Retrieved list. Count: %d", len(list.Items)))

	isEverythingReady := true

	for _, database := range list.Items {
		if taskv1.Failed == database.Status.State {
			return false, errors.DatabaseCreationError{
				DatabaseType:      errors.DatabaseTypeRedis,
				EnvironmentConfig: database.Spec.EnvironmentConfig,
				Reason:            "No free databases found in environment",
			}
		}
		isEverythingReady = isEverythingReady && database.Status.State == taskv1.Complete
	}

	if isEverythingReady {
		logger.V(1).Info("All redis databases are ready")
	} else {
		logger.V(0).Info("Not all databases are ready yet")
	}

	return isEverythingReady, nil
}

func (r RedisTaskHandler) getPopulatedDatabase(
	site *sitev1.StagingSite,
	config *configv1.ServiceConfig,
	environmentName string,
) (taskv1.RedisDatabase, error) {
	database := taskv1.RedisDatabase{}
	if err := database.PopulateFomSite(site, config, environmentName); nil != err {
		return database, err
	}

	if err := ctrl.SetControllerReference(site, &database, r.Scheme); nil != err {
		return database, err
	}

	return database, nil
}

func (r RedisTaskHandler) getFirstFreeDatabaseInEnvironment(
	ctx context.Context,
	namespace string,
	environmentConfig taskv1.EnvironmentConfig,
) (uint32, error) {
	redisConfig := configv1.RedisConfig{}
	log.FromContext(ctx).Info(
		fmt.Sprintf(
			"objectkey: %v",
			client.ObjectKey{Namespace: namespace, Name: environmentConfig.Environment},
		),
	)
	if err := r.Reader.Get(
		ctx,
		client.ObjectKey{Namespace: namespace, Name: environmentConfig.Environment},
		&redisConfig,
	); nil != err {
		return 0, err
	}

	if redisConfig.Name == "" {
		return 0, errors.DatabaseCreationError{
			DatabaseType:      errors.DatabaseTypeRedis,
			EnvironmentConfig: environmentConfig,
			Reason:            "Failed to load redis config",
		}
	}

	list := taskv1.RedisDatabaseList{}
	if err := r.Reader.List(
		ctx,
		&list,
		client.InNamespace(namespace),
		client.MatchingLabels{labels.RedisEnvironment: environmentConfig.Environment},
	); nil != err {
		return 0, err
	}

	availableDatabases := map[uint32]bool{}
	for i := uint32(0); i < redisConfig.Spec.AvailableDatabaseCount; i++ {
		availableDatabases[i] = true
	}

	for _, reservation := range list.Items {
		availableDatabases[reservation.Spec.DatabaseNumber] = false
	}

	for i := uint32(0); i < redisConfig.Spec.AvailableDatabaseCount; i++ {
		if availableDatabases[i] {
			return i, nil
		}
	}

	return 0, errors.DatabaseCreationError{
		DatabaseType:      errors.DatabaseTypeRedis,
		EnvironmentConfig: environmentConfig,
		Reason:            "No free databases found in environment",
	}
}
