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
	"fmt"
	"github.com/getsentry/sentry-go"
	"github.com/go-logr/logr"
	configv1 "github.com/szeber/kube-stager/apis/config/v1"
	controllerconfigv1 "github.com/szeber/kube-stager/apis/controller-config/v1"
	sitev1 "github.com/szeber/kube-stager/apis/site/v1"
	"github.com/szeber/kube-stager/handlers/template"
	"github.com/szeber/kube-stager/helpers"
	"github.com/szeber/kube-stager/helpers/labels"
	"github.com/szeber/kube-stager/helpers/pod"
	"github.com/szeber/kube-stager/internal/controllers"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	jobv1 "github.com/szeber/kube-stager/apis/job/v1"
)

// BackupReconciler reconciles a Backup object
type BackupReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Config controllerconfigv1.ProjectConfig
	Clock
}

type realClock struct{}

func (_ realClock) Now() time.Time {
	return time.Now()
}

type Clock interface {
	Now() time.Time
}

//+kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create
//+kubebuilder:rbac:groups=config.operator.kube-stager.io,resources=serviceconfigs,verbs=get;list;watch
//+kubebuilder:rbac:groups=site.operator.kube-stager.io,resources=stagingsites,verbs=get;list;watch
//+kubebuilder:rbac:groups=job.operator.kube-stager.io,resources=backups,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=job.operator.kube-stager.io,resources=backups/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=job.operator.kube-stager.io,resources=backups/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *BackupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	result, err := r.doReconcile(ctx, req)

	if nil != err {
		sentry.CaptureException(err)
	}

	return result, err
}

func (r *BackupReconciler) doReconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	job := &jobv1.Backup{}

	logger.V(0).Info("Loading backup job", "namespace", req.Namespace, "name", req.Name)

	if err := r.Get(ctx, req.NamespacedName, job); err != nil {
		if client.IgnoreNotFound(err) != nil {
			logger.Error(err, "unable to fetch backup job")
		}

		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.V(1).Info("Loaded job")

	if r.ensureStatusIsInitialised(job) {
		return controller.SaveStatusUpdatesIfObjectChanged(true, r.Status(), ctx, job, ctrl.Result{Requeue: true}, nil)
	}

	if job.Status.State.IsFinal() {
		logger.V(0).Info("Job is in a final state, skipping", "state", job.Status.State)
	}

	isChanged, err := r.ensureJobsAreUpToDate(ctx, job)

	return controller.SaveStatusUpdatesIfObjectChanged(isChanged, r.Status(), ctx, job, ctrl.Result{}, err)
}

func (r *BackupReconciler) ensureStatusIsInitialised(backup *jobv1.Backup) bool {
	changed := false

	if "" == backup.Status.State {
		backup.Status.State = jobv1.Pending
		changed = true
	}

	if 0 == len(backup.Status.Services) {
		backup.Status.Services = map[string]jobv1.BackupStatusDetail{}
	}

	return changed
}

func (r *BackupReconciler) ensureJobsAreUpToDate(ctx context.Context, job *jobv1.Backup) (bool, error) {
	logger := log.FromContext(ctx)

	logger.V(1).Info("Loading site")
	site := &sitev1.StagingSite{}
	if err := r.Get(ctx, client.ObjectKey{Namespace: job.Namespace, Name: job.Spec.SiteName}, site); nil != err {
		return false, err
	}

	services, isChanged, err := r.loadServicesForSite(ctx, job, site)
	if err != nil {
		return isChanged, err
	}

	if len(services) == 0 {
		logger.V(0).Info("No backups are required as there are no services with backups enabled in the site")
		job.Status.State = jobv1.Complete
		_ = r.updateJobStartedAtIfNeeded(job, r.Clock.Now())
		if nil == job.Status.JobFinishedAt {
			t := metav1.NewTime(r.Clock.Now())
			job.Status.JobFinishedAt = &t
		}

		return true, nil
	}

	isChanged, allServicesFinished, lastFinishedAt, err := r.processServiceInJob(
		ctx,
		job,
		services,
		isChanged,
		site,
		logger,
	)
	if nil != err {
		return isChanged, err
	}

	if jobv1.Failed != job.Status.State && allServicesFinished {
		job.Status.State = jobv1.Complete
		if nil != lastFinishedAt {
			job.Status.JobFinishedAt = &metav1.Time{Time: *lastFinishedAt}
		}
		isChanged = true
	}

	return isChanged, nil
}

func (r *BackupReconciler) processServiceInJob(
	ctx context.Context,
	job *jobv1.Backup,
	services map[string]configv1.ServiceConfig,
	isChanged bool,
	site *sitev1.StagingSite,
	logger logr.Logger,
) (bool, bool, *time.Time, error) {
	var lastFinishedAt *time.Time
	allServicesFinished := true

	for name, service := range services {
		batchJobName := r.getBatchJobName(name, job.Name)
		batchJob := &batchv1.Job{}
		serviceStatus := job.Status.Services[name]

		if serviceStatus.State.IsFinal() {
			if jobv1.Failed == serviceStatus.State {
				job.Status.State = jobv1.Failed
				isChanged = true
			} else if jobv1.Complete == serviceStatus.State {
				if nil == lastFinishedAt || lastFinishedAt.Before(serviceStatus.JobFinishedAt.Time) {
					lastFinishedAt = &serviceStatus.JobFinishedAt.Time
					isChanged = true
				}
			}

			continue
		}

		if err := r.Get(
			ctx,
			client.ObjectKey{Namespace: job.Namespace, Name: batchJobName},
			batchJob,
		); nil != client.IgnoreNotFound(err) {
			return isChanged, false, lastFinishedAt, err
		} else if nil != err {
			batchJob, err = r.getNewBackupJob(ctx, job, *site, service)
			if nil != err {
				return isChanged, false, lastFinishedAt, err
			}

			now := r.Clock.Now()
			isChanged = true
			_ = r.updateJobStartedAtIfNeeded(job, now)
			allServicesFinished = false
			job.Status.State = jobv1.Running
			serviceStatus.JobStartedAt = &metav1.Time{Time: now}
			serviceStatus.JobFinishedAt = nil
			serviceStatus.State = jobv1.Running

			if err := r.Create(ctx, batchJob); nil != err {
				return isChanged, false, lastFinishedAt, err
			}
		} else {
			serviceStatus.JobStartedAt = &batchJob.CreationTimestamp
			isChanged = isChanged || r.updateJobStartedAtIfNeeded(job, batchJob.CreationTimestamp.Time)
			serviceFinished := false

			for _, v := range batchJob.Status.Conditions {
				if v.Type == batchv1.JobComplete && v.Status == corev1.ConditionTrue {
					logger.V(0).Info("Job finished successfully")
					serviceStatus.State = jobv1.Complete
					serviceStatus.JobFinishedAt = &v.LastTransitionTime
					if nil == lastFinishedAt || lastFinishedAt.Before(v.LastTransitionTime.Time) {
						lastFinishedAt = &v.LastTransitionTime.Time
					}
					isChanged = true
					serviceFinished = true
					break
				}

				if v.Type == batchv1.JobFailed && v.Status == corev1.ConditionTrue {
					logger.V(0).Info("Job failed", "status", v.Message, "reason", v.Reason)
					job.Status.State = jobv1.Failed
					serviceStatus.State = jobv1.Failed
					isChanged = true
					break
				}
			}

			allServicesFinished = allServicesFinished && serviceFinished
		}

		job.Status.Services[name] = serviceStatus
	}

	return isChanged, allServicesFinished, lastFinishedAt, nil
}

func (r *BackupReconciler) loadServicesForSite(
	ctx context.Context,
	job *jobv1.Backup,
	site *sitev1.StagingSite,
) (map[string]configv1.ServiceConfig, bool, error) {
	isChanged := false
	services := make(map[string]configv1.ServiceConfig)

	for name := range site.Status.Services {
		service := &configv1.ServiceConfig{}
		if err := r.Get(ctx, client.ObjectKey{Namespace: job.Namespace, Name: name}, service); nil != err {
			return nil, isChanged, err
		}
		if nil == service.Spec.BackupPodSpec {
			continue
		}
		if "" == job.Status.Services[name].State {
			serviceState := job.Status.Services[name]
			serviceState.State = jobv1.Pending
			job.Status.Services[name] = serviceState
			isChanged = true
		}

		services[name] = *service
	}

	return services, isChanged, nil
}

func (r *BackupReconciler) updateJobStartedAtIfNeeded(job *jobv1.Backup, currentStartedAt time.Time) bool {
	if nil == job.Status.JobStartedAt || job.Status.JobStartedAt.Time.After(currentStartedAt) {
		job.Status.JobStartedAt = &metav1.Time{Time: currentStartedAt}
		return true
	}

	return false
}

func (r *BackupReconciler) getBatchJobName(serviceName string, jobName string) string {
	return helpers.ShortenHumanReadableValue(fmt.Sprintf("backup-%s-%s", serviceName, jobName), 50)
}

func (r *BackupReconciler) getNewBackupJob(
	ctx context.Context,
	job *jobv1.Backup,
	site sitev1.StagingSite,
	serviceConfig configv1.ServiceConfig,
) (*batchv1.Job, error) {
	backoffLimit := r.Config.BackupJobConfig.BackoffLimit
	deadlineSeconds := int64(r.Config.BackupJobConfig.DeadlineSeconds)
	ttlSeconds := r.Config.BackupJobConfig.TtlSeconds
	jobLabels := map[string]string{
		labels.Type:    "backup",
		labels.JobName: job.Name,
		labels.Site:    job.Spec.SiteName,
		labels.Service: serviceConfig.Name,
	}
	templateHandler := template.NewSite(site, serviceConfig)
	err := template.LoadConfigs(
		&templateHandler,
		ctx,
		r,
		site.Spec.Services[serviceConfig.Name].MysqlEnvironment,
		site.Spec.Services[serviceConfig.Name].MongoEnvironment,
		site.Spec.Services[serviceConfig.Name].RedisEnvironment,
	)
	if nil != err {
		return nil, err
	}

	podSpec, err := helpers.ReplaceTemplateVariablesInPodSpec(*serviceConfig.Spec.BackupPodSpec, &templateHandler)
	if nil != err {
		return nil, err
	}

	if podSpec.RestartPolicy != corev1.RestartPolicyOnFailure && podSpec.RestartPolicy != corev1.RestartPolicyNever {
		podSpec.RestartPolicy = corev1.RestartPolicyOnFailure
	}

	batchJob := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.getBatchJobName(serviceConfig.Name, job.Name),
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
				Spec: pod.SetExtraEnvVarsOnPodSpec(podSpec, &site, &serviceConfig),
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
func (r *BackupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// set up a real clock, since we're not in a test
	if r.Clock == nil {
		r.Clock = realClock{}
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&jobv1.Backup{}).
		Owns(&batchv1.Job{}).
		Complete(r)
}
