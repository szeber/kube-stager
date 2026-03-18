package task

import (
	"fmt"
	"time"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	configv1 "github.com/szeber/kube-stager/apis/config/v1"
	taskv1 "github.com/szeber/kube-stager/apis/task/v1"
	"github.com/szeber/kube-stager/helpers"
	"github.com/szeber/kube-stager/internal/testutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("MongoDatabaseController", func() {
	const (
		timeout  = 10 * time.Second
		interval = 200 * time.Millisecond
	)

	Describe("when a MongoDatabase and MongoConfig exist", func() {
		var (
			ns        string
			envName   string
			dbName    string
			configObj *configv1.MongoConfig
			dbObj     *taskv1.MongoDatabase
		)

		BeforeEach(func() {
			ns = fmt.Sprintf("mongo-ok-%d", GinkgoParallelProcess())
			envName = "mongo-env"
			dbName = "mongo-db"

			nsObj := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: ns},
			}
			Expect(k8sClient.Create(ctx, nsObj)).To(Succeed())

			mockMongoReconciler.SetReconcileFunc(func(
				database *taskv1.MongoDatabase,
				config configv1.MongoConfig,
				logger logr.Logger,
			) (bool, error) {
				database.Status.State = taskv1.Complete
				return true, nil
			})
			mockMongoReconciler.SetDeleteFunc(nil)

			configObj = testutil.NewTestMongoConfig(envName, ns)
			Expect(k8sClient.Create(ctx, configObj)).To(Succeed())

			dbObj = testutil.NewTestMongoDatabase(dbName, ns, "site1", "svc1", envName)
			Expect(k8sClient.Create(ctx, dbObj)).To(Succeed())
		})

		AfterEach(func() {
			mockMongoReconciler.SetReconcileFunc(nil)
			mockMongoReconciler.SetDeleteFunc(nil)
		})

		It("should add the finalizer and reconcile to Complete", func() {
			fetched := &taskv1.MongoDatabase{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(dbObj), fetched)).To(Succeed())
				g.Expect(fetched.ObjectMeta.Finalizers).To(ContainElement(helpers.MongoFinalizerName))
				g.Expect(fetched.Status.State).To(Equal(taskv1.Complete))
			}, timeout, interval).Should(Succeed())
		})
	})

	Describe("when the MongoConfig does not exist", func() {
		var (
			ns     string
			dbName string
			dbObj  *taskv1.MongoDatabase
		)

		BeforeEach(func() {
			ns = fmt.Sprintf("mongo-noconfig-%d", GinkgoParallelProcess())
			dbName = "mongo-db-noconfig"

			nsObj := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: ns},
			}
			Expect(k8sClient.Create(ctx, nsObj)).To(Succeed())

			mockMongoReconciler.SetReconcileFunc(func(
				database *taskv1.MongoDatabase,
				config configv1.MongoConfig,
				logger logr.Logger,
			) (bool, error) {
				database.Status.State = taskv1.Complete
				return true, nil
			})
			mockMongoReconciler.SetDeleteFunc(nil)

			dbObj = testutil.NewTestMongoDatabase(dbName, ns, "site1", "svc1", "nonexistent-env")
			Expect(k8sClient.Create(ctx, dbObj)).To(Succeed())
		})

		AfterEach(func() {
			mockMongoReconciler.SetReconcileFunc(nil)
			mockMongoReconciler.SetDeleteFunc(nil)
		})

		It("should not reach Complete status", func() {
			fetched := &taskv1.MongoDatabase{}
			Consistently(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(dbObj), fetched)).To(Succeed())
				g.Expect(fetched.Status.State).NotTo(Equal(taskv1.Complete))
			}, 2*time.Second, interval).Should(Succeed())
		})
	})

	Describe("when a reconciled MongoDatabase is deleted", func() {
		var (
			ns        string
			envName   string
			dbName    string
			configObj *configv1.MongoConfig
			dbObj     *taskv1.MongoDatabase
		)

		BeforeEach(func() {
			ns = fmt.Sprintf("mongo-del-%d", GinkgoParallelProcess())
			envName = "mongo-env-del"
			dbName = "mongo-db-del"

			nsObj := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: ns},
			}
			Expect(k8sClient.Create(ctx, nsObj)).To(Succeed())

			mockMongoReconciler.SetReconcileFunc(func(
				database *taskv1.MongoDatabase,
				config configv1.MongoConfig,
				logger logr.Logger,
			) (bool, error) {
				database.Status.State = taskv1.Complete
				return true, nil
			})
			mockMongoReconciler.SetDeleteFunc(func(
				database *taskv1.MongoDatabase,
				config configv1.MongoConfig,
				logger logr.Logger,
			) error {
				return nil
			})

			configObj = testutil.NewTestMongoConfig(envName, ns)
			Expect(k8sClient.Create(ctx, configObj)).To(Succeed())

			dbObj = testutil.NewTestMongoDatabase(dbName, ns, "site1", "svc1", envName)
			Expect(k8sClient.Create(ctx, dbObj)).To(Succeed())

			// Wait for the finalizer and Complete status before deleting
			fetched := &taskv1.MongoDatabase{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(dbObj), fetched)).To(Succeed())
				g.Expect(fetched.ObjectMeta.Finalizers).To(ContainElement(helpers.MongoFinalizerName))
				g.Expect(fetched.Status.State).To(Equal(taskv1.Complete))
			}, timeout, interval).Should(Succeed())
		})

		AfterEach(func() {
			mockMongoReconciler.SetReconcileFunc(nil)
			mockMongoReconciler.SetDeleteFunc(nil)
		})

		It("should run the delete handler and remove the resource", func() {
			Expect(k8sClient.Delete(ctx, dbObj)).To(Succeed())

			Eventually(func(g Gomega) {
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(dbObj), &taskv1.MongoDatabase{})
				g.Expect(errors.IsNotFound(err)).To(BeTrue())
			}, timeout, interval).Should(Succeed())
		})
	})

	Describe("when the mock reconciler returns an error", func() {
		var (
			ns        string
			envName   string
			dbName    string
			configObj *configv1.MongoConfig
			dbObj     *taskv1.MongoDatabase
		)

		BeforeEach(func() {
			ns = fmt.Sprintf("mongo-err-%d", GinkgoParallelProcess())
			envName = "mongo-env-err"
			dbName = "mongo-db-err"

			nsObj := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: ns},
			}
			Expect(k8sClient.Create(ctx, nsObj)).To(Succeed())

			mockMongoReconciler.SetReconcileFunc(func(
				database *taskv1.MongoDatabase,
				config configv1.MongoConfig,
				logger logr.Logger,
			) (bool, error) {
				return false, fmt.Errorf("mock reconcile error")
			})
			mockMongoReconciler.SetDeleteFunc(nil)

			configObj = testutil.NewTestMongoConfig(envName, ns)
			Expect(k8sClient.Create(ctx, configObj)).To(Succeed())

			dbObj = testutil.NewTestMongoDatabase(dbName, ns, "site1", "svc1", envName)
			Expect(k8sClient.Create(ctx, dbObj)).To(Succeed())
		})

		AfterEach(func() {
			mockMongoReconciler.SetReconcileFunc(nil)
			mockMongoReconciler.SetDeleteFunc(nil)
		})

		It("should not reach Complete status", func() {
			fetched := &taskv1.MongoDatabase{}
			Consistently(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(dbObj), fetched)).To(Succeed())
				g.Expect(fetched.Status.State).NotTo(Equal(taskv1.Complete))
			}, 2*time.Second, interval).Should(Succeed())
		})
	})
})
