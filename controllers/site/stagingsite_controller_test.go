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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	configv1 "github.com/szeber/kube-stager/apis/config/v1"
	jobv1 "github.com/szeber/kube-stager/apis/job/v1"
	sitev1 "github.com/szeber/kube-stager/apis/site/v1"
	taskv1 "github.com/szeber/kube-stager/apis/task/v1"
	"github.com/szeber/kube-stager/helpers"
	"github.com/szeber/kube-stager/helpers/annotations"
	"github.com/szeber/kube-stager/internal/testutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func createNamespace() string {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-site-",
		},
	}
	Expect(k8sClient.Create(ctx, ns)).To(Succeed())
	return ns.Name
}

var _ = Describe("StagingSite controller", func() {

	const (
		timeout  = 10 * time.Second
		interval = 200 * time.Millisecond
	)

	Describe("New StagingSite gets finalizer and status initialized", func() {
		var (
			ns       string
			siteName string
		)

		BeforeEach(func() {
			ns = createNamespace()
			siteName = "site-finalizer"
			testClock.SetNow(time.Now())

			sc := testutil.NewTestServiceConfig("web", ns, "web")
			Expect(k8sClient.Create(ctx, sc)).To(Succeed())

			site := testutil.NewTestStagingSite(siteName, ns, map[string]sitev1.StagingSiteService{
				"web": {
					ImageTag: "latest",
					Replicas: 1,
				},
			})
			site.Annotations = map[string]string{}
			Expect(k8sClient.Create(ctx, site)).To(Succeed())
		})

		It("should add the finalizer, set state, enable the site, and set the annotation", func() {
			fetched := &sitev1.StagingSite{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: siteName, Namespace: ns}, fetched)).To(Succeed())
				g.Expect(fetched.Finalizers).To(ContainElement(helpers.SiteFinalizerName))
			}, timeout, interval).Should(Succeed())

			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: siteName, Namespace: ns}, fetched)).To(Succeed())
				g.Expect(fetched.Status.State).NotTo(BeEmpty())
				g.Expect(fetched.Status.Enabled).To(BeTrue())
				g.Expect(fetched.Annotations).To(HaveKey(annotations.StagingSiteLastSpecChangeAt))
			}, timeout, interval).Should(Succeed())
		})
	})

	Describe("DisableAt and DeleteAt calculated from spec", func() {
		var (
			ns       string
			siteName string
		)

		BeforeEach(func() {
			ns = createNamespace()
			siteName = "site-times"
			testClock.SetNow(time.Now())

			sc := testutil.NewTestServiceConfig("web", ns, "web")
			Expect(k8sClient.Create(ctx, sc)).To(Succeed())

			site := testutil.NewTestStagingSite(siteName, ns, map[string]sitev1.StagingSiteService{
				"web": {
					ImageTag: "latest",
					Replicas: 1,
				},
			})
			site.Annotations = map[string]string{}
			site.Spec.DisableAfter = sitev1.TimeInterval{Days: 2}
			site.Spec.DeleteAfter = sitev1.TimeInterval{Days: 7}
			Expect(k8sClient.Create(ctx, site)).To(Succeed())
		})

		It("should set DisableAt approximately 2 days in the future and DeleteAt approximately 7 days in the future", func() {
			fetched := &sitev1.StagingSite{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: siteName, Namespace: ns}, fetched)).To(Succeed())
				g.Expect(fetched.Status.DisableAt).NotTo(BeNil())
				g.Expect(fetched.Status.DeleteAt).NotTo(BeNil())

				now := testClock.Now()
				disableDelta := fetched.Status.DisableAt.Time.Sub(now)
				deleteDelta := fetched.Status.DeleteAt.Time.Sub(now)

				g.Expect(disableDelta).To(BeNumerically("~", 2*24*time.Hour, 5*time.Minute))
				g.Expect(deleteDelta).To(BeNumerically("~", 7*24*time.Hour, 5*time.Minute))
			}, timeout, interval).Should(Succeed())
		})
	})

	Describe("DisableAfter with Never flag", func() {
		var (
			ns       string
			siteName string
		)

		BeforeEach(func() {
			ns = createNamespace()
			siteName = "site-never"
			testClock.SetNow(time.Now())

			sc := testutil.NewTestServiceConfig("web", ns, "web")
			Expect(k8sClient.Create(ctx, sc)).To(Succeed())

			site := testutil.NewTestStagingSite(siteName, ns, map[string]sitev1.StagingSiteService{
				"web": {
					ImageTag: "latest",
					Replicas: 1,
				},
			})
			site.Annotations = map[string]string{}
			site.Spec.DisableAfter = sitev1.TimeInterval{Never: true}
			site.Spec.DeleteAfter = sitev1.TimeInterval{Days: 7}
			Expect(k8sClient.Create(ctx, site)).To(Succeed())
		})

		It("should leave DisableAt nil", func() {
			fetched := &sitev1.StagingSite{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: siteName, Namespace: ns}, fetched)).To(Succeed())
				g.Expect(fetched.Status.DeleteAt).NotTo(BeNil())
			}, timeout, interval).Should(Succeed())

			Consistently(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: siteName, Namespace: ns}, fetched)).To(Succeed())
				g.Expect(fetched.Status.DisableAt).To(BeNil())
			}, 1*time.Second, interval).Should(Succeed())
		})
	})

	Describe("Service status populated", func() {
		var (
			ns       string
			siteName string
		)

		BeforeEach(func() {
			ns = createNamespace()
			siteName = "site-svc"
			testClock.SetNow(time.Now())

			sc := testutil.NewTestServiceConfig("web", ns, "web")
			Expect(k8sClient.Create(ctx, sc)).To(Succeed())

			site := testutil.NewTestStagingSite(siteName, ns, map[string]sitev1.StagingSiteService{
				"web": {
					ImageTag: "latest",
					Replicas: 1,
				},
			})
			site.Annotations = map[string]string{}
			Expect(k8sClient.Create(ctx, site)).To(Succeed())
		})

		It("should populate Status.Services with username and dbName", func() {
			fetched := &sitev1.StagingSite{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: siteName, Namespace: ns}, fetched)).To(Succeed())
				g.Expect(fetched.Status.Services).To(HaveKey("web"))
				svcStatus := fetched.Status.Services["web"]
				g.Expect(svcStatus.Username).NotTo(BeEmpty())
				g.Expect(svcStatus.DbName).NotTo(BeEmpty())
			}, timeout, interval).Should(Succeed())
		})
	})

	Describe("Database task resources created", func() {
		var (
			ns       string
			siteName string
		)

		BeforeEach(func() {
			ns = createNamespace()
			siteName = "site-db"
			testClock.SetNow(time.Now())

			mysqlCfg := &configv1.MysqlConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mysql1",
					Namespace: ns,
				},
				Spec: configv1.MysqlConfigSpec{
					Host:     "mysql.example.com",
					Username: "admin",
					Password: "adminpass",
					Port:     3306,
				},
			}
			Expect(k8sClient.Create(ctx, mysqlCfg)).To(Succeed())

			sc := testutil.NewTestServiceConfig("web", ns, "web")
			sc.Spec.DefaultMysqlEnvironment = "mysql1"
			Expect(k8sClient.Create(ctx, sc)).To(Succeed())

			site := testutil.NewTestStagingSite(siteName, ns, map[string]sitev1.StagingSiteService{
				"web": {
					ImageTag:         "latest",
					Replicas:         1,
					MysqlEnvironment: "mysql1",
				},
			})
			site.Annotations = map[string]string{}
			Expect(k8sClient.Create(ctx, site)).To(Succeed())
		})

		It("should create a MysqlDatabase resource", func() {
			Eventually(func(g Gomega) {
				var list taskv1.MysqlDatabaseList
				g.Expect(k8sClient.List(ctx, &list, client.InNamespace(ns))).To(Succeed())
				g.Expect(list.Items).NotTo(BeEmpty())

				found := false
				for _, db := range list.Items {
					if db.Spec.EnvironmentConfig.SiteName == siteName &&
						db.Spec.EnvironmentConfig.ServiceName == "web" &&
						db.Spec.EnvironmentConfig.Environment == "mysql1" {
						found = true
						break
					}
				}
				g.Expect(found).To(BeTrue(), "expected a MysqlDatabase for site %s with service web and environment mysql1", siteName)
			}, timeout, interval).Should(Succeed())
		})
	})

	Describe("Site deletion removes finalizer", func() {
		var (
			ns       string
			siteName string
		)

		BeforeEach(func() {
			ns = createNamespace()
			siteName = "site-delete"
			testClock.SetNow(time.Now())

			sc := testutil.NewTestServiceConfig("web", ns, "web")
			Expect(k8sClient.Create(ctx, sc)).To(Succeed())

			site := testutil.NewTestStagingSite(siteName, ns, map[string]sitev1.StagingSiteService{
				"web": {
					ImageTag: "latest",
					Replicas: 1,
				},
			})
			site.Annotations = map[string]string{}
			site.Spec.BackupBeforeDelete = false
			Expect(k8sClient.Create(ctx, site)).To(Succeed())

			fetched := &sitev1.StagingSite{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: siteName, Namespace: ns}, fetched)).To(Succeed())
				g.Expect(fetched.Finalizers).To(ContainElement(helpers.SiteFinalizerName))
			}, timeout, interval).Should(Succeed())
		})

		It("should remove the finalizer and allow the object to be deleted", func() {
			site := &sitev1.StagingSite{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: siteName, Namespace: ns}, site)).To(Succeed())
			Expect(k8sClient.Delete(ctx, site)).To(Succeed())

			Eventually(func(g Gomega) {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: siteName, Namespace: ns}, &sitev1.StagingSite{})
				g.Expect(err).To(HaveOccurred())
				g.Expect(client.IgnoreNotFound(err)).To(Succeed())
			}, timeout, interval).Should(Succeed())
		})
	})

	Describe("MongoDatabase task resource created", func() {
		var (
			ns       string
			siteName string
		)

		BeforeEach(func() {
			ns = createNamespace()
			siteName = "site-mongo"
			testClock.SetNow(time.Now())

			mongoCfg := testutil.NewTestMongoConfig("mongo1", ns)
			Expect(k8sClient.Create(ctx, mongoCfg)).To(Succeed())

			sc := testutil.NewTestServiceConfig("web", ns, "web")
			sc.Spec.DefaultMongoEnvironment = "mongo1"
			Expect(k8sClient.Create(ctx, sc)).To(Succeed())

			site := testutil.NewTestStagingSite(siteName, ns, map[string]sitev1.StagingSiteService{
				"web": {
					ImageTag:         "latest",
					Replicas:         1,
					MongoEnvironment: "mongo1",
				},
			})
			site.Annotations = map[string]string{}
			Expect(k8sClient.Create(ctx, site)).To(Succeed())
		})

		It("should create a MongoDatabase resource", func() {
			Eventually(func(g Gomega) {
				var list taskv1.MongoDatabaseList
				g.Expect(k8sClient.List(ctx, &list, client.InNamespace(ns))).To(Succeed())
				g.Expect(list.Items).NotTo(BeEmpty())

				found := false
				for _, db := range list.Items {
					if db.Spec.EnvironmentConfig.SiteName == siteName &&
						db.Spec.EnvironmentConfig.ServiceName == "web" &&
						db.Spec.EnvironmentConfig.Environment == "mongo1" {
						found = true
						break
					}
				}
				g.Expect(found).To(BeTrue(), "expected a MongoDatabase for site %s with service web and environment mongo1", siteName)
			}, timeout, interval).Should(Succeed())
		})
	})

	Describe("RedisDatabase task resource created", func() {
		var (
			ns       string
			siteName string
		)

		BeforeEach(func() {
			ns = createNamespace()
			siteName = "site-redis"
			testClock.SetNow(time.Now())

			redisCfg := testutil.NewTestRedisConfig("redis1", ns)
			Expect(k8sClient.Create(ctx, redisCfg)).To(Succeed())

			sc := testutil.NewTestServiceConfig("web", ns, "web")
			sc.Spec.DefaultRedisEnvironment = "redis1"
			Expect(k8sClient.Create(ctx, sc)).To(Succeed())

			site := testutil.NewTestStagingSite(siteName, ns, map[string]sitev1.StagingSiteService{
				"web": {
					ImageTag:         "latest",
					Replicas:         1,
					RedisEnvironment: "redis1",
				},
			})
			site.Annotations = map[string]string{}
			Expect(k8sClient.Create(ctx, site)).To(Succeed())
		})

		It("should create a RedisDatabase resource", func() {
			Eventually(func(g Gomega) {
				var list taskv1.RedisDatabaseList
				g.Expect(k8sClient.List(ctx, &list, client.InNamespace(ns))).To(Succeed())
				g.Expect(list.Items).NotTo(BeEmpty())

				found := false
				for _, db := range list.Items {
					if db.Spec.EnvironmentConfig.SiteName == siteName &&
						db.Spec.EnvironmentConfig.ServiceName == "web" &&
						db.Spec.EnvironmentConfig.Environment == "redis1" {
						found = true
						break
					}
				}
				g.Expect(found).To(BeTrue(), "expected a RedisDatabase for site %s with service web and environment redis1", siteName)
			}, timeout, interval).Should(Succeed())
		})
	})

	Describe("Site reaches Complete state with no databases", func() {
		var (
			ns       string
			siteName string
		)

		BeforeEach(func() {
			ns = createNamespace()
			siteName = "site-complete"
			testClock.SetNow(time.Now())

			sc := testutil.NewTestServiceConfig("web", ns, "web")
			Expect(k8sClient.Create(ctx, sc)).To(Succeed())

			site := testutil.NewTestStagingSite(siteName, ns, map[string]sitev1.StagingSiteService{
				"web": {
					ImageTag: "latest",
					Replicas: 1,
				},
			})
			site.Annotations = map[string]string{}
			Expect(k8sClient.Create(ctx, site)).To(Succeed())
		})

		It("should eventually reach Complete state", func() {
			fetched := &sitev1.StagingSite{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: siteName, Namespace: ns}, fetched)).To(Succeed())
				g.Expect(fetched.Status.State).To(Equal(sitev1.StateComplete))
			}, timeout, interval).Should(Succeed())

			Expect(fetched.Status.DatabaseCreationComplete).To(BeTrue())
			Expect(fetched.Status.DatabaseInitialisationComplete).To(BeTrue())
			Expect(fetched.Status.DatabaseMigrationsComplete).To(BeTrue())
			Expect(fetched.Status.ConfigsAreCreated).To(BeTrue())
			Expect(fetched.Status.WorkloadsAreCreated).To(BeTrue())
			Expect(fetched.Status.NetworkingObjectsAreCreated).To(BeTrue())
		})
	})

	Describe("Disabled site behavior", func() {
		var (
			ns       string
			siteName string
		)

		BeforeEach(func() {
			ns = createNamespace()
			siteName = "site-disabled"
			testClock.SetNow(time.Now())

			sc := testutil.NewTestServiceConfig("web", ns, "web")
			Expect(k8sClient.Create(ctx, sc)).To(Succeed())

			site := testutil.NewTestStagingSite(siteName, ns, map[string]sitev1.StagingSiteService{
				"web": {
					ImageTag: "latest",
					Replicas: 1,
				},
			})
			site.Annotations = map[string]string{}
			site.Spec.Enabled = false
			Expect(k8sClient.Create(ctx, site)).To(Succeed())
		})

		It("should set Status.Enabled to false", func() {
			fetched := &sitev1.StagingSite{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: siteName, Namespace: ns}, fetched)).To(Succeed())
				g.Expect(fetched.Status.State).NotTo(BeEmpty())
				g.Expect(fetched.Status.Enabled).To(BeFalse())
			}, timeout, interval).Should(Succeed())
		})
	})

	Describe("Auto-disable when DisableAt time passes", func() {
		var (
			ns       string
			siteName string
		)

		BeforeEach(func() {
			ns = createNamespace()
			siteName = "site-autodisable"
			testClock.SetNow(time.Now())

			sc := testutil.NewTestServiceConfig("web", ns, "web")
			Expect(k8sClient.Create(ctx, sc)).To(Succeed())

			site := testutil.NewTestStagingSite(siteName, ns, map[string]sitev1.StagingSiteService{
				"web": {
					ImageTag: "latest",
					Replicas: 1,
				},
			})
			site.Annotations = map[string]string{}
			site.Spec.DisableAfter = sitev1.TimeInterval{Minutes: 1}
			site.Spec.DeleteAfter = sitev1.TimeInterval{Never: true}
			Expect(k8sClient.Create(ctx, site)).To(Succeed())
		})

		It("should disable the site when the clock advances past DisableAt", func() {
			fetched := &sitev1.StagingSite{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: siteName, Namespace: ns}, fetched)).To(Succeed())
				g.Expect(fetched.Status.DisableAt).NotTo(BeNil())
				g.Expect(fetched.Status.Enabled).To(BeTrue())
			}, timeout, interval).Should(Succeed())

			testClock.SetNow(fetched.Status.DisableAt.Add(1 * time.Minute))

			// envtest won't re-enqueue after the deadline passes, so write an annotation to force reconciliation
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: siteName, Namespace: ns}, fetched)).To(Succeed())
				if fetched.Annotations == nil {
					fetched.Annotations = map[string]string{}
				}
				fetched.Annotations["test-trigger"] = "autodisable"
				g.Expect(k8sClient.Update(ctx, fetched)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: siteName, Namespace: ns}, fetched)).To(Succeed())
				g.Expect(fetched.Status.Enabled).To(BeFalse())
			}, timeout, interval).Should(Succeed())
		})
	})

	Describe("Auto-delete when DeleteAt time passes", func() {
		var (
			ns       string
			siteName string
		)

		BeforeEach(func() {
			ns = createNamespace()
			siteName = "site-autodelete"
			testClock.SetNow(time.Now())

			sc := testutil.NewTestServiceConfig("web", ns, "web")
			Expect(k8sClient.Create(ctx, sc)).To(Succeed())

			site := testutil.NewTestStagingSite(siteName, ns, map[string]sitev1.StagingSiteService{
				"web": {
					ImageTag: "latest",
					Replicas: 1,
				},
			})
			site.Annotations = map[string]string{}
			site.Spec.DisableAfter = sitev1.TimeInterval{Never: true}
			site.Spec.DeleteAfter = sitev1.TimeInterval{Minutes: 1}
			site.Spec.BackupBeforeDelete = false
			Expect(k8sClient.Create(ctx, site)).To(Succeed())
		})

		It("should delete the site when the clock advances past DeleteAt", func() {
			fetched := &sitev1.StagingSite{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: siteName, Namespace: ns}, fetched)).To(Succeed())
				g.Expect(fetched.Status.DeleteAt).NotTo(BeNil())
			}, timeout, interval).Should(Succeed())

			testClock.SetNow(fetched.Status.DeleteAt.Add(1 * time.Minute))

			// envtest won't re-enqueue after the deadline passes, so write an annotation to force reconciliation
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: siteName, Namespace: ns}, fetched)).To(Succeed())
				if fetched.Annotations == nil {
					fetched.Annotations = map[string]string{}
				}
				fetched.Annotations["test-trigger"] = "autodelete"
				g.Expect(k8sClient.Update(ctx, fetched)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			Eventually(func(g Gomega) {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: siteName, Namespace: ns}, &sitev1.StagingSite{})
				g.Expect(err).To(HaveOccurred())
				g.Expect(client.IgnoreNotFound(err)).To(Succeed())
			}, timeout, interval).Should(Succeed())
		})
	})

	Describe("Multiple services", func() {
		var (
			ns       string
			siteName string
		)

		BeforeEach(func() {
			ns = createNamespace()
			siteName = "site-multi"
			testClock.SetNow(time.Now())

			sc1 := testutil.NewTestServiceConfig("web", ns, "web")
			Expect(k8sClient.Create(ctx, sc1)).To(Succeed())

			sc2 := testutil.NewTestServiceConfig("api", ns, "api")
			Expect(k8sClient.Create(ctx, sc2)).To(Succeed())

			site := testutil.NewTestStagingSite(siteName, ns, map[string]sitev1.StagingSiteService{
				"web": {
					ImageTag: "latest",
					Replicas: 1,
				},
				"api": {
					ImageTag: "v1.0.0",
					Replicas: 1,
				},
			})
			site.Annotations = map[string]string{}
			Expect(k8sClient.Create(ctx, site)).To(Succeed())
		})

		It("should populate status for all services", func() {
			fetched := &sitev1.StagingSite{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: siteName, Namespace: ns}, fetched)).To(Succeed())
				g.Expect(fetched.Status.Services).To(HaveKey("web"))
				g.Expect(fetched.Status.Services).To(HaveKey("api"))
				g.Expect(fetched.Status.Services["web"].Username).NotTo(BeEmpty())
				g.Expect(fetched.Status.Services["api"].Username).NotTo(BeEmpty())
				g.Expect(fetched.Status.Services["web"].DbName).NotTo(BeEmpty())
				g.Expect(fetched.Status.Services["api"].DbName).NotTo(BeEmpty())
			}, timeout, interval).Should(Succeed())
		})
	})

	Describe("DeleteAfter and DisableAfter both set to Never", func() {
		var (
			ns       string
			siteName string
		)

		BeforeEach(func() {
			ns = createNamespace()
			siteName = "site-both-never"
			testClock.SetNow(time.Now())

			sc := testutil.NewTestServiceConfig("web", ns, "web")
			Expect(k8sClient.Create(ctx, sc)).To(Succeed())

			site := testutil.NewTestStagingSite(siteName, ns, map[string]sitev1.StagingSiteService{
				"web": {
					ImageTag: "latest",
					Replicas: 1,
				},
			})
			site.Annotations = map[string]string{}
			site.Spec.DisableAfter = sitev1.TimeInterval{Never: true}
			site.Spec.DeleteAfter = sitev1.TimeInterval{Never: true}
			Expect(k8sClient.Create(ctx, site)).To(Succeed())
		})

		It("should leave both DisableAt and DeleteAt nil", func() {
			fetched := &sitev1.StagingSite{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: siteName, Namespace: ns}, fetched)).To(Succeed())
				g.Expect(fetched.Status.State).NotTo(BeEmpty())
				g.Expect(fetched.Annotations).To(HaveKey(annotations.StagingSiteLastSpecChangeAt))
			}, timeout, interval).Should(Succeed())

			Consistently(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: siteName, Namespace: ns}, fetched)).To(Succeed())
				g.Expect(fetched.Status.DisableAt).To(BeNil())
				g.Expect(fetched.Status.DeleteAt).To(BeNil())
			}, 1*time.Second, interval).Should(Succeed())
		})
	})

	Describe("Deployment and Service creation", func() {
		var (
			ns       string
			siteName string
		)

		BeforeEach(func() {
			ns = createNamespace()
			siteName = "site-workload"
			testClock.SetNow(time.Now())

			sc := testutil.NewTestServiceConfigWithDefaults("web", ns, "web")
			Expect(k8sClient.Create(ctx, sc)).To(Succeed())

			site := testutil.NewTestStagingSite(siteName, ns, map[string]sitev1.StagingSiteService{
				"web": {
					ImageTag: "latest",
					Replicas: 1,
				},
			})
			site.Annotations = map[string]string{}
			Expect(k8sClient.Create(ctx, site)).To(Succeed())
		})

		It("should create Deployment and Service objects", func() {
			Eventually(func(g Gomega) {
				var deployments appsv1.DeploymentList
				g.Expect(k8sClient.List(ctx, &deployments, client.InNamespace(ns))).To(Succeed())
				g.Expect(deployments.Items).NotTo(BeEmpty())
			}, timeout, interval).Should(Succeed())

			Eventually(func(g Gomega) {
				var services corev1.ServiceList
				g.Expect(k8sClient.List(ctx, &services, client.InNamespace(ns))).To(Succeed())
				// Filter out the kubernetes default service
				found := false
				for _, svc := range services.Items {
					if svc.Name != "kubernetes" {
						found = true
						break
					}
				}
				g.Expect(found).To(BeTrue(), "expected at least one non-default Service")
			}, timeout, interval).Should(Succeed())
		})
	})

	Describe("Missing ServiceConfig error handling", func() {
		var (
			ns       string
			siteName string
		)

		BeforeEach(func() {
			ns = createNamespace()
			siteName = "site-missing-sc"
			testClock.SetNow(time.Now())

			// Do NOT create the ServiceConfig - reference a non-existent one
			site := testutil.NewTestStagingSite(siteName, ns, map[string]sitev1.StagingSiteService{
				"nonexistent": {
					ImageTag: "latest",
					Replicas: 1,
				},
			})
			site.Annotations = map[string]string{}
			Expect(k8sClient.Create(ctx, site)).To(Succeed())
		})

		It("should not reach Complete state", func() {
			fetched := &sitev1.StagingSite{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: siteName, Namespace: ns}, fetched)).To(Succeed())
				g.Expect(fetched.Finalizers).To(ContainElement(helpers.SiteFinalizerName))
			}, timeout, interval).Should(Succeed())

			Consistently(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: siteName, Namespace: ns}, fetched)).To(Succeed())
				g.Expect(fetched.Status.State).NotTo(Equal(sitev1.StateComplete))
			}, 2*time.Second, interval).Should(Succeed())
		})
	})

	Describe("BackupBeforeDelete=true during deletion", func() {
		var (
			ns       string
			siteName string
		)

		BeforeEach(func() {
			ns = createNamespace()
			siteName = "site-backup-del"
			testClock.SetNow(time.Now())

			sc := testutil.NewTestServiceConfig("web", ns, "web")
			Expect(k8sClient.Create(ctx, sc)).To(Succeed())

			site := testutil.NewTestStagingSite(siteName, ns, map[string]sitev1.StagingSiteService{
				"web": {
					ImageTag: "latest",
					Replicas: 1,
				},
			})
			site.Annotations = map[string]string{}
			site.Spec.BackupBeforeDelete = true
			Expect(k8sClient.Create(ctx, site)).To(Succeed())

			fetched := &sitev1.StagingSite{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: siteName, Namespace: ns}, fetched)).To(Succeed())
				g.Expect(fetched.Status.State).To(Equal(sitev1.StateComplete))
			}, timeout, interval).Should(Succeed())
		})

		It("should not delete the site immediately because the backup is not complete", func() {
			site := &sitev1.StagingSite{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: siteName, Namespace: ns}, site)).To(Succeed())
			Expect(k8sClient.Delete(ctx, site)).To(Succeed())

			Eventually(func(g Gomega) {
				var backups jobv1.BackupList
				g.Expect(k8sClient.List(ctx, &backups, client.InNamespace(ns))).To(Succeed())
				g.Expect(backups.Items).NotTo(BeEmpty())
			}, timeout, interval).Should(Succeed())

			Consistently(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: siteName, Namespace: ns}, &sitev1.StagingSite{})).To(Succeed())
			}, 2*time.Second, interval).Should(Succeed())
		})
	})

	Describe("DailyBackupWindowHour scheduling", func() {
		var (
			ns       string
			siteName string
		)

		BeforeEach(func() {
			ns = createNamespace()
			siteName = "site-bkp-sched"
			testClock.SetNow(time.Now())

			sc := testutil.NewTestServiceConfig("web", ns, "web")
			Expect(k8sClient.Create(ctx, sc)).To(Succeed())

			backupHour := int32(14)
			site := testutil.NewTestStagingSite(siteName, ns, map[string]sitev1.StagingSiteService{
				"web": {
					ImageTag: "latest",
					Replicas: 1,
				},
			})
			site.Annotations = map[string]string{}
			site.Spec.DailyBackupWindowHour = &backupHour
			Expect(k8sClient.Create(ctx, site)).To(Succeed())
		})

		It("should set NextBackupTime with the configured hour", func() {
			fetched := &sitev1.StagingSite{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: siteName, Namespace: ns}, fetched)).To(Succeed())
				g.Expect(fetched.Status.NextBackupTime).NotTo(BeNil())
				g.Expect(fetched.Status.NextBackupTime.UTC().Hour()).To(Equal(14))
			}, timeout, interval).Should(Succeed())
		})

		It("should create a Scheduled Backup when clock advances past NextBackupTime", func() {
			fetched := &sitev1.StagingSite{}

			// Wait for site to reach Complete and have NextBackupTime set
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: siteName, Namespace: ns}, fetched)).To(Succeed())
				g.Expect(fetched.Status.State).To(Equal(sitev1.StateComplete))
				g.Expect(fetched.Status.NextBackupTime).NotTo(BeNil())
			}, 30*time.Second, interval).Should(Succeed())

			testClock.SetNow(fetched.Status.NextBackupTime.Add(1 * time.Minute))

			// envtest won't re-enqueue after the deadline passes, so write an annotation to force reconciliation
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: siteName, Namespace: ns}, fetched)).To(Succeed())
				if fetched.Annotations == nil {
					fetched.Annotations = map[string]string{}
				}
				fetched.Annotations["test-trigger"] = "backup-schedule"
				g.Expect(k8sClient.Update(ctx, fetched)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			Eventually(func(g Gomega) {
				var backups jobv1.BackupList
				g.Expect(k8sClient.List(ctx, &backups, client.InNamespace(ns))).To(Succeed())
				found := false
				for _, b := range backups.Items {
					if b.Spec.BackupType == jobv1.BackupTypeScheduled {
						found = true
						break
					}
				}
				g.Expect(found).To(BeTrue(), "expected a Scheduled Backup to be created")
			}, 30*time.Second, interval).Should(Succeed())
		})
	})

	Describe("Spec change re-triggers reconciliation", func() {
		var (
			ns       string
			siteName string
		)

		BeforeEach(func() {
			ns = createNamespace()
			siteName = "site-spec-change"
			testClock.SetNow(time.Now())

			sc := testutil.NewTestServiceConfig("web", ns, "web")
			Expect(k8sClient.Create(ctx, sc)).To(Succeed())

			site := testutil.NewTestStagingSite(siteName, ns, map[string]sitev1.StagingSiteService{
				"web": {
					ImageTag: "latest",
					Replicas: 1,
				},
			})
			site.Annotations = map[string]string{}
			Expect(k8sClient.Create(ctx, site)).To(Succeed())
		})

		It("should reset state to Pending when imageTag changes and eventually return to Complete", func() {
			fetched := &sitev1.StagingSite{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: siteName, Namespace: ns}, fetched)).To(Succeed())
				g.Expect(fetched.Status.State).To(Equal(sitev1.StateComplete))
			}, timeout, interval).Should(Succeed())

			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: siteName, Namespace: ns}, fetched)).To(Succeed())
				services := fetched.Spec.Services
				svc := services["web"]
				svc.ImageTag = "v2.0.0"
				services["web"] = svc
				fetched.Spec.Services = services
				g.Expect(k8sClient.Update(ctx, fetched)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: siteName, Namespace: ns}, fetched)).To(Succeed())
				g.Expect(fetched.Status.State).To(Equal(sitev1.StateComplete))
			}, timeout, interval).Should(Succeed())
		})
	})
})
