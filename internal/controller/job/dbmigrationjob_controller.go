/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package job

import (
	"context"
	"errors"
	"fmt"
	"github.com/getsentry/sentry-go"
	configv1 "github.com/szeber/kube-stager/api/config/v1"
	controllerconfigv1 "github.com/szeber/kube-stager/api/controller-config/v1"
	sitev1 "github.com/szeber/kube-stager/api/site/v1"
	"github.com/szeber/kube-stager/handlers/importer"
	"github.com/szeber/kube-stager/handlers/template"
	"github.com/szeber/kube-stager/helpers"
	"github.com/szeber/kube-stager/helpers/labels"
	"github.com/szeber/kube-stager/helpers/pod"
	"github.com/szeber/kube-stager/internal/controller"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	jobv1 "github.com/szeber/kube-stager/api/job/v1"
)

// DbMigrationJobReconciler reconciles a DbMigrationJob object
type DbMigrationJobReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	Config        controllerconfigv1.ProjectConfig
	ImportHandler *importer.ImportHandler
}

//+kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;delete;deletecollection
//+kubebuilder:rbac:groups=job.operator.kube-stager.io,resources=dbmigrationjobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=job.operator.kube-stager.io,resources=dbmigrationjobs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=job.operator.kube-stager.io,resources=dbmigrationjobs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *DbMigrationJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	result, err := r.doReconcile(ctx, req)

	if nil != err {
		sentry.CaptureException(err)
	}

	return result, err
}

func (r *DbMigrationJobReconciler) doReconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if r.ImportHandler.IsObjectBeingImported(importer.TYPE_DB_MIGRATION_JOB, req.NamespacedName) {
		logger.Info("Skipping job as import is in progress")

		return ctrl.Result{}, nil
	}

	var job jobv1.DbMigrationJob

	if err := r.Get(ctx, req.NamespacedName, &job); err != nil {
		if client.IgnoreNotFound(err) != nil {
			logger.Error(err, "unable to fetch db migration job")
		}

		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if job.Spec.ImageTag != job.Status.LastMigratedImageTag {
		logger.V(0).Info("Either this is a new job, or the image name has changed. Updating job state")
		if err := r.deleteAssociatedJobs(&job, ctx); nil != err {
			return controller.SaveStatusUpdatesIfObjectChanged(false, r.Status(), ctx, &job, ctrl.Result{}, err)
		}
		job.Status.State = jobv1.Pending
		job.Status.LastMigratedImageTag = job.Spec.ImageTag
		job.Status.DeadlineTimestamp = &metav1.Time{Time: time.Now().Add(time.Duration(job.Spec.DeadlineSeconds) * time.Second)}
		return controller.SaveStatusUpdatesIfObjectChanged(
			true,
			r.Status(),
			ctx,
			&job,
			ctrl.Result{Requeue: true},
			nil,
		)
	}

	switch job.Status.State {
	case jobv1.Pending:
		changed, err := r.createJobIfNeeded(&job, ctx)
		return controller.SaveStatusUpdatesIfObjectChanged(changed, r.Status(), ctx, &job, ctrl.Result{}, err)
	case jobv1.Failed, jobv1.Complete:
		logger.V(0).Info("Job is in final state, ignoring", "state", job.Status.State)
		return ctrl.Result{}, nil
	case jobv1.Running:
		changed, err := r.processRunningJob(&job, ctx)
		return controller.SaveStatusUpdatesIfObjectChanged(changed, r.Status(), ctx, &job, ctrl.Result{}, err)
	default:
		return ctrl.Result{}, errors.New(fmt.Sprintf("Unknown state: %s", job.Status.State))
	}
}

func (r *DbMigrationJobReconciler) processRunningJob(job *jobv1.DbMigrationJob, ctx context.Context) (bool, error) {
	logger := log.FromContext(ctx)
	logger.V(0).Info("Loading migration job")

	jobList, err := r.getOwnedMigrationJobs(job, ctx)
	if nil != err {
		return false, err
	}

	if len(jobList.Items) != 1 {
		logger.V(0).Info("Invalid number of batch jobs found. Expected to find 1 job.", "count", len(jobList.Items))
		job.Status.JobNotFoundCount = job.Status.JobNotFoundCount + 1
		if job.Status.JobNotFoundCount > DbInitMaxJobFailedLoadLimit {
			logger.V(0).Info("The maximum not found time exceeded the limit, failing the job")
			job.Status.State = jobv1.Failed
		}
		return true, nil
	}

	changed := false
	if job.Status.JobNotFoundCount > 0 {
		job.Status.JobNotFoundCount = 0
		changed = true
	}

	batchJob := jobList.Items[0]

	for _, v := range batchJob.Status.Conditions {
		if v.Type == batchv1.JobComplete && v.Status == corev1.ConditionTrue {
			logger.V(0).Info("Job finished successfully")
			job.Status.State = jobv1.Complete
			return true, nil
		}

		if v.Type == batchv1.JobFailed && v.Status == corev1.ConditionTrue {
			logger.V(0).Info("Job failed", "status", v.Message, "reason", v.Reason)
			job.Status.State = jobv1.Failed
			return true, nil
		}
	}

	if time.Now().After(job.Status.DeadlineTimestamp.Time) {
		logger.V(0).Info("The job deadline has expired. Failing job.")
		job.Status.State = jobv1.Failed
		return true, nil
	}

	logger.V(0).Info("Job is still running")
	return changed, nil
}

func (r *DbMigrationJobReconciler) deleteAssociatedJobs(job *jobv1.DbMigrationJob, ctx context.Context) error {
	batchJob := batchv1.Job{}
	return r.DeleteAllOf(
		ctx,
		&batchJob,
		client.InNamespace(job.Namespace),
		client.MatchingLabels{labels.JobName: job.Name, labels.Type: "dbmigration"},
	)
}

func (r *DbMigrationJobReconciler) createJobIfNeeded(job *jobv1.DbMigrationJob, ctx context.Context) (bool, error) {
	logger := log.FromContext(ctx)
	logger.V(1).Info("Job is in Pending state")
	jobList, err := r.getOwnedMigrationJobs(job, ctx)
	if nil != err {
		return false, err
	}

	if len(jobList.Items) > 0 {
		logger.V(1).Info("There is already a job created, changing state to Running")
		job.Status.State = jobv1.Running
		return true, nil
	}

	serviceConfig, err := r.getServiceConfig(ctx, job.Namespace, job.Spec.ServiceName)
	if nil != err {
		return false, err
	}

	logger.V(0).Info("Creating job")
	batchJob, err := r.createJob(ctx, job, serviceConfig)
	if nil != err {
		return false, nil
	}

	if err := r.Create(ctx, batchJob); nil != err {
		return false, err
	}

	logger.V(0).Info("Job created, setting state to running")
	job.Status.State = jobv1.Running

	return true, nil
}

func (r *DbMigrationJobReconciler) getServiceConfig(
	ctx context.Context,
	namespace string,
	name string,
) (*configv1.ServiceConfig, error) {
	var config configv1.ServiceConfig
	if err := r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, &config); nil != err {
		return nil, err
	}

	return &config, nil
}

func (r *DbMigrationJobReconciler) getOwnedMigrationJobs(
	job *jobv1.DbMigrationJob,
	ctx context.Context,
) (*batchv1.JobList, error) {
	var jobList batchv1.JobList
	labelMatcher := client.MatchingLabels{labels.JobName: job.Name, labels.Type: "dbmigration"}
	if err := r.List(ctx, &jobList, client.InNamespace(job.Namespace), labelMatcher); nil != err {
		return nil, err
	}

	return &jobList, nil
}

func (r *DbMigrationJobReconciler) createJob(
	ctx context.Context,
	job *jobv1.DbMigrationJob,
	serviceConfig *configv1.ServiceConfig,
) (*batchv1.Job, error) {
	if nil == serviceConfig.Spec.MigrationJobPodSpec {
		return nil, errors.New("no migration pod spec specified in the service config")
	}

	var site sitev1.StagingSite
	err := r.Get(ctx, client.ObjectKey{Namespace: job.Namespace, Name: job.Spec.SiteName}, &site)
	if nil != err {
		return nil, err
	}

	backoffLimit := r.Config.MigrationJobConfig.BackoffLimit
	deadlineSeconds := int64(r.Config.MigrationJobConfig.DeadlineSeconds)
	ttlSeconds := r.Config.MigrationJobConfig.TtlSeconds
	jobLabels := map[string]string{
		labels.Type:    "dbmigration",
		labels.JobName: job.Name,
		labels.Site:    job.Spec.SiteName,
		labels.Service: job.Spec.ServiceName,
	}
	templateHandler := template.NewSite(site, *serviceConfig)
	err = template.LoadConfigs(&templateHandler, ctx, r)
	if nil != err {
		return nil, err
	}

	podSpec, err := helpers.ReplaceTemplateVariablesInPodSpec(*serviceConfig.Spec.MigrationJobPodSpec, &templateHandler)
	if nil != err {
		return nil, err
	}

	if podSpec.RestartPolicy != corev1.RestartPolicyOnFailure && podSpec.RestartPolicy != corev1.RestartPolicyNever {
		podSpec.RestartPolicy = corev1.RestartPolicyOnFailure
	}

	batchJob := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      helpers.ShortenHumanReadableValue(fmt.Sprintf("dbmigration-%s", job.Name), 50),
			Namespace: job.Namespace,
			Labels:    jobLabels,
		},
		Spec: batchv1.JobSpec{
			ActiveDeadlineSeconds: &deadlineSeconds,
			BackoffLimit:          &backoffLimit,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: jobLabels,
				},
				Spec: pod.SetExtraEnvVarsOnPodSpec(podSpec, &site, serviceConfig),
			},
			TTLSecondsAfterFinished: &ttlSeconds,
		},
	}

	if err = ctrl.SetControllerReference(job, batchJob, r.Scheme); nil != err {
		return nil, err
	}

	return batchJob, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DbMigrationJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&jobv1.DbMigrationJob{}).
		Owns(&batchv1.Job{}).
		Complete(r)
}
