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

package task

import (
	"context"
	"github.com/getsentry/sentry-go"
	configv1 "github.com/szeber/kube-stager/apis/config/v1"
	controller "github.com/szeber/kube-stager/controllers"
	"github.com/szeber/kube-stager/handlers/database"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	taskv1 "github.com/szeber/kube-stager/apis/task/v1"
)

// RedisDatabaseReconciler reconciles a RedisDatabase object
type RedisDatabaseReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=task.operator.kube-stager.io,resources=redisdatabases,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=task.operator.kube-stager.io,resources=redisdatabases/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=task.operator.kube-stager.io,resources=redisdatabases/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *RedisDatabaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	result, err := r.doReconcile(ctx, req)

	if nil != err {
		sentry.CaptureException(err)
	}

	return result, err
}

func (r *RedisDatabaseReconciler) doReconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var db taskv1.RedisDatabase

	if err := r.Get(ctx, req.NamespacedName, &db); nil != err {
		if client.IgnoreNotFound(err) != nil {
			logger.Error(err, "unable to fetch database")
		}

		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.Info("Fetched database, fetching config")

	var config configv1.RedisConfig

	configKey := client.ObjectKey{Namespace: db.Namespace, Name: db.Spec.EnvironmentConfig.Environment}
	if err := r.Get(ctx, configKey, &config); nil != err {
		return ctrl.Result{}, err
	}

	changed, err := database.ReconcileRedis(&db, config, logger)

	return controller.SaveStatusUpdatesIfObjectChanged(changed, r.Status(), ctx, &db, ctrl.Result{}, err)
}

// SetupWithManager sets up the controller with the Manager.
func (r *RedisDatabaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&taskv1.RedisDatabase{}).
		Complete(r)
}
