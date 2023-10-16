package job

import (
	"context"
	configv1 "github.com/szeber/kube-stager/api/config/v1"
	jobv1 "github.com/szeber/kube-stager/api/job/v1"
	sitev1 "github.com/szeber/kube-stager/api/site/v1"
	"github.com/szeber/kube-stager/helpers/errors"
	"github.com/szeber/kube-stager/helpers/labels"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type DbInitJobHandler struct {
	Reader client.Reader
	Writer client.Writer
	Scheme *runtime.Scheme
}

func (r DbInitJobHandler) EnsureJobsAreCreated(site *sitev1.StagingSite, ctx context.Context) (bool, error) {
	logger := log.FromContext(ctx)

	logger.V(0).Info("Retrieving database init job list")

	var list jobv1.DbInitJobList
	err := r.Reader.List(ctx, &list, client.InNamespace(site.Namespace), client.MatchingLabels{labels.Site: site.Name})
	if nil != err {
		return false, err
	}

	logger.V(1).Info("Retrieved list", "count", len(list.Items))
	logger.V(1).Info("Getting changes required to reconcile the db init jobs")

	jobsToDelete := make(map[string]jobv1.DbInitJob)
	jobsToCreate := make(map[string]jobv1.DbInitJob)

	for name, service := range site.Spec.Services {
		if service.MysqlEnvironment == "" && service.MongoEnvironment == "" {
			// Neither mysql or mongo are required, no db init is needed
			continue
		}
		var serviceConfig configv1.ServiceConfig
		err = r.Reader.Get(ctx, client.ObjectKey{Namespace: site.Namespace, Name: name}, &serviceConfig)
		if nil != err {
			return false, err
		}

		if serviceConfig.Spec.DbInitPodSpec == nil {
			// No db init pod spec is set, no db init is required
			continue
		}

		jobsToCreate[name], err = r.getPopulatedJob(
			site,
			&serviceConfig,
			service.MysqlEnvironment,
			service.MongoEnvironment,
		)
		if nil != err {
			return false, err
		}
	}

	for _, database := range list.Items {
		serviceName := database.Spec.ServiceName

		if _, ok := jobsToCreate[serviceName]; ok {
			delete(jobsToCreate, serviceName)
		} else {
			jobsToDelete[serviceName] = database
		}
	}

	isComplete := len(jobsToDelete) == 0 && len(jobsToCreate) == 0

	for serviceName, database := range jobsToDelete {
		logger.V(1).Info("Deleting init job for service " + serviceName)
		if err = r.Writer.Delete(ctx, &database); nil != err {
			return isComplete, err
		}
	}
	for serviceName, database := range jobsToCreate {
		logger.V(1).Info("Creating init job for service " + serviceName)
		if err = r.Writer.Create(ctx, &database); nil != err {
			return isComplete, err
		}
	}

	logger.V(0).Info("Init jobs created")

	return isComplete, nil
}

func (r DbInitJobHandler) EnsureJobsAreComplete(site *sitev1.StagingSite, ctx context.Context) (bool, error) {
	logger := log.FromContext(ctx)

	logger.V(0).Info("Retrieving database init job list")

	var list jobv1.DbInitJobList
	err := r.Reader.List(ctx, &list, client.InNamespace(site.Namespace), client.MatchingLabels{labels.Site: site.Name})
	if nil != err {
		return false, err
	}
	logger.V(1).Info("Retrieved list", "count", len(list.Items))

	isEverythingReady := true

	for _, database := range list.Items {
		if jobv1.Failed == database.Status.State {
			return false, errors.DatabaseInitError{
				SiteName:    database.Spec.SiteName,
				ServiceName: database.Spec.ServiceName,
			}
		}
		isEverythingReady = isEverythingReady && database.Status.State == jobv1.Complete
	}

	if isEverythingReady {
		logger.V(1).Info("All database init jobs are complete")
	} else {
		logger.V(0).Info("Not all database init jobs are complete yet")
	}

	return isEverythingReady, nil
}

func (r DbInitJobHandler) getPopulatedJob(
	site *sitev1.StagingSite,
	serviceConfig *configv1.ServiceConfig,
	mysqlEnvironment string,
	mongoEnvironment string,
) (jobv1.DbInitJob, error) {
	job := jobv1.DbInitJob{}
	if err := job.PopulateFomSite(site, serviceConfig, mysqlEnvironment, mongoEnvironment); nil != err {
		return job, err
	}

	if err := ctrl.SetControllerReference(site, &job, r.Scheme); nil != err {
		return job, err
	}

	return job, nil
}
