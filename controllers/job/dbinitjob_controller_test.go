package job

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	configv1 "github.com/szeber/kube-stager/apis/config/v1"
	jobv1 "github.com/szeber/kube-stager/apis/job/v1"
	sitev1 "github.com/szeber/kube-stager/apis/site/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("DbInitJobController", func() {
	const (
		timeout  = 10 * time.Second
		interval = 200 * time.Millisecond
	)

	Describe("when a new DbInitJob is created with valid mysql config and DbInitPodSpec", func() {
		var (
			ns          string
			siteName    string
			serviceName string
			mysqlName   string
			jobName     string
		)

		BeforeEach(func() {
			ns = fmt.Sprintf("dbinit-new-%d", GinkgoParallelProcess())
			siteName = "init-site"
			serviceName = "init-svc"
			mysqlName = "init-mysql"
			jobName = "init-job"

			nsObj := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: ns},
			}
			Expect(k8sClient.Create(ctx, nsObj)).To(Succeed())

			mysqlConfig := &configv1.MysqlConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      mysqlName,
					Namespace: ns,
				},
				Spec: configv1.MysqlConfigSpec{
					Host:     "mysql.local",
					Username: "root",
					Password: "rootpass",
					Port:     3306,
				},
			}
			Expect(k8sClient.Create(ctx, mysqlConfig)).To(Succeed())

			serviceConfig := &configv1.ServiceConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      serviceName,
					Namespace: ns,
				},
				Spec: configv1.ServiceConfigSpec{
					ShortName: "isv",
					DeploymentPodSpec: corev1.PodSpec{
						Containers: []corev1.Container{{
							Name:  "app",
							Image: "busybox:latest",
						}},
					},
					DbInitPodSpec: &corev1.PodSpec{
						RestartPolicy: corev1.RestartPolicyNever,
						Containers: []corev1.Container{{
							Name:    "dbinit",
							Image:   "busybox:latest",
							Command: []string{"echo", "init"},
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

		It("should initialise the status and eventually reach Pending or Running", func() {
			job := &jobv1.DbInitJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      jobName,
					Namespace: ns,
				},
				Spec: jobv1.DbInitJobSpec{
					SiteName:         siteName,
					ServiceName:      serviceName,
					MysqlEnvironment: mysqlName,
					DbInitSource:     "master",
					DatabaseName:     "testdb",
					Username:         "testuser",
					Password:         "testpass",
					DeadlineSeconds:  300,
				},
			}
			Expect(k8sClient.Create(ctx, job)).To(Succeed())

			fetched := &jobv1.DbInitJob{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(job), fetched)).To(Succeed())
				g.Expect(fetched.Status.State).To(BeElementOf(jobv1.Pending, jobv1.Running))
			}, timeout, interval).Should(Succeed())

			Expect(fetched.Status.DeadlineTimestamp).NotTo(BeNil())
		})
	})

	Describe("when neither mysql nor mongo environment is specified", func() {
		var (
			ns          string
			siteName    string
			serviceName string
			jobName     string
		)

		BeforeEach(func() {
			ns = fmt.Sprintf("dbinit-nodb-%d", GinkgoParallelProcess())
			siteName = "nodb-site"
			serviceName = "nodb-svc"
			jobName = "nodb-job"

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
					ShortName: "ndb",
					DeploymentPodSpec: corev1.PodSpec{
						Containers: []corev1.Container{{
							Name:  "app",
							Image: "busybox:latest",
						}},
					},
					DbInitPodSpec: &corev1.PodSpec{
						RestartPolicy: corev1.RestartPolicyNever,
						Containers: []corev1.Container{{
							Name:    "dbinit",
							Image:   "busybox:latest",
							Command: []string{"echo", "init"},
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
				},
			}
			Expect(k8sClient.Create(ctx, site)).To(Succeed())
		})

		It("should fail because no database config is available", func() {
			job := &jobv1.DbInitJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      jobName,
					Namespace: ns,
				},
				Spec: jobv1.DbInitJobSpec{
					SiteName:        siteName,
					ServiceName:     serviceName,
					DbInitSource:    "master",
					DatabaseName:    "testdb",
					Username:        "testuser",
					Password:        "testpass",
					DeadlineSeconds: 300,
				},
			}
			Expect(k8sClient.Create(ctx, job)).To(Succeed())

			fetched := &jobv1.DbInitJob{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(job), fetched)).To(Succeed())
				g.Expect(fetched.Status.State).To(Equal(jobv1.Failed))
			}, timeout, interval).Should(Succeed())
		})
	})

	Describe("when a new DbInitJob is created with valid mongo config and DbInitPodSpec", func() {
		var (
			ns          string
			siteName    string
			serviceName string
			mongoName   string
			jobName     string
		)

		BeforeEach(func() {
			ns = fmt.Sprintf("dbinit-mongo-%d", GinkgoParallelProcess())
			siteName = "mongo-init-site"
			serviceName = "mongo-init-svc"
			mongoName = "mongo-env"
			jobName = "mongo-init-job"

			nsObj := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: ns},
			}
			Expect(k8sClient.Create(ctx, nsObj)).To(Succeed())

			mongoConfig := &configv1.MongoConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      mongoName,
					Namespace: ns,
				},
				Spec: configv1.MongoConfigSpec{
					Host1:    "mongo.local",
					Username: "admin",
					Password: "adminpass",
					Port:     27017,
				},
			}
			Expect(k8sClient.Create(ctx, mongoConfig)).To(Succeed())

			serviceConfig := &configv1.ServiceConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      serviceName,
					Namespace: ns,
				},
				Spec: configv1.ServiceConfigSpec{
					ShortName: "miv",
					DeploymentPodSpec: corev1.PodSpec{
						Containers: []corev1.Container{{
							Name:  "app",
							Image: "busybox:latest",
						}},
					},
					DbInitPodSpec: &corev1.PodSpec{
						RestartPolicy: corev1.RestartPolicyNever,
						Containers: []corev1.Container{{
							Name:    "dbinit",
							Image:   "busybox:latest",
							Command: []string{"echo", "init-mongo"},
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

		It("should initialise the status and reach Pending or Running", func() {
			job := &jobv1.DbInitJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      jobName,
					Namespace: ns,
				},
				Spec: jobv1.DbInitJobSpec{
					SiteName:         siteName,
					ServiceName:      serviceName,
					MongoEnvironment: mongoName,
					DbInitSource:     "master",
					DatabaseName:     "testdb",
					Username:         "testuser",
					Password:         "testpass",
					DeadlineSeconds:  300,
				},
			}
			Expect(k8sClient.Create(ctx, job)).To(Succeed())

			fetched := &jobv1.DbInitJob{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(job), fetched)).To(Succeed())
				g.Expect(fetched.Status.State).To(BeElementOf(jobv1.Pending, jobv1.Running))
			}, timeout, interval).Should(Succeed())

			Expect(fetched.Status.DeadlineTimestamp).NotTo(BeNil())
		})
	})

	Describe("when the deadline is exceeded", func() {
		var (
			ns          string
			siteName    string
			serviceName string
			mysqlName   string
			jobName     string
		)

		BeforeEach(func() {
			ns = fmt.Sprintf("dbinit-deadline-%d", GinkgoParallelProcess())
			siteName = "deadline-site"
			serviceName = "deadline-svc"
			mysqlName = "deadline-mysql"
			jobName = "deadline-job"

			nsObj := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: ns},
			}
			Expect(k8sClient.Create(ctx, nsObj)).To(Succeed())

			mysqlConfig := &configv1.MysqlConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      mysqlName,
					Namespace: ns,
				},
				Spec: configv1.MysqlConfigSpec{
					Host:     "mysql.local",
					Username: "root",
					Password: "rootpass",
					Port:     3306,
				},
			}
			Expect(k8sClient.Create(ctx, mysqlConfig)).To(Succeed())

			serviceConfig := &configv1.ServiceConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      serviceName,
					Namespace: ns,
				},
				Spec: configv1.ServiceConfigSpec{
					ShortName: "dlv",
					DeploymentPodSpec: corev1.PodSpec{
						Containers: []corev1.Container{{
							Name:  "app",
							Image: "busybox:latest",
						}},
					},
					DbInitPodSpec: &corev1.PodSpec{
						RestartPolicy: corev1.RestartPolicyNever,
						Containers: []corev1.Container{{
							Name:    "dbinit",
							Image:   "busybox:latest",
							Command: []string{"echo", "init"},
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
			job := &jobv1.DbInitJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      jobName,
					Namespace: ns,
				},
				Spec: jobv1.DbInitJobSpec{
					SiteName:         siteName,
					ServiceName:      serviceName,
					MysqlEnvironment: mysqlName,
					DbInitSource:     "master",
					DatabaseName:     "testdb",
					Username:         "testuser",
					Password:         "testpass",
					DeadlineSeconds:  1,
				},
			}
			Expect(k8sClient.Create(ctx, job)).To(Succeed())

			fetched := &jobv1.DbInitJob{}
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
		})
	})

	Describe("when ServiceConfig has no DbInitPodSpec", func() {
		var (
			ns          string
			siteName    string
			serviceName string
			mysqlName   string
			jobName     string
		)

		BeforeEach(func() {
			ns = fmt.Sprintf("dbinit-nopod-%d", GinkgoParallelProcess())
			siteName = "nopod-site"
			serviceName = "nopod-svc"
			mysqlName = "nopod-mysql"
			jobName = "nopod-job"

			nsObj := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: ns},
			}
			Expect(k8sClient.Create(ctx, nsObj)).To(Succeed())

			mysqlConfig := &configv1.MysqlConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      mysqlName,
					Namespace: ns,
				},
				Spec: configv1.MysqlConfigSpec{
					Host:     "mysql.local",
					Username: "root",
					Password: "rootpass",
					Port:     3306,
				},
			}
			Expect(k8sClient.Create(ctx, mysqlConfig)).To(Succeed())

			serviceConfig := &configv1.ServiceConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      serviceName,
					Namespace: ns,
				},
				Spec: configv1.ServiceConfigSpec{
					ShortName: "npd",
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
				},
			}
			Expect(k8sClient.Create(ctx, site)).To(Succeed())
		})

		It("should fail because no DbInitPodSpec is set", func() {
			job := &jobv1.DbInitJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      jobName,
					Namespace: ns,
				},
				Spec: jobv1.DbInitJobSpec{
					SiteName:         siteName,
					ServiceName:      serviceName,
					MysqlEnvironment: mysqlName,
					DbInitSource:     "master",
					DatabaseName:     "testdb",
					Username:         "testuser",
					Password:         "testpass",
					DeadlineSeconds:  300,
				},
			}
			Expect(k8sClient.Create(ctx, job)).To(Succeed())

			fetched := &jobv1.DbInitJob{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(job), fetched)).To(Succeed())
				g.Expect(fetched.Status.State).To(Equal(jobv1.Failed))
			}, timeout, interval).Should(Succeed())
		})
	})
})
