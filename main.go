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

package main

import (
	"flag"
	"github.com/getsentry/sentry-go"
	"os"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/webhook"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	webhook2 "github.com/szeber/kube-stager/handlers/webhook"

	configv1 "github.com/szeber/kube-stager/apis/config/v1"
	controllerconfigv1 "github.com/szeber/kube-stager/apis/controller-config/v1"
	jobv1 "github.com/szeber/kube-stager/apis/job/v1"
	sitev1 "github.com/szeber/kube-stager/apis/site/v1"
	taskv1 "github.com/szeber/kube-stager/apis/task/v1"
	jobcontrollers "github.com/szeber/kube-stager/controllers/job"
	sitecontrollers "github.com/szeber/kube-stager/controllers/site"
	taskcontrollers "github.com/szeber/kube-stager/controllers/task"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(configv1.AddToScheme(scheme))
	utilruntime.Must(taskv1.AddToScheme(scheme))
	utilruntime.Must(jobv1.AddToScheme(scheme))
	utilruntime.Must(sitev1.AddToScheme(scheme))
	utilruntime.Must(controllerconfigv1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var configFile string
	flag.StringVar(
		&configFile, "config", "",
		"The controller will load its initial configuration from this file. "+
			"Omit this flag to use the default configuration values. "+
			"Command-line flags override configuration from this file.",
	)
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	var err error
	ctrlConfig := controllerconfigv1.ProjectConfig{
		InitJobConfig: controllerconfigv1.JobConfig{
			DeadlineSeconds: 600,
			TtlSeconds:      600,
			BackoffLimit:    0,
		},
		MigrationJobConfig: controllerconfigv1.JobConfig{
			DeadlineSeconds: 600,
			TtlSeconds:      600,
			BackoffLimit:    3,
		},
		BackupJobConfig: controllerconfigv1.JobConfig{
			DeadlineSeconds: 600,
			TtlSeconds:      600,
			BackoffLimit:    3,
		},
	}
	options := ctrl.Options{
		Scheme:                 scheme,
		LeaderElectionID:       "ec56737d.operator.kube-stager.io",
		Port:                   9443,
		MetricsBindAddress:     ":8080",
		HealthProbeBindAddress: ":8081",
		LeaderElection:         false,
	}
	if configFile != "" {
		options, err = options.AndFrom(ctrl.ConfigFile().AtPath(configFile).OfKind(&ctrlConfig))
		if err != nil {
			setupLog.Error(err, "unable to load the config file")
			os.Exit(1)
		}
	}

	setupLog.Info("Finished loading config", "config", ctrlConfig)

	if "" != ctrlConfig.SentryDsn {
		err := sentry.Init(
			sentry.ClientOptions{
				Dsn: ctrlConfig.SentryDsn,
			},
		)
		if err != nil {
			setupLog.Error(err, "Sentry init failed")
			os.Exit(1)
		}
		setupLog.Info("Sentry init complete")

		defer sentry.Flush(2 * time.Second)
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), options)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&taskcontrollers.MysqlDatabaseReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "MysqlDatabase")
		os.Exit(1)
	}
	if err = (&taskcontrollers.MongoDatabaseReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "MongoDatabase")
		os.Exit(1)
	}
	if err = (&taskcontrollers.RedisDatabaseReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "RedisDatabase")
		os.Exit(1)
	}
	if err = (&jobcontrollers.DbInitJobReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Config: ctrlConfig,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "DbInitJob")
		os.Exit(1)
	}
	if err = (&jobcontrollers.DbMigrationJobReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Config: ctrlConfig,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "DbMigrationJob")
		os.Exit(1)
	}
	if err = (&sitecontrollers.StagingSiteReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "StagingSite")
		os.Exit(1)
	}
	if err = (&sitev1.StagingSite{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "StagingSite")
		os.Exit(1)
	}
	if err = (&jobcontrollers.BackupReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Config: ctrlConfig,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Backup")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	setupLog.Info("registering advanced webhooks to the webhook server")

	mgr.GetWebhookServer().Register(
		"/validate-config-operator-kube-stager-io-v1-serviceconfig",
		&webhook.Admission{Handler: &webhook2.ServiceConfigCreateOrUpdateHandler{Client: mgr.GetClient()}},
	)
	mgr.GetWebhookServer().Register(
		"/validate-config-operator-kube-stager-io-v1-serviceconfig-deletion",
		&webhook.Admission{Handler: &webhook2.ServiceConfigDeleteHandler{Client: mgr.GetClient()}},
	)
	mgr.GetWebhookServer().Register(
		"/validate-config-operator-kube-stager-io-v1-mongoconfig-deletion",
		&webhook.Admission{Handler: &webhook2.MongoConfigDeleteHandler{Client: mgr.GetClient()}},
	)
	mgr.GetWebhookServer().Register(
		"/validate-config-operator-kube-stager-io-v1-mysqlconfig-deletion",
		&webhook.Admission{Handler: &webhook2.MysqlConfigDeleteHandler{Client: mgr.GetClient()}},
	)
	mgr.GetWebhookServer().Register(
		"/validate-config-operator-kube-stager-io-v1-redisconfig-deletion",
		&webhook.Admission{Handler: &webhook2.RedisConfigDeleteHandler{Client: mgr.GetClient()}},
	)
	mgr.GetWebhookServer().Register(
		"/mutate-site-operator-kube-stager-io-v1-stagingsite-advanced",
		&webhook.Admission{Handler: &webhook2.StagingsiteHandler{Client: mgr.GetClient()}},
	)
	mgr.GetWebhookServer().Register(
		"/mutate-job-operator-kube-stager-io-v1-backup-advanced",
		&webhook.Admission{
			Handler: &webhook2.BackupCreateOrUpdateHandler{
				Client: mgr.GetClient(),
				Scheme: mgr.GetScheme(),
			},
		},
	)

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

//+kubebuilder:webhook:path=/validate-config-operator-kube-stager-io-v1-serviceconfig,mutating=false,failurePolicy=fail,groups="config.operator.kube-stager.io",resources=serviceconfigs,verbs=create;update,versions=v1,name=serviceconfig-handler.operator.kube-stager.io,sideEffects=none,admissionReviewVersions={v1,v1beta1}
//+kubebuilder:webhook:path=/validate-config-operator-kube-stager-io-v1-serviceconfig-deletion,mutating=false,failurePolicy=fail,groups="config.operator.kube-stager.io",resources=serviceconfigs,verbs=delete,versions=v1,name=serviceconfig-delete-handler.operator.kube-stager.io,sideEffects=none,admissionReviewVersions={v1,v1beta1}
//+kubebuilder:webhook:path=/validate-config-operator-kube-stager-io-v1-mongoconfig-deletion,mutating=false,failurePolicy=fail,groups="config.operator.kube-stager.io",resources=mongoconfigs,verbs=delete,versions=v1,name=mongoconfig-delete-handler.operator.kube-stager.io,sideEffects=none,admissionReviewVersions={v1,v1beta1}
//+kubebuilder:webhook:path=/validate-config-operator-kube-stager-io-v1-mysqlconfig-deletion,mutating=false,failurePolicy=fail,groups="config.operator.kube-stager.io",resources=mysqlconfigs,verbs=delete,versions=v1,name=mysqlconfig-delete-handler.operator.kube-stager.io,sideEffects=none,admissionReviewVersions={v1,v1beta1}
//+kubebuilder:webhook:path=/validate-config-operator-kube-stager-io-v1-redisconfig-deletion,mutating=false,failurePolicy=fail,groups="config.operator.kube-stager.io",resources=redisconfigs,verbs=delete,versions=v1,name=redisconfig-delete-handler.operator.kube-stager.io,sideEffects=none,admissionReviewVersions={v1,v1beta1}
//+kubebuilder:webhook:path=/mutate-site-operator-kube-stager-io-v1-stagingsite-advanced,mutating=true,failurePolicy=fail,groups="site.operator.kube-stager.io",resources=stagingsites,verbs=create;update,versions=v1,name=stagingsite-handler.operator.kube-stager.io,sideEffects=none,admissionReviewVersions={v1,v1beta1}
//+kubebuilder:webhook:path=/mutate-job-operator-kube-stager-io-v1-backup-advanced,mutating=true,failurePolicy=fail,groups="job.operator.kube-stager.io",resources=backups,verbs=create;update,versions=v1,name=backup-handler.operator.kube-stager.io,sideEffects=none,admissionReviewVersions={v1,v1beta1}
