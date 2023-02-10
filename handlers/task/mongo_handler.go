package task

import (
	"context"
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

type MongoTaskHandler struct {
	Reader client.Reader
	Writer client.Writer
	Scheme *runtime.Scheme
}

func (r MongoTaskHandler) EnsureDatabasesAreCreated(site *sitev1.StagingSite, ctx context.Context) (bool, error) {
	logger := log.FromContext(ctx)

	logger.V(0).Info("Retrieving mongo database list")

	var list taskv1.MongoDatabaseList
	err := r.Reader.List(ctx, &list, client.InNamespace(site.Namespace), client.MatchingLabels{labels.Site: site.Name})
	if nil != err {
		return false, err
	}

	logger.V(1).Info("Retrieved list.", "count", len(list.Items))
	logger.V(1).Info("Getting changes required to reconcile the mongo databases")

	databasesToDelete := make(map[string]taskv1.MongoDatabase)
	databasesToUpdate := make(map[string]taskv1.MongoDatabase)
	databasesToCreate := make(map[string]taskv1.MongoDatabase)

	for name, service := range site.Spec.Services {
		if service.MongoEnvironment == "" {
			continue
		}
		var serviceConfig configv1.ServiceConfig
		err = r.Reader.Get(ctx, client.ObjectKey{Namespace: site.Namespace, Name: name}, &serviceConfig)
		if nil != err {
			return false, err
		}
		databasesToCreate[name], err = r.getPopulatedDatabase(site, &serviceConfig, service.MongoEnvironment)
		if nil != err {
			return false, err
		}
	}

	for _, database := range list.Items {
		serviceName := database.Spec.EnvironmentConfig.ServiceName

		if expectedDatabase, ok := databasesToCreate[serviceName]; ok {
			if !expectedDatabase.Matches(database) {
				database.UpdateFromExpected(expectedDatabase)
				databasesToUpdate[serviceName] = database
			}
			delete(databasesToCreate, serviceName)
		} else {
			databasesToDelete[serviceName] = database
		}
	}

	isComplete := len(databasesToDelete) == 0 && len(databasesToCreate) == 0 && len(databasesToUpdate) == 0

	for serviceName, database := range databasesToDelete {
		logger.V(1).Info("Deleting mongo for service " + serviceName)
		if err = r.Writer.Delete(ctx, &database); nil != err {
			return isComplete, err
		}
	}
	for serviceName, database := range databasesToCreate {
		logger.V(1).Info("Creating mongo for service " + serviceName)
		if err = r.Writer.Create(ctx, &database); nil != err {
			return isComplete, err
		}
	}
	for serviceName, database := range databasesToUpdate {
		logger.V(1).Info("Updating mongo for service " + serviceName)
		if err = r.Writer.Update(ctx, &database); nil != err {
			return isComplete, err
		}
	}

	logger.V(0).Info("Mongo databases created/updated")

	return isComplete, nil
}

func (r MongoTaskHandler) EnsureDatabasesAreReady(site *sitev1.StagingSite, ctx context.Context) (bool, error) {
	logger := log.FromContext(ctx)

	logger.V(0).Info("Retrieving mongo database list")

	var list taskv1.MongoDatabaseList
	err := r.Reader.List(ctx, &list, client.InNamespace(site.Namespace), client.MatchingLabels{labels.Site: site.Name})
	if nil != err {
		return false, err
	}
	logger.V(1).Info("Retrieved list.", "count", len(list.Items))

	isEverythingReady := true

	for _, database := range list.Items {
		if taskv1.Failed == database.Status.State {
			return false, errors.DatabaseCreationError{
				DatabaseType:      errors.DatabaseTypeMongo,
				EnvironmentConfig: database.Spec.EnvironmentConfig,
			}
		}

		isEverythingReady = isEverythingReady && database.Status.State == taskv1.Complete
	}

	if isEverythingReady {
		logger.V(1).Info("All mongo databases are ready")
	} else {
		logger.V(0).Info("Not all databases are ready yet")
	}

	return isEverythingReady, nil
}

func (r MongoTaskHandler) getPopulatedDatabase(
	site *sitev1.StagingSite,
	config *configv1.ServiceConfig,
	environmentName string,
) (taskv1.MongoDatabase, error) {
	database := taskv1.MongoDatabase{}
	if err := database.PopulateFomSite(site, config, environmentName); nil != err {
		return database, err
	}

	if err := ctrl.SetControllerReference(site, &database, r.Scheme); nil != err {
		return database, err
	}

	return database, nil
}
