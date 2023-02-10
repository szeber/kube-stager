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

package site

import (
	"context"
	"errors"
	"fmt"
	"github.com/getsentry/sentry-go"
	"github.com/go-logr/logr"
	api "github.com/szeber/kube-stager/apis"
	configv1 "github.com/szeber/kube-stager/apis/config/v1"
	jobv1 "github.com/szeber/kube-stager/apis/job/v1"
	taskv1 "github.com/szeber/kube-stager/apis/task/v1"
	controller "github.com/szeber/kube-stager/controllers"
	"github.com/szeber/kube-stager/handlers/job"
	sitehandler "github.com/szeber/kube-stager/handlers/site"
	"github.com/szeber/kube-stager/handlers/task"
	"github.com/szeber/kube-stager/helpers"
	"github.com/szeber/kube-stager/helpers/annotations"
	errorhelpers "github.com/szeber/kube-stager/helpers/errors"
	"github.com/szeber/kube-stager/helpers/indexes"
	"hash/fnv"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sort"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	sitev1 "github.com/szeber/kube-stager/apis/site/v1"
)

// StagingSiteReconciler reconciles a StagingSite object
type StagingSiteReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Clock
}

type realClock struct{}

func (_ realClock) Now() time.Time {
	return time.Now()
}

type Clock interface {
	Now() time.Time
}

//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=config.operator.kube-stager.io,resources=mongoconfigs,verbs=get;list;watch
//+kubebuilder:rbac:groups=config.operator.kube-stager.io,resources=mysqlconfigs,verbs=get;list;watch
//+kubebuilder:rbac:groups=config.operator.kube-stager.io,resources=redisconfigs,verbs=get;list;watch
//+kubebuilder:rbac:groups=config.operator.kube-stager.io,resources=serviceconfigs,verbs=get;list;watch;update;patch;
//+kubebuilder:rbac:groups=task.operator.kube-stager.io,resources=mongodatabases,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=task.operator.kube-stager.io,resources=mysqldatabases,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=task.operator.kube-stager.io,resources=redisdatabases,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=site.operator.kube-stager.io,resources=stagingsites,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=site.operator.kube-stager.io,resources=stagingsites/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=site.operator.kube-stager.io,resources=stagingsites/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *StagingSiteReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	result, err := r.doReconcile(ctx, req)

	if nil != err {
		sentry.CaptureException(err)
	}

	return result, err
}

func (r *StagingSiteReconciler) doReconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	site := &sitev1.StagingSite{}

	if err := r.Get(ctx, req.NamespacedName, site); err != nil {
		if client.IgnoreNotFound(err) != nil {
			logger.Error(err, "unable to fetch staging site")
		}

		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.V(0).Info("Fetched staging site " + site.Name)

	if nil != site.DeletionTimestamp {
		return r.handleDelete(ctx, logger, site)
	}

	if !helpers.SliceContainsString(site.Finalizers, helpers.SiteFinalizerName) {
		return r.appendFinalizer(ctx, site)
	}

	logger.V(0).Info("Ensuring the status is up to date")
	if changed, err := r.ensureStatusIsUpToDate(site, ctx); nil != err || changed {
		result := ctrl.Result{}
		if changed && nil == err {
			result = ctrl.Result{Requeue: true}
		}
		logger.V(0).Info("Status changed", "result", result)
		return r.SaveStatusUpdatesIfObjectChanged(changed, ctx, site, result, err)
	}

	r.resetStatusErrors(site)

	if nil != site.Status.DeleteAt && site.Status.DeleteAt.Time.Before(r.Clock.Now()) {
		err := r.Delete(ctx, site)
		return ctrl.Result{}, err
	}

	isSiteChanged := false
	logger.V(0).Info("Ensuring databases are created")
	if changed, err := r.ensureDatabasesAreCreated(site, ctx); nil != err {
		return r.SaveStatusUpdatesIfObjectChanged(changed, ctx, site, ctrl.Result{}, err)
	} else {
		isSiteChanged = changed
	}

	logger.V(0).Info("Ensuring configs are up to date")
	if changed, err := r.ensureConfigsAreUpToDate(site, ctx); nil != err {
		return r.SaveStatusUpdatesIfObjectChanged(isSiteChanged || changed, ctx, site, ctrl.Result{}, err)
	} else {
		isSiteChanged = isSiteChanged || changed
	}
	if !site.Status.ConfigsAreCreated || !site.Status.DatabaseCreationComplete {
		// DB init requires created dbs and may need configs, so wait till it's done
		return r.SaveStatusUpdatesIfObjectChanged(isSiteChanged, ctx, site, ctrl.Result{}, nil)
	}

	logger.V(0).Info("Ensuring databases are initialised")
	if changed, err := r.ensureDatabasesAreInitialised(site, ctx); nil != err {
		return r.SaveStatusUpdatesIfObjectChanged(isSiteChanged || changed, ctx, site, ctrl.Result{}, err)
	} else {
		isSiteChanged = isSiteChanged || changed
	}
	if !site.Status.DatabaseInitialisationComplete {
		// DB migration needs the init to complete first, so wait till it's done
		return r.SaveStatusUpdatesIfObjectChanged(isSiteChanged, ctx, site, ctrl.Result{}, nil)
	}

	logger.V(0).Info("Ensuring databases are migrated")
	if changed, err := r.ensureDatabaseMigrationJobsAreCreated(site, ctx); nil != err {
		return r.SaveStatusUpdatesIfObjectChanged(isSiteChanged || changed, ctx, site, ctrl.Result{}, err)
	} else {
		isSiteChanged = isSiteChanged || changed
	}
	if !site.Status.DatabaseMigrationsComplete {
		// To avoid the deployment running into issues wait until the migrations are complete
		return r.SaveStatusUpdatesIfObjectChanged(isSiteChanged, ctx, site, ctrl.Result{}, nil)
	}

	logger.V(0).Info("Ensuring workloads are up to date")
	if changed, err := r.ensureWorkloadObjectsAreUpToDate(site, ctx); nil != err {
		return r.SaveStatusUpdatesIfObjectChanged(isSiteChanged || changed, ctx, site, ctrl.Result{}, err)
	} else {
		isSiteChanged = isSiteChanged || changed
	}

	logger.V(0).Info("Ensuring networking objects are up to date")
	if changed, err := r.ensureNetworkingObjectsAreUpToDate(site, ctx); nil != err {
		return r.SaveStatusUpdatesIfObjectChanged(isSiteChanged || changed, ctx, site, ctrl.Result{}, err)
	} else {
		isSiteChanged = isSiteChanged || changed
	}

	if site.Status.State != sitev1.StateComplete {
		site.Status.State = sitev1.StateComplete
		isSiteChanged = true
	}

	if nil != site.Status.NextBackupTime && site.Status.NextBackupTime.Time.Before(r.Clock.Now()) {
		changed, err := r.handleBackup(site, ctx)
		isSiteChanged = isSiteChanged || changed
		if nil != err {
			return r.SaveStatusUpdatesIfObjectChanged(
				isSiteChanged,
				ctx,
				site,
				r.getCtrlResultWithRecheckInterval(ctx, site),
				err,
			)
		}
	}

	return r.SaveStatusUpdatesIfObjectChanged(
		isSiteChanged,
		ctx,
		site,
		r.getCtrlResultWithRecheckInterval(ctx, site),
		nil,
	)
}

func (r *StagingSiteReconciler) handleBackup(site *sitev1.StagingSite, ctx context.Context) (bool, error) {
	handler := job.BackupHandler{
		Reader: r,
		Writer: r,
		Scheme: r.Scheme,
	}

	err := handler.Create(site, ctx, jobv1.BackupTypeScheduled, r.Clock.Now())
	if nil != err {
		return false, err
	}

	site.Status.NextBackupTime, err = r.getNextBackupTimeForSite(site)
	if nil != err {
		return true, err
	}

	return true, nil
}

func (r *StagingSiteReconciler) appendFinalizer(ctx context.Context, site *sitev1.StagingSite) (ctrl.Result, error) {
	site.Finalizers = append(site.Finalizers, helpers.SiteFinalizerName)
	if err := r.Update(ctx, site); nil != err {
		return ctrl.Result{}, err
	}
	return ctrl.Result{Requeue: true}, nil
}

func (r *StagingSiteReconciler) handleDelete(ctx context.Context, logger logr.Logger, site *sitev1.StagingSite) (
	ctrl.Result,
	error,
) {
	logger.Info("Performing pre-deletion tasks")
	if !helpers.SliceContainsString(site.Finalizers, helpers.SiteFinalizerName) {
		logger.Info("Pre deletion tasks already completed")
		// The finalizer is already complete, ignore
		return ctrl.Result{}, nil
	}

	if site.Status.State != sitev1.StatePending {
		logger.Info("Setting the site state to pending")
		site.Status.State = sitev1.StatePending
		return r.SaveStatusUpdatesIfObjectChanged(true, ctx, site, ctrl.Result{Requeue: true}, nil)
	}

	if site.Spec.BackupBeforeDelete {
		logger.Info("Making a backup of the databases before deletion")
		backupHandler := job.BackupHandler{
			Reader: r,
			Writer: r,
			Scheme: r.Scheme,
		}
		if complete, err := backupHandler.EnsureFinalBackupIsComplete(site, ctx); nil != err || !complete {
			return ctrl.Result{}, err
		}
	}

	logger.Info("Pre deletion tasks complete, removing finalizer")
	site.Finalizers = helpers.RemoveStringFromSlice(site.Finalizers, helpers.SiteFinalizerName)
	err := r.Update(ctx, site)
	return ctrl.Result{}, err
}

func (r *StagingSiteReconciler) getCtrlResultWithRecheckInterval(
	ctx context.Context,
	site *sitev1.StagingSite,
) ctrl.Result {
	now := r.Clock.Now()
	times := []time.Time{}
	logger := log.FromContext(ctx)

	logger.V(0).Info("Scheduling next site check")

	if nil != site.Status.DisableAt && site.Status.Enabled {
		if site.Status.DisableAt.Time.Before(now) {
			return ctrl.Result{Requeue: true}
		}

		times = append(times, site.Status.DisableAt.Time)
	}

	if nil != site.Status.DeleteAt {
		if site.Status.DeleteAt.Time.Before(now) {
			return ctrl.Result{Requeue: true}
		}
		times = append(times, site.Status.DeleteAt.Time)
	}

	if nil != site.Status.NextBackupTime {
		if site.Status.NextBackupTime.Time.Before(now) {
			return ctrl.Result{Requeue: true}
		}
		times = append(times, site.Status.NextBackupTime.Time)
	}

	if 0 == len(times) {
		logger.V(0).Info("No scheduled checks necessary")
		return ctrl.Result{}
	}

	sort.Slice(times, func(i, j int) bool { return times[i].Before(times[j]) })

	logger.V(0).Info("Next recheck at", "nextCheck", times[0].String(), "nextCheckIn", times[0].Sub(now))
	return ctrl.Result{RequeueAfter: times[0].Sub(now)}
}

func (r *StagingSiteReconciler) ensureStatusIsUpToDate(site *sitev1.StagingSite, ctx context.Context) (bool, error) {
	isChanged := false

	var err error
	lastSpecChangedAt := time.Time{}

	if "" == site.Status.State {
		site.Status.State = sitev1.StatePending
	}

	if "" == site.Annotations[annotations.StagingSiteLastSpecChangeAt] {
		lastSpecChangedAt = r.Clock.Now()
		site.Annotations[annotations.StagingSiteLastSpecChangeAt] = lastSpecChangedAt.Format(time.RFC3339)
	} else if lastSpecChangedAt, err = time.Parse(
		time.RFC3339,
		site.Annotations[annotations.StagingSiteLastSpecChangeAt],
	); nil != err {
		lastSpecChangedAt = r.Clock.Now()
		site.Annotations[annotations.StagingSiteLastSpecChangeAt] = lastSpecChangedAt.Format(time.RFC3339)
	}

	if site.Spec.DisableAfter.Never {
		if nil != site.Status.DisableAt {
			site.Status.DisableAt = nil
			isChanged = true
		}
	} else {
		disableAt := &metav1.Time{Time: lastSpecChangedAt.Add(site.Spec.DisableAfter.ToDuration())}
		if nil == site.Status.DisableAt || !site.Status.DisableAt.Equal(disableAt) {
			site.Status.DisableAt = disableAt
			isChanged = true
		}
	}

	if site.Spec.DeleteAfter.Never {
		if nil != site.Status.DeleteAt {
			site.Status.DeleteAt = nil
			isChanged = true
		}
	} else {
		deleteAt := &metav1.Time{Time: lastSpecChangedAt.Add(site.Spec.DeleteAfter.ToDuration())}
		if nil == site.Status.DeleteAt || !site.Status.DeleteAt.Equal(deleteAt) {
			site.Status.DeleteAt = deleteAt
			isChanged = true
		}
	}

	expectedEnabled := site.Spec.Enabled && (nil == site.Status.DisableAt || site.Status.DisableAt.Time.After(r.Clock.Now()))

	if site.Status.Enabled != expectedEnabled {
		site.Status.Enabled = expectedEnabled
		site.Status.State = sitev1.StatePending
		isChanged = true
	}

	if nil == site.Status.LastAppliedConfiguration || site.Status.LastAppliedConfiguration.Time.Before(lastSpecChangedAt) {
		site.Status.LastAppliedConfiguration = &metav1.Time{Time: lastSpecChangedAt}
		isChanged = true
	}

	serviceConfigs := map[string]*configv1.ServiceConfig{}

	for name := range site.Spec.Services {
		config := &configv1.ServiceConfig{}
		if err := r.Get(ctx, client.ObjectKey{Namespace: site.Namespace, Name: name}, config); nil != err {
			return false, err
		}
		serviceConfigs[config.Name] = config
	}

	for name, serviceStatus := range site.Status.Services {
		if config, ok := serviceConfigs[name]; ok {
			expectedUsername := api.MakeUsername(site, config)
			expectedDbName := api.MakeDatabaseName(site, config)
			if serviceStatus.Username != expectedUsername || serviceStatus.DbName != expectedDbName {
				serviceStatus.Username = expectedUsername
				serviceStatus.DbName = expectedDbName
				site.Status.Services[name] = serviceStatus
			}
			delete(serviceConfigs, name)
		} else {
			isChanged = true
			delete(site.Status.Services, name)
		}
	}

	isChanged = isChanged || 0 != len(serviceConfigs)

	if 0 == len(site.Status.Services) {
		site.Status.Services = make(map[string]sitev1.StagingSiteServiceStatus)
	}

	for name, config := range serviceConfigs {
		site.Status.Services[name] = sitev1.StagingSiteServiceStatus{
			Username:         api.MakeUsername(site, config),
			DbName:           api.MakeDatabaseName(site, config),
			DeploymentStatus: appsv1.DeploymentStatus{},
		}
	}

	if "" == site.Status.WorkloadHealth {
		site.Status.WorkloadHealth = sitev1.WorkloadHealthIncomplete
	}

	if nil != site.Spec.DailyBackupWindowHour {
		if nil == site.Status.NextBackupTime || site.Status.NextBackupTime.Hour() != int(*site.Spec.DailyBackupWindowHour) {
			site.Status.NextBackupTime, err = r.getNextBackupTimeForSite(site)
			if nil != err {
				return isChanged, err
			}
			isChanged = true
		}
	} else if nil != site.Status.NextBackupTime {
		site.Status.NextBackupTime = nil
		isChanged = true
	}

	var backupList jobv1.BackupList
	var finishedBackups []jobv1.Backup
	if err = r.List(
		ctx,
		&backupList,
		client.InNamespace(site.Namespace),
		client.MatchingFields{indexes.SiteName: site.Name},
	); nil != err {
		return isChanged, err
	}

	lastBackupTime := site.Status.LastBackupTime
	backupControlClaimed := false

	for _, backup := range backupList.Items {
		if jobv1.Complete == backup.Status.State && nil != backup.Status.JobFinishedAt {
			if nil == lastBackupTime || lastBackupTime.Before(backup.Status.JobFinishedAt) {
				lastBackupTime = backup.Status.JobFinishedAt
				isChanged = true
			}
		}
		if backup.Status.State.IsFinal() {
			finishedBackups = append(finishedBackups, backup)
		}
		if nil == metav1.GetControllerOf(&backup) {
			if err = ctrl.SetControllerReference(site, &backup, r.Scheme); nil != err {
				return isChanged, nil
			}
			if err = r.Update(ctx, &backup); nil != err {
				return isChanged, nil
			}
			backupControlClaimed = true
		}
	}

	if nil != lastBackupTime && (nil == site.Status.LastBackupTime || site.Status.LastBackupTime.Before(lastBackupTime)) {
		site.Status.LastBackupTime = lastBackupTime
		isChanged = true
	}

	if backupControlClaimed {
		return true, nil
	}

	if len(finishedBackups) > 3 {
		sort.Slice(
			finishedBackups,
			func(i, j int) bool {
				if finishedBackups[i].Status.JobStartedAt == nil {
					return finishedBackups[j].Status.JobStartedAt != nil
				}
				return finishedBackups[i].Status.JobStartedAt.Before(finishedBackups[j].Status.JobStartedAt)
			},
		)

		for i, backupJob := range finishedBackups {
			if int32(i) >= int32(len(finishedBackups))-3 {
				break
			}
			if err := r.Delete(ctx, &backupJob); nil != err {
				return isChanged, err
			}
			log.FromContext(ctx).V(1).Info("Cleaned up old backup", "backup", backupJob)
		}
	}

	return isChanged, nil
}

func (r *StagingSiteReconciler) resetStatusErrors(site *sitev1.StagingSite) {
	if sitev1.StateFailed == site.Status.State {
		// If we are in a failed state reset to pending and clear the error message so they can be set and populated if
		// the child resources still require that
		site.Status.State = sitev1.StatePending
		site.Status.ErrorMessage = ""
	}
}

func (r *StagingSiteReconciler) getNextBackupTimeForSite(site *sitev1.StagingSite) (*metav1.Time, error) {
	if nil == site.Spec.DailyBackupWindowHour || *site.Spec.DailyBackupWindowHour < 0 {
		return nil, nil
	}

	hash := fnv.New32a()
	if _, err := hash.Write([]byte(site.Name)); nil != err {
		return nil, err
	}

	offsetSeconds := hash.Sum32() % 3600

	minutes := offsetSeconds / 60
	seconds := offsetSeconds % 60

	now := r.Clock.Now()
	nextRunAt, err := time.Parse(
		time.RFC3339,
		fmt.Sprintf(
			"%04d-%02d-%02dT%02d:%02d:%02dZ",
			now.Year(),
			now.Month(),
			now.Day(),
			*site.Spec.DailyBackupWindowHour,
			minutes,
			seconds,
		),
	)
	if nil != err {
		return nil, err
	}

	for nextRunAt.Before(now) {
		nextRunAt = nextRunAt.Add(24 * time.Hour)
	}

	return &metav1.Time{Time: nextRunAt}, nil
}

func (r *StagingSiteReconciler) ensureDatabasesAreCreated(site *sitev1.StagingSite, ctx context.Context) (bool, error) {
	handlers := []task.TaskHandler{
		task.MysqlTaskHandler{Reader: r, Writer: r, Scheme: r.Scheme},
		task.MongoTaskHandler{Reader: r, Writer: r, Scheme: r.Scheme},
		task.RedisTaskHandler{Reader: r, Writer: r, Scheme: r.Scheme},
	}

	originalCompletion := site.Status.DatabaseCreationComplete
	site.Status.DatabaseCreationComplete = true

	for _, handler := range handlers {
		if isComplete, err := handler.EnsureDatabasesAreCreated(site, ctx); nil != err {
			return false, err
		} else {
			site.Status.DatabaseCreationComplete = site.Status.DatabaseCreationComplete && isComplete
		}
	}

	site.Status.DatabaseInitialisationComplete = true

	for _, handler := range handlers {
		if isComplete, err := handler.EnsureDatabasesAreReady(site, ctx); nil != err {
			return false, err
		} else {
			site.Status.DatabaseCreationComplete = site.Status.DatabaseCreationComplete && isComplete
		}
	}

	return originalCompletion != site.Status.DatabaseCreationComplete, nil
}

func (r *StagingSiteReconciler) ensureDatabasesAreInitialised(site *sitev1.StagingSite, ctx context.Context) (
	bool,
	error,
) {
	handler := job.DbInitJobHandler{Reader: r, Writer: r, Scheme: r.Scheme}

	originalCompletion := site.Status.DatabaseInitialisationComplete
	site.Status.DatabaseInitialisationComplete = true

	isComplete, err := handler.EnsureJobsAreCreated(site, ctx)
	if nil != err {
		return false, err
	}
	site.Status.DatabaseInitialisationComplete = site.Status.DatabaseInitialisationComplete && isComplete

	isComplete, err = handler.EnsureJobsAreComplete(site, ctx)
	if nil != err {
		return false, err
	}
	site.Status.DatabaseInitialisationComplete = site.Status.DatabaseInitialisationComplete && isComplete

	return originalCompletion != site.Status.DatabaseInitialisationComplete, nil
}

func (r *StagingSiteReconciler) ensureDatabaseMigrationJobsAreCreated(
	site *sitev1.StagingSite,
	ctx context.Context,
) (bool, error) {
	handler := job.DbMigrationJobHandler{Reader: r, Writer: r, Scheme: r.Scheme}

	originalCompletion := site.Status.DatabaseMigrationsComplete
	site.Status.DatabaseMigrationsComplete = true

	isComplete, err := handler.EnsureJobsAreCreated(site, ctx)
	if nil != err {
		return false, err
	}
	site.Status.DatabaseMigrationsComplete = site.Status.DatabaseMigrationsComplete && isComplete

	isComplete, err = handler.EnsureJobsAreComplete(site, ctx)
	if nil != err {
		return false, err
	}
	site.Status.DatabaseMigrationsComplete = site.Status.DatabaseMigrationsComplete && isComplete

	return originalCompletion != site.Status.DatabaseMigrationsComplete, nil
}

func (r *StagingSiteReconciler) ensureConfigsAreUpToDate(site *sitev1.StagingSite, ctx context.Context) (bool, error) {
	handler := sitehandler.ConfigHandler{Reader: r, Writer: r, Scheme: r.Scheme}
	isChanged := false

	if changed, err := handler.EnsureConfigsAreUpToDate(site, ctx); nil != err {
		return false, err
	} else {
		isChanged = changed
	}
	return isChanged, nil
}

func (r *StagingSiteReconciler) ensureWorkloadObjectsAreUpToDate(site *sitev1.StagingSite, ctx context.Context) (
	bool,
	error,
) {
	handler := sitehandler.WorkloadHandler{Reader: r, Writer: r, Scheme: r.Scheme}
	isChanged := false

	if changed, err := handler.EnsureWorkloadObjectsAreUpToDate(site, ctx); nil != err {
		return false, err
	} else {
		isChanged = changed
	}
	return isChanged, nil
}

func (r *StagingSiteReconciler) ensureNetworkingObjectsAreUpToDate(site *sitev1.StagingSite, ctx context.Context) (
	bool,
	error,
) {
	handler := sitehandler.NetworkingHandler{Reader: r, Writer: r, Scheme: r.Scheme}
	isChanged := false

	if changed, err := handler.EnsureNetworkingObjectsAreUpToDate(site, ctx); nil != err {
		return false, err
	} else {
		isChanged = changed
	}
	return isChanged, nil
}

func (r *StagingSiteReconciler) SaveStatusUpdatesIfObjectChanged(
	isChanged bool,
	ctx context.Context,
	site *sitev1.StagingSite,
	result ctrl.Result,
	err error,
) (ctrl.Result, error) {
	if nil != err {
		var controllerError errorhelpers.ControllerError
		if errors.As(err, &controllerError) {
			if controllerError.IsFinal() {
				log.FromContext(ctx).Error(controllerError, "Received final controller error. Failing site")
				site.Status.State = sitev1.StateFailed
				site.Status.ErrorMessage = err.Error()
				isChanged = true
				result = ctrl.Result{}
				err = nil
			}
		}
		if sitev1.StateFailed == site.Status.State && !isChanged {
			isChanged = true
		}
	}

	r.setSiteState(site)

	return controller.SaveStatusUpdatesIfObjectChanged(isChanged, r.Status(), ctx, site, result, err)
}

func (r *StagingSiteReconciler) setSiteState(site *sitev1.StagingSite) {
	if sitev1.StateFailed == site.Status.State {
		return
	}

	status := site.Status

	if status.DatabaseCreationComplete &&
		status.DatabaseInitialisationComplete &&
		status.DatabaseMigrationsComplete &&
		status.ConfigsAreCreated &&
		status.NetworkingObjectsAreCreated &&
		status.WorkloadsAreCreated &&
		nil == site.DeletionTimestamp {
		site.Status.State = sitev1.StateComplete
	} else {
		site.Status.State = sitev1.StatePending
	}

	if status.State != sitev1.StateComplete {
		status.WorkloadHealth = sitev1.WorkloadHealthIncomplete
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *StagingSiteReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// set up a real clock, since we're not in a test
	if r.Clock == nil {
		r.Clock = realClock{}
	}

	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(), &configv1.ServiceConfig{}, indexes.ShortName, func(rawObj client.Object) []string {
			// grab the config object, extract the short name.
			config := rawObj.(*configv1.ServiceConfig)
			return []string{config.Spec.ShortName}
		},
	); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&configv1.ServiceConfig{},
		indexes.DefaultMongoEnvironment,
		func(rawObj client.Object) []string {
			config := rawObj.(*configv1.ServiceConfig)
			return []string{fmt.Sprintf("%v", config.Spec.DefaultMongoEnvironment)}
		},
	); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&configv1.ServiceConfig{},
		indexes.DefaultMysqlEnvironment,
		func(rawObj client.Object) []string {
			config := rawObj.(*configv1.ServiceConfig)
			return []string{fmt.Sprintf("%v", config.Spec.DefaultMysqlEnvironment)}
		},
	); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&configv1.ServiceConfig{},
		indexes.DefaultRedisEnvironment,
		func(rawObj client.Object) []string {
			config := rawObj.(*configv1.ServiceConfig)
			return []string{fmt.Sprintf("%v", config.Spec.DefaultRedisEnvironment)}
		},
	); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(), &jobv1.Backup{}, indexes.SiteName, func(rawObj client.Object) []string {
			// grab the config object, extract the short name.
			config := rawObj.(*jobv1.Backup)
			return []string{config.Spec.SiteName}
		},
	); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&sitev1.StagingSite{}).
		Owns(&taskv1.MongoDatabase{}).
		Owns(&taskv1.MysqlDatabase{}).
		Owns(&taskv1.RedisDatabase{}).
		Owns(&jobv1.DbInitJob{}).
		Owns(&jobv1.DbMigrationJob{}).
		Owns(&appsv1.Deployment{}).
		Owns(&jobv1.Backup{}).
		Complete(r)
}
