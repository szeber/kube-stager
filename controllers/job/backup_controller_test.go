package job

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	configv1 "github.com/szeber/kube-stager/apis/config/v1"
	jobv1 "github.com/szeber/kube-stager/apis/job/v1"
	sitev1 "github.com/szeber/kube-stager/apis/site/v1"
	appmetrics "github.com/szeber/kube-stager/internal/metrics"
	"github.com/szeber/kube-stager/internal/metricstest"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("BackupController", func() {
	const (
		timeout  = 10 * time.Second
		interval = 200 * time.Millisecond
	)

	Describe("when a new Backup is created with services that have BackupPodSpec", func() {
		var (
			ns          string
			siteName    string
			serviceName string
			backupName  string
		)

		BeforeEach(func() {
			ns = fmt.Sprintf("backup-new-%d", GinkgoParallelProcess())
			siteName = "test-site"
			serviceName = "test-svc"
			backupName = "test-backup"

			nsObj := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: ns},
			}
			Expect(k8sClient.Create(ctx, nsObj)).To(Succeed())

			serviceConfig := &configv1.ServiceConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      serviceName,
					Namespace: ns,
				},
				Spec: configv1.ServiceConfigSpec{
					ShortName: "tsv",
					DeploymentPodSpec: corev1.PodSpec{
						Containers: []corev1.Container{{
							Name:  "app",
							Image: "busybox:latest",
						}},
					},
					BackupPodSpec: &corev1.PodSpec{
						RestartPolicy: corev1.RestartPolicyNever,
						Containers: []corev1.Container{{
							Name:    "backup",
							Image:   "busybox:latest",
							Command: []string{"echo", "backup"},
						}},
					},
				},
			}
			Expect(k8sClient.Create(ctx, serviceConfig)).To(Succeed())

			site := &sitev1.StagingSite{
				ObjectMeta: metav1.ObjectMeta{
					Name:      siteName,
					Namespace: ns,
				},
				Spec: sitev1.StagingSiteSpec{
					Enabled: true,
					Services: map[string]sitev1.StagingSiteService{
						serviceName: {ImageTag: "latest", Replicas: 1},
					},
				},
			}
			Expect(k8sClient.Create(ctx, site)).To(Succeed())

			// Status is a subresource, so update it separately
			site.Status.State = sitev1.StatePending
			site.Status.WorkloadHealth = sitev1.WorkloadHealthIncomplete
			site.Status.Services = map[string]sitev1.StagingSiteServiceStatus{
				serviceName: {
					DbName:   "testdb",
					Username: "testuser",
				},
			}
			Expect(k8sClient.Status().Update(ctx, site)).To(Succeed())
		})

		It("should initialise the backup status", func() {
			backup := &jobv1.Backup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      backupName,
					Namespace: ns,
				},
				Spec: jobv1.BackupSpec{
					SiteName:   siteName,
					BackupType: jobv1.BackupTypeManual,
				},
			}
			Expect(k8sClient.Create(ctx, backup)).To(Succeed())

			fetched := &jobv1.Backup{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(backup), fetched)).To(Succeed())
				g.Expect(fetched.Status.State).NotTo(BeEmpty())
			}, timeout, interval).Should(Succeed())

			Expect(fetched.Status.State).To(BeElementOf(jobv1.Pending, jobv1.Running, jobv1.Complete))
		})
	})

	Describe("when no services have BackupPodSpec", func() {
		var (
			ns         string
			siteName   string
			backupName string
		)

		BeforeEach(func() {
			ns = fmt.Sprintf("backup-nosvc-%d", GinkgoParallelProcess())
			siteName = "site-nobackup"
			backupName = "backup-nobackup"

			nsObj := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: ns},
			}
			Expect(k8sClient.Create(ctx, nsObj)).To(Succeed())

			serviceConfig := &configv1.ServiceConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "svc-nobackup",
					Namespace: ns,
				},
				Spec: configv1.ServiceConfigSpec{
					ShortName: "snb",
					DeploymentPodSpec: corev1.PodSpec{
						Containers: []corev1.Container{{
							Name:  "app",
							Image: "busybox:latest",
						}},
					},
				},
			}
			Expect(k8sClient.Create(ctx, serviceConfig)).To(Succeed())

			site := &sitev1.StagingSite{
				ObjectMeta: metav1.ObjectMeta{
					Name:      siteName,
					Namespace: ns,
				},
				Spec: sitev1.StagingSiteSpec{
					Enabled: true,
					Services: map[string]sitev1.StagingSiteService{
						"svc-nobackup": {ImageTag: "latest", Replicas: 1},
					},
				},
			}
			Expect(k8sClient.Create(ctx, site)).To(Succeed())

			site.Status.State = sitev1.StatePending
			site.Status.WorkloadHealth = sitev1.WorkloadHealthIncomplete
			site.Status.Services = map[string]sitev1.StagingSiteServiceStatus{
				"svc-nobackup": {
					DbName:   "testdb",
					Username: "testuser",
				},
			}
			Expect(k8sClient.Status().Update(ctx, site)).To(Succeed())
		})

		It("should transition to Complete", func() {
			completionsBefore := metricstest.GetCounterValue(appmetrics.JobCompletions, "backup", "success")
			backupCompletionsBefore := metricstest.GetCounterValue(appmetrics.BackupCompletions, ns, string(jobv1.BackupTypeManual))

			backup := &jobv1.Backup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      backupName,
					Namespace: ns,
				},
				Spec: jobv1.BackupSpec{
					SiteName:   siteName,
					BackupType: jobv1.BackupTypeManual,
				},
			}
			Expect(k8sClient.Create(ctx, backup)).To(Succeed())

			fetched := &jobv1.Backup{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(backup), fetched)).To(Succeed())
				g.Expect(fetched.Status.State).To(Equal(jobv1.Complete))
			}, timeout, interval).Should(Succeed())

			completionsAfter := metricstest.GetCounterValue(appmetrics.JobCompletions, "backup", "success")
			Expect(completionsAfter-completionsBefore).To(BeNumerically(">=", 1), "expected job_completions_total(backup, success) to be incremented")

			backupCompletionsAfter := metricstest.GetCounterValue(appmetrics.BackupCompletions, ns, string(jobv1.BackupTypeManual))
			Expect(backupCompletionsAfter-backupCompletionsBefore).To(BeNumerically(">=", 1), "expected backup_completions_total to be incremented")
		})
	})

	Describe("when multiple services have BackupPodSpec", func() {
		var (
			ns       string
			siteName string
		)

		BeforeEach(func() {
			ns = fmt.Sprintf("backup-multi-%d", GinkgoParallelProcess())
			siteName = "multi-site"

			nsObj := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: ns},
			}
			Expect(k8sClient.Create(ctx, nsObj)).To(Succeed())

			for _, svcName := range []string{"svc-a", "svc-b"} {
				serviceConfig := &configv1.ServiceConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name:      svcName,
						Namespace: ns,
					},
					Spec: configv1.ServiceConfigSpec{
						ShortName: svcName[:3],
						DeploymentPodSpec: corev1.PodSpec{
							Containers: []corev1.Container{{
								Name:  "app",
								Image: "busybox:latest",
							}},
						},
						BackupPodSpec: &corev1.PodSpec{
							RestartPolicy: corev1.RestartPolicyNever,
							Containers: []corev1.Container{{
								Name:    "backup",
								Image:   "busybox:latest",
								Command: []string{"echo", "backup"},
							}},
						},
					},
				}
				Expect(k8sClient.Create(ctx, serviceConfig)).To(Succeed())
			}

			site := &sitev1.StagingSite{
				ObjectMeta: metav1.ObjectMeta{
					Name:      siteName,
					Namespace: ns,
				},
				Spec: sitev1.StagingSiteSpec{
					Enabled: true,
					Services: map[string]sitev1.StagingSiteService{
						"svc-a": {ImageTag: "latest", Replicas: 1},
						"svc-b": {ImageTag: "latest", Replicas: 1},
					},
				},
			}
			Expect(k8sClient.Create(ctx, site)).To(Succeed())

			site.Status.State = sitev1.StatePending
			site.Status.WorkloadHealth = sitev1.WorkloadHealthIncomplete
			site.Status.Services = map[string]sitev1.StagingSiteServiceStatus{
				"svc-a": {DbName: "testdb_a", Username: "usera"},
				"svc-b": {DbName: "testdb_b", Username: "userb"},
			}
			Expect(k8sClient.Status().Update(ctx, site)).To(Succeed())
		})

		It("should create service-level backup statuses for both services", func() {
			backup := &jobv1.Backup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "backup-multi",
					Namespace: ns,
				},
				Spec: jobv1.BackupSpec{
					SiteName:   siteName,
					BackupType: jobv1.BackupTypeManual,
				},
			}
			Expect(k8sClient.Create(ctx, backup)).To(Succeed())

			fetched := &jobv1.Backup{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(backup), fetched)).To(Succeed())
				g.Expect(fetched.Status.Services).To(HaveKey("svc-a"))
				g.Expect(fetched.Status.Services).To(HaveKey("svc-b"))
			}, timeout, interval).Should(Succeed())
		})
	})

	Describe("when BackupType is Final", func() {
		var (
			ns       string
			siteName string
		)

		BeforeEach(func() {
			ns = fmt.Sprintf("backup-final-type-%d", GinkgoParallelProcess())
			siteName = "final-site"

			nsObj := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: ns},
			}
			Expect(k8sClient.Create(ctx, nsObj)).To(Succeed())

			serviceConfig := &configv1.ServiceConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "final-svc",
					Namespace: ns,
				},
				Spec: configv1.ServiceConfigSpec{
					ShortName: "fin",
					DeploymentPodSpec: corev1.PodSpec{
						Containers: []corev1.Container{{
							Name:  "app",
							Image: "busybox:latest",
						}},
					},
					BackupPodSpec: &corev1.PodSpec{
						RestartPolicy: corev1.RestartPolicyNever,
						Containers: []corev1.Container{{
							Name:    "backup",
							Image:   "busybox:latest",
							Command: []string{"echo", "final-backup"},
						}},
					},
				},
			}
			Expect(k8sClient.Create(ctx, serviceConfig)).To(Succeed())

			site := &sitev1.StagingSite{
				ObjectMeta: metav1.ObjectMeta{
					Name:      siteName,
					Namespace: ns,
				},
				Spec: sitev1.StagingSiteSpec{
					Enabled: true,
					Services: map[string]sitev1.StagingSiteService{
						"final-svc": {ImageTag: "latest", Replicas: 1},
					},
				},
			}
			Expect(k8sClient.Create(ctx, site)).To(Succeed())

			site.Status.State = sitev1.StatePending
			site.Status.WorkloadHealth = sitev1.WorkloadHealthIncomplete
			site.Status.Services = map[string]sitev1.StagingSiteServiceStatus{
				"final-svc": {DbName: "testdb", Username: "testuser"},
			}
			Expect(k8sClient.Status().Update(ctx, site)).To(Succeed())
		})

		It("should initialise the backup with Final type and set status", func() {
			backup := &jobv1.Backup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "backup-final-type",
					Namespace: ns,
				},
				Spec: jobv1.BackupSpec{
					SiteName:   siteName,
					BackupType: jobv1.BackupTypeFinal,
				},
			}
			Expect(k8sClient.Create(ctx, backup)).To(Succeed())

			fetched := &jobv1.Backup{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(backup), fetched)).To(Succeed())
				g.Expect(fetched.Status.State).NotTo(BeEmpty())
				g.Expect(fetched.Spec.BackupType).To(Equal(jobv1.BackupTypeFinal))
			}, timeout, interval).Should(Succeed())
		})
	})

	Describe("when the underlying batch Job fails", func() {
		var (
			ns          string
			siteName    string
			serviceName string
			backupName  string
		)

		BeforeEach(func() {
			ns = fmt.Sprintf("backup-fail-%d", GinkgoParallelProcess())
			siteName = "test-site-fail"
			serviceName = "test-svc-fail"
			backupName = "test-backup-fail"

			nsObj := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: ns},
			}
			Expect(k8sClient.Create(ctx, nsObj)).To(Succeed())

			serviceConfig := &configv1.ServiceConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      serviceName,
					Namespace: ns,
				},
				Spec: configv1.ServiceConfigSpec{
					ShortName: "tsf",
					DeploymentPodSpec: corev1.PodSpec{
						Containers: []corev1.Container{{
							Name:  "app",
							Image: "busybox:latest",
						}},
					},
					BackupPodSpec: &corev1.PodSpec{
						RestartPolicy: corev1.RestartPolicyNever,
						Containers: []corev1.Container{{
							Name:    "backup",
							Image:   "busybox:latest",
							Command: []string{"false"},
						}},
					},
				},
			}
			Expect(k8sClient.Create(ctx, serviceConfig)).To(Succeed())

			site := &sitev1.StagingSite{
				ObjectMeta: metav1.ObjectMeta{
					Name:      siteName,
					Namespace: ns,
				},
				Spec: sitev1.StagingSiteSpec{
					Enabled: true,
					Services: map[string]sitev1.StagingSiteService{
						serviceName: {ImageTag: "latest", Replicas: 1},
					},
				},
			}
			Expect(k8sClient.Create(ctx, site)).To(Succeed())

			// Status is a subresource, so update it separately
			site.Status.State = sitev1.StatePending
			site.Status.WorkloadHealth = sitev1.WorkloadHealthIncomplete
			site.Status.Services = map[string]sitev1.StagingSiteServiceStatus{
				serviceName: {
					DbName:   "testdb",
					Username: "testuser",
				},
			}
			Expect(k8sClient.Status().Update(ctx, site)).To(Succeed())
		})

		It("should transition to Failed when the batch Job reports a Failed condition", func() {
			failuresBefore := metricstest.GetCounterValue(appmetrics.JobCompletions, "backup", "failure")
			backup := &jobv1.Backup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      backupName,
					Namespace: ns,
				},
				Spec: jobv1.BackupSpec{
					SiteName:   siteName,
					BackupType: jobv1.BackupTypeManual,
				},
			}
			Expect(k8sClient.Create(ctx, backup)).To(Succeed())

			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(backup), backup)).To(Succeed())
				g.Expect(backup.Status.State).To(Equal(jobv1.Running))
			}, timeout, interval).Should(Succeed())

			var batchJobs batchv1.JobList
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.List(ctx, &batchJobs, client.InNamespace(ns))).To(Succeed())
				g.Expect(batchJobs.Items).NotTo(BeEmpty())
			}, timeout, interval).Should(Succeed())

			// Simulate batch Job failure by setting the Failed condition.
			// K8s 1.35+ requires startTime on finished jobs and FailureTarget before Failed.
			batchJob := &batchJobs.Items[0]
			now := metav1.Now()
			batchJob.Status.StartTime = &now
			batchJob.Status.Conditions = append(batchJob.Status.Conditions,
				batchv1.JobCondition{
					Type:               batchv1.JobFailureTarget,
					Status:             corev1.ConditionTrue,
					LastTransitionTime: now,
					Reason:             "BackoffLimitExceeded",
					Message:            "Job has reached the specified backoff limit",
				},
				batchv1.JobCondition{
					Type:               batchv1.JobFailed,
					Status:             corev1.ConditionTrue,
					LastTransitionTime: now,
					Reason:             "BackoffLimitExceeded",
					Message:            "Job has reached the specified backoff limit",
				},
			)
			Expect(k8sClient.Status().Update(ctx, batchJob)).To(Succeed())

			// Force re-reconciliation by touching the backup
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(backup), backup)).To(Succeed())
				if backup.Annotations == nil {
					backup.Annotations = map[string]string{}
				}
				backup.Annotations["test-trigger"] = "fail"
				g.Expect(k8sClient.Update(ctx, backup)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(backup), backup)).To(Succeed())
				g.Expect(backup.Status.State).To(Equal(jobv1.Failed))
			}, timeout, interval).Should(Succeed())

			failuresAfter := metricstest.GetCounterValue(appmetrics.JobCompletions, "backup", "failure")
			Expect(failuresAfter-failuresBefore).To(BeNumerically(">=", 1), "expected job_completions_total(backup, failure) to be incremented")
		})
	})

	Describe("when a Backup is already in a final state", func() {
		var (
			ns         string
			backupName string
		)

		BeforeEach(func() {
			ns = fmt.Sprintf("backup-final-%d", GinkgoParallelProcess())
			backupName = "backup-done"

			nsObj := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: ns},
			}
			Expect(k8sClient.Create(ctx, nsObj)).To(Succeed())

			site := &sitev1.StagingSite{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "site-done",
					Namespace: ns,
				},
				Spec: sitev1.StagingSiteSpec{
					Enabled: true,
				},
			}
			Expect(k8sClient.Create(ctx, site)).To(Succeed())
		})

		It("should stay Complete", func() {
			backup := &jobv1.Backup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      backupName,
					Namespace: ns,
				},
				Spec: jobv1.BackupSpec{
					SiteName:   "site-done",
					BackupType: jobv1.BackupTypeManual,
				},
			}
			Expect(k8sClient.Create(ctx, backup)).To(Succeed())

			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(backup), backup)).To(Succeed())
				g.Expect(backup.Status.State).NotTo(BeEmpty())
			}, timeout, interval).Should(Succeed())

			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(backup), backup)).To(Succeed())
				now := metav1.Now()
				backup.Status.State = jobv1.Complete
				backup.Status.JobStartedAt = &now
				backup.Status.JobFinishedAt = &now
				g.Expect(k8sClient.Status().Update(ctx, backup)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			Consistently(func(g Gomega) {
				fetched := &jobv1.Backup{}
				g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(backup), fetched)).To(Succeed())
				g.Expect(fetched.Status.State).To(Equal(jobv1.Complete))
			}, 1*time.Second, interval).Should(Succeed())
		})
	})
})
