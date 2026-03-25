package job

import (
	"context"
	configv1 "github.com/szeber/kube-stager/apis/config/v1"
	jobv1 "github.com/szeber/kube-stager/apis/job/v1"
	sitev1 "github.com/szeber/kube-stager/apis/site/v1"
	"github.com/szeber/kube-stager/helpers/errors"
	"github.com/szeber/kube-stager/helpers/labels"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type DbMigrationJobHandler struct {
	Reader client.Reader
	Writer client.Writer
	Scheme *runtime.Scheme
}

func (r DbMigrationJobHandler) EnsureJobsAreCreated(site *sitev1.StagingSite, ctx context.Context) (bool, error) {
	logger := log.FromContext(ctx)

	logger.V(0).Info("Retrieving database migration job list")

	var list jobv1.DbMigrationJobList
	err := r.Reader.List(ctx, &list, client.InNamespace(site.Namespace), client.MatchingLabels{labels.Site: site.Name})
	if err != nil {
		return false, err
	}

	logger.V(1).Info("Retrieved list", "count", len(list.Items))
	logger.V(1).Info("Getting changes required to reconcile the db migration jobs")

	jobsToCreate := make(map[string]jobv1.DbMigrationJob)
	jobsToUpdate := make(map[string]jobv1.DbMigrationJob)
	jobsToDelete := make(map[string]jobv1.DbMigrationJob)

	if site.Status.Enabled {
		for name, service := range site.Spec.Services {
			if service.MysqlEnvironment == "" && service.MongoEnvironment == "" {
				// Neither mysql or mongo are required, no db migration is needed
				continue
			}
			var serviceConfig configv1.ServiceConfig
			err = r.Reader.Get(ctx, client.ObjectKey{Namespace: site.Namespace, Name: name}, &serviceConfig)
			if err != nil {
				return false, err
			}

			if serviceConfig.Spec.MigrationJobPodSpec == nil {
				// No pod spec for migrations, so this service does not do migrations
				continue
			}

			jobsToCreate[name], err = r.getPopulatedJob(site, &serviceConfig)
			logger.V(2).Info(
				"Populating migration job",
				"site",
				site.Name,
				"service",
				serviceConfig.Name,
				"tag",
				site.Spec.Services[serviceConfig.Name].ImageTag,
			)
			if err != nil {
				return false, err
			}
		}
	}

	for _, existingJob := range list.Items {
		serviceName := existingJob.Spec.ServiceName

		if jobToCreate, ok := jobsToCreate[serviceName]; ok {
			if !existingJob.Matches(&jobToCreate) {
				existingJob.UpdateFrom(&jobToCreate)
				jobsToUpdate[serviceName] = existingJob
			}
			delete(jobsToCreate, serviceName)
		} else {
			jobsToDelete[serviceName] = existingJob
		}
	}

	isComplete := len(jobsToDelete) == 0 && len(jobsToCreate) == 0

	for serviceName, job := range jobsToDelete {
		logger.V(1).Info("Deleting migration job for service " + serviceName)
		if err = r.Writer.Delete(ctx, &job); err != nil {
			return isComplete, err
		}
	}
	for serviceName, job := range jobsToUpdate {
		logger.V(1).Info("Updating migration job for service " + serviceName)
		if err = r.Writer.Update(ctx, &job); err != nil {
			return isComplete, err
		}
	}
	for serviceName, job := range jobsToCreate {
		logger.V(1).Info("Creating migration job for service " + serviceName)
		if err = r.Writer.Create(ctx, &job); err != nil {
			return isComplete, err
		}
	}

	logger.V(0).Info("Migration jobs created")

	return isComplete, nil
}

func (r DbMigrationJobHandler) EnsureJobsAreComplete(site *sitev1.StagingSite, ctx context.Context) (bool, error) {
	logger := log.FromContext(ctx)

	logger.V(0).Info("Retrieving database migration job list")

	var list jobv1.DbMigrationJobList
	err := r.Reader.List(ctx, &list, client.InNamespace(site.Namespace), client.MatchingLabels{labels.Site: site.Name})
	if err != nil {
		return false, err
	}
	logger.V(1).Info("Retrieved list", "count", len(list.Items))

	isEverythingReady := true

	for _, database := range list.Items {
		if database.Status.State == jobv1.Failed {
			return false, errors.DatabaseMigrationError{
				SiteName:    database.Spec.SiteName,
				ServiceName: database.Spec.ServiceName,
			}
		}
		isEverythingReady = isEverythingReady && database.Status.State == jobv1.Complete
	}

	if isEverythingReady {
		logger.V(1).Info("All database migration jobs are complete")
	} else {
		logger.V(0).Info("Not all database migration jobs are complete yet")
	}

	return isEverythingReady, nil
}

func (r DbMigrationJobHandler) getPopulatedJob(
	site *sitev1.StagingSite,
	serviceConfig *configv1.ServiceConfig,
) (jobv1.DbMigrationJob, error) {
	job := jobv1.DbMigrationJob{}
	if err := job.PopulateFomSite(site, serviceConfig); err != nil {
		return job, err
	}

	if err := ctrl.SetControllerReference(site, &job, r.Scheme); err != nil {
		return job, err
	}

	return job, nil
}
