package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	namespace = "kube_stager"
)

var factory = promauto.With(metrics.Registry)

// BuildInfo exposes build metadata. Set once at startup with version and Go version.
var BuildInfo = factory.NewGaugeVec(prometheus.GaugeOpts{
	Namespace: namespace,
	Name:      "build_info",
	Help:      "Build information for the kube-stager operator. Always 1.",
}, []string{"version", "go_version"})

// SiteStateTransitions counts state transitions on StagingSite resources.
var SiteStateTransitions = factory.NewCounterVec(prometheus.CounterOpts{
	Namespace: namespace,
	Name:      "site_state_transitions_total",
	Help:      "Total number of StagingSite state transitions.",
}, []string{"namespace", "from_state", "to_state"})

// SiteAutoDisabled counts sites auto-disabled by timeout.
var SiteAutoDisabled = factory.NewCounterVec(prometheus.CounterOpts{
	Namespace: namespace,
	Name:      "site_auto_disabled_total",
	Help:      "Total number of StagingSites automatically disabled by timeout.",
}, []string{"namespace"})

// SiteAutoDeleted counts sites auto-deleted by timeout.
var SiteAutoDeleted = factory.NewCounterVec(prometheus.CounterOpts{
	Namespace: namespace,
	Name:      "site_auto_deleted_total",
	Help:      "Total number of StagingSites automatically deleted by timeout.",
}, []string{"namespace"})

// SiteProvisioningDuration tracks time from StagingSite creation to Complete state.
var SiteProvisioningDuration = factory.NewHistogramVec(prometheus.HistogramOpts{
	Namespace: namespace,
	Name:      "site_provisioning_duration_seconds",
	Help:      "Time from StagingSite creation to Complete state in seconds.",
	Buckets:   []float64{10, 30, 60, 120, 300, 600, 1800, 3600},
}, []string{"namespace"})

// DatabaseOperations counts database provisioning operations.
var DatabaseOperations = factory.NewCounterVec(prometheus.CounterOpts{
	Namespace: namespace,
	Name:      "database_operations_total",
	Help:      "Total database provisioning operations.",
}, []string{"type", "operation", "result"})

// DatabaseOperationDuration tracks the duration of database operations.
var DatabaseOperationDuration = factory.NewHistogramVec(prometheus.HistogramOpts{
	Namespace: namespace,
	Name:      "database_operation_duration_seconds",
	Help:      "Duration of database operations in seconds.",
	Buckets:   []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30},
}, []string{"type", "operation"})

// JobCompletions counts job completions by kind and result.
var JobCompletions = factory.NewCounterVec(prometheus.CounterOpts{
	Namespace: namespace,
	Name:      "job_completions_total",
	Help:      "Total job completions by kind and result.",
}, []string{"kind", "result"})

// JobDuration tracks job execution duration from start to completion.
var JobDuration = factory.NewHistogramVec(prometheus.HistogramOpts{
	Namespace: namespace,
	Name:      "job_duration_seconds",
	Help:      "Duration of jobs from start to completion in seconds.",
	Buckets:   []float64{5, 15, 30, 60, 120, 300, 600, 1200},
}, []string{"kind"})

// BackupCompletions counts backup completions by type.
var BackupCompletions = factory.NewCounterVec(prometheus.CounterOpts{
	Namespace: namespace,
	Name:      "backup_completions_total",
	Help:      "Total backup completions by type.",
}, []string{"namespace", "backup_type"})

// Errors counts controller errors classified by finality.
var Errors = factory.NewCounterVec(prometheus.CounterOpts{
	Namespace: namespace,
	Name:      "errors_total",
	Help:      "Total controller errors classified by controller and finality.",
}, []string{"controller", "final"})

// WebhookDenied counts admission requests denied by validation logic.
var WebhookDenied = factory.NewCounterVec(prometheus.CounterOpts{
	Namespace: namespace,
	Name:      "webhook_denied_total",
	Help:      "Total admission requests denied by webhook validation logic.",
}, []string{"webhook", "reason"})
