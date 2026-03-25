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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("DbMigrationJobController", func() {
	const (
		timeout  = 10 * time.Second
		interval = 200 * time.Millisecond
	)

	Describe("when a new DbMigrationJob is created with valid MigrationJobPodSpec", func() {
		var (
			ns          string
			siteName    string
			serviceName string
			jobName     string
		)

		BeforeEach(func() {
			ns = fmt.Sprintf("dbmig-new-%d", GinkgoParallelProcess())
			siteName = "mig-site"
			serviceName = "mig-svc"
			jobName = "mig-job"

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
					ShortName: "msv",
					DeploymentPodSpec: corev1.PodSpec{
						Containers: []corev1.Container{{
							Name:  "app",
							Image: "busybox:latest",
						}},
					},
					MigrationJobPodSpec: &corev1.PodSpec{
						RestartPolicy: corev1.RestartPolicyNever,
						Containers: []corev1.Container{{
							Name:    "migrate",
							Image:   "busybox:latest",
							Command: []string{"echo", "migrate"},
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
						serviceName: {ImageTag: "v1.0.0", Replicas: 1},
					},
				},
			}
			Expect(k8sClient.Create(ctx, site)).To(Succeed())

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

		It("should initialise and reach Pending or Running", func() {
			job := &jobv1.DbMigrationJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      jobName,
					Namespace: ns,
				},
				Spec: jobv1.DbMigrationJobSpec{
					SiteName:        siteName,
					ServiceName:     serviceName,
					ImageTag:        "v1.0.0",
					DeadlineSeconds: 300,
				},
			}
			Expect(k8sClient.Create(ctx, job)).To(Succeed())

			fetched := &jobv1.DbMigrationJob{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(job), fetched)).To(Succeed())
				g.Expect(fetched.Status.State).To(BeElementOf(jobv1.Pending, jobv1.Running))
			}, timeout, interval).Should(Succeed())

			Expect(fetched.Status.LastMigratedImageTag).To(Equal("v1.0.0"))
			Expect(fetched.Status.DeadlineTimestamp).NotTo(BeNil())
		})
	})

	Describe("when the image tag changes on a completed DbMigrationJob", func() {
		var (
			ns          string
			siteName    string
			serviceName string
			jobName     string
		)

		BeforeEach(func() {
			ns = fmt.Sprintf("dbmig-retag-%d", GinkgoParallelProcess())
			siteName = "retag-site"
			serviceName = "retag-svc"
			jobName = "retag-job"

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
					ShortName: "rtg",
					DeploymentPodSpec: corev1.PodSpec{
						Containers: []corev1.Container{{
							Name:  "app",
							Image: "busybox:latest",
						}},
					},
					MigrationJobPodSpec: &corev1.PodSpec{
						RestartPolicy: corev1.RestartPolicyNever,
						Containers: []corev1.Container{{
							Name:    "migrate",
							Image:   "busybox:latest",
							Command: []string{"echo", "migrate"},
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
						serviceName: {ImageTag: "v1.0.0", Replicas: 1},
					},
				},
			}
			Expect(k8sClient.Create(ctx, site)).To(Succeed())

			site.Status.State = sitev1.StatePending
			site.Status.WorkloadHealth = sitev1.WorkloadHealthIncomplete
			site.Status.Services = map[string]sitev1.StagingSiteServiceStatus{
				serviceName: {DbName: "testdb", Username: "testuser"},
			}
			Expect(k8sClient.Status().Update(ctx, site)).To(Succeed())
		})

		It("should reset to Pending when image tag changes", func() {
			job := &jobv1.DbMigrationJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      jobName,
					Namespace: ns,
				},
				Spec: jobv1.DbMigrationJobSpec{
					SiteName:        siteName,
					ServiceName:     serviceName,
					ImageTag:        "v1.0.0",
					DeadlineSeconds: 300,
				},
			}
			Expect(k8sClient.Create(ctx, job)).To(Succeed())

			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(job), job)).To(Succeed())
				g.Expect(job.Status.LastMigratedImageTag).To(Equal("v1.0.0"))
			}, timeout, interval).Should(Succeed())

			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(job), job)).To(Succeed())
				job.Status.State = jobv1.Complete
				g.Expect(k8sClient.Status().Update(ctx, job)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(job), job)).To(Succeed())
				job.Spec.ImageTag = "v2.0.0"
				g.Expect(k8sClient.Update(ctx, job)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			Eventually(func(g Gomega) {
				fetched := &jobv1.DbMigrationJob{}
				g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(job), fetched)).To(Succeed())
				g.Expect(fetched.Status.LastMigratedImageTag).To(Equal("v2.0.0"))
				g.Expect(fetched.Status.State).To(BeElementOf(jobv1.Pending, jobv1.Running))
			}, timeout, interval).Should(Succeed())
		})
	})

	Describe("when the deadline is exceeded", func() {
		var (
			ns          string
			siteName    string
			serviceName string
			jobName     string
		)

		BeforeEach(func() {
			ns = fmt.Sprintf("dbmig-deadline-%d", GinkgoParallelProcess())
			siteName = "deadline-mig-site"
			serviceName = "deadline-mig-svc"
			jobName = "deadline-mig-job"

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
					ShortName: "dmv",
					DeploymentPodSpec: corev1.PodSpec{
						Containers: []corev1.Container{{
							Name:  "app",
							Image: "busybox:latest",
						}},
					},
					MigrationJobPodSpec: &corev1.PodSpec{
						RestartPolicy: corev1.RestartPolicyNever,
						Containers: []corev1.Container{{
							Name:    "migrate",
							Image:   "busybox:latest",
							Command: []string{"echo", "migrate"},
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
						serviceName: {ImageTag: "v1.0.0", Replicas: 1},
					},
				},
			}
			Expect(k8sClient.Create(ctx, site)).To(Succeed())

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

		It("should transition to Failed after the deadline passes", func() {
			failuresBefore := metricstest.GetCounterValue(appmetrics.JobCompletions, "dbmigration", "failure")
			job := &jobv1.DbMigrationJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      jobName,
					Namespace: ns,
				},
				Spec: jobv1.DbMigrationJobSpec{
					SiteName:        siteName,
					ServiceName:     serviceName,
					ImageTag:        "v1.0.0",
					DeadlineSeconds: 1,
				},
			}
			Expect(k8sClient.Create(ctx, job)).To(Succeed())

			fetched := &jobv1.DbMigrationJob{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(job), fetched)).To(Succeed())
				g.Expect(fetched.Status.State).To(Equal(jobv1.Running))
			}, timeout, interval).Should(Succeed())

			// Set the deadline timestamp to the past so the next reconciliation sees it as expired
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(job), fetched)).To(Succeed())
				fetched.Status.DeadlineTimestamp = &metav1.Time{Time: time.Now().Add(-1 * time.Second)}
				g.Expect(k8sClient.Status().Update(ctx, fetched)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			// envtest won't re-enqueue after the deadline passes, so write an annotation to force reconciliation
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(job), fetched)).To(Succeed())
				if fetched.Annotations == nil {
					fetched.Annotations = map[string]string{}
				}
				fetched.Annotations["test-trigger"] = "deadline"
				g.Expect(k8sClient.Update(ctx, fetched)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(job), fetched)).To(Succeed())
				g.Expect(fetched.Status.State).To(Equal(jobv1.Failed))
			}, timeout, interval).Should(Succeed())

			failuresAfter := metricstest.GetCounterValue(appmetrics.JobCompletions, "dbmigration", "failure")
			Expect(failuresAfter-failuresBefore).To(BeNumerically(">=", 1), "expected job_completions_total(dbmigration, failure) to be incremented")
		})
	})

	Describe("when ServiceConfig has no MigrationJobPodSpec", func() {
		var (
			ns          string
			siteName    string
			serviceName string
			jobName     string
		)

		BeforeEach(func() {
			ns = fmt.Sprintf("dbmig-nopod-%d", GinkgoParallelProcess())
			siteName = "nopod-mig-site"
			serviceName = "nopod-mig-svc"
			jobName = "nopod-mig-job"

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
					ShortName: "npm",
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
						serviceName: {ImageTag: "v1.0.0", Replicas: 1},
					},
				},
			}
			Expect(k8sClient.Create(ctx, site)).To(Succeed())

			site.Status.State = sitev1.StatePending
			site.Status.WorkloadHealth = sitev1.WorkloadHealthIncomplete
			site.Status.Services = map[string]sitev1.StagingSiteServiceStatus{
				serviceName: {DbName: "testdb", Username: "testuser"},
			}
			Expect(k8sClient.Status().Update(ctx, site)).To(Succeed())
		})

		It("should remain in Pending state because no migration pod spec is available", func() {
			job := &jobv1.DbMigrationJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      jobName,
					Namespace: ns,
				},
				Spec: jobv1.DbMigrationJobSpec{
					SiteName:        siteName,
					ServiceName:     serviceName,
					ImageTag:        "v1.0.0",
					DeadlineSeconds: 300,
				},
			}
			Expect(k8sClient.Create(ctx, job)).To(Succeed())

			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(job), job)).To(Succeed())
				g.Expect(job.Status.LastMigratedImageTag).To(Equal("v1.0.0"))
			}, timeout, interval).Should(Succeed())

			// It should stay in Pending since it can't create a batch job without MigrationJobPodSpec
			Consistently(func(g Gomega) {
				fetched := &jobv1.DbMigrationJob{}
				g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(job), fetched)).To(Succeed())
				g.Expect(fetched.Status.State).NotTo(Equal(jobv1.Complete))
			}, 2*time.Second, interval).Should(Succeed())
		})
	})

	Describe("when the DbMigrationJob is already in a final state with matching image tag", func() {
		var (
			ns      string
			jobName string
		)

		BeforeEach(func() {
			ns = fmt.Sprintf("dbmig-final-%d", GinkgoParallelProcess())
			jobName = "mig-done"

			nsObj := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: ns},
			}
			Expect(k8sClient.Create(ctx, nsObj)).To(Succeed())

			serviceConfig := &configv1.ServiceConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "done-svc",
					Namespace: ns,
				},
				Spec: configv1.ServiceConfigSpec{
					ShortName: "dsv",
					DeploymentPodSpec: corev1.PodSpec{
						Containers: []corev1.Container{{
							Name:  "app",
							Image: "busybox:latest",
						}},
					},
					MigrationJobPodSpec: &corev1.PodSpec{
						RestartPolicy: corev1.RestartPolicyNever,
						Containers: []corev1.Container{{
							Name:    "migrate",
							Image:   "busybox:latest",
							Command: []string{"echo", "migrate"},
						}},
					},
				},
			}
			Expect(k8sClient.Create(ctx, serviceConfig)).To(Succeed())

			site := &sitev1.StagingSite{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "done-site",
					Namespace: ns,
				},
				Spec: sitev1.StagingSiteSpec{
					Enabled: true,
					Services: map[string]sitev1.StagingSiteService{
						"done-svc": {ImageTag: "v2.0.0", Replicas: 1},
					},
				},
			}
			Expect(k8sClient.Create(ctx, site)).To(Succeed())

			site.Status.State = sitev1.StatePending
			site.Status.WorkloadHealth = sitev1.WorkloadHealthIncomplete
			site.Status.Services = map[string]sitev1.StagingSiteServiceStatus{
				"done-svc": {
					DbName:   "testdb",
					Username: "testuser",
				},
			}
			Expect(k8sClient.Status().Update(ctx, site)).To(Succeed())
		})

		It("should stay Complete", func() {
			job := &jobv1.DbMigrationJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      jobName,
					Namespace: ns,
				},
				Spec: jobv1.DbMigrationJobSpec{
					SiteName:        "done-site",
					ServiceName:     "done-svc",
					ImageTag:        "v2.0.0",
					DeadlineSeconds: 300,
				},
			}
			Expect(k8sClient.Create(ctx, job)).To(Succeed())

			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(job), job)).To(Succeed())
				g.Expect(job.Status.LastMigratedImageTag).To(Equal("v2.0.0"))
			}, timeout, interval).Should(Succeed())

			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(job), job)).To(Succeed())
				job.Status.State = jobv1.Complete
				g.Expect(k8sClient.Status().Update(ctx, job)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			Consistently(func(g Gomega) {
				fetched := &jobv1.DbMigrationJob{}
				g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(job), fetched)).To(Succeed())
				g.Expect(fetched.Status.State).To(Equal(jobv1.Complete))
				g.Expect(fetched.Status.LastMigratedImageTag).To(Equal("v2.0.0"))
			}, 1*time.Second, interval).Should(Succeed())
		})
	})
})
