package task

import (
	"fmt"
	"time"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	configv1 "github.com/szeber/kube-stager/apis/config/v1"
	taskv1 "github.com/szeber/kube-stager/apis/task/v1"
	"github.com/szeber/kube-stager/internal/testutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("RedisDatabaseController", func() {
	const (
		timeout  = 10 * time.Second
		interval = 200 * time.Millisecond
	)

	Describe("when a RedisDatabase and RedisConfig exist", func() {
		var (
			ns        string
			envName   string
			dbName    string
			configObj *configv1.RedisConfig
			dbObj     *taskv1.RedisDatabase
		)

		BeforeEach(func() {
			ns = fmt.Sprintf("redis-ok-%d", GinkgoParallelProcess())
			envName = "redis-env"
			dbName = "redis-db"

			nsObj := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: ns},
			}
			Expect(k8sClient.Create(ctx, nsObj)).To(Succeed())

			mockRedisReconciler.SetReconcileFunc(func(
				database *taskv1.RedisDatabase,
				config configv1.RedisConfig,
				logger logr.Logger,
			) (bool, error) {
				database.Status.State = taskv1.Complete
				return true, nil
			})

			configObj = testutil.NewTestRedisConfig(envName, ns)
			Expect(k8sClient.Create(ctx, configObj)).To(Succeed())

			dbObj = testutil.NewTestRedisDatabase(dbName, ns, "site1", "svc1", envName, 1)
			Expect(k8sClient.Create(ctx, dbObj)).To(Succeed())
		})

		AfterEach(func() {
			mockRedisReconciler.SetReconcileFunc(nil)
		})

		It("should reconcile to Complete", func() {
			fetched := &taskv1.RedisDatabase{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(dbObj), fetched)).To(Succeed())
				g.Expect(fetched.Status.State).To(Equal(taskv1.Complete))
			}, timeout, interval).Should(Succeed())
		})
	})

	Describe("when the RedisConfig does not exist", func() {
		var (
			ns     string
			dbName string
			dbObj  *taskv1.RedisDatabase
		)

		BeforeEach(func() {
			ns = fmt.Sprintf("redis-noconfig-%d", GinkgoParallelProcess())
			dbName = "redis-db-noconfig"

			nsObj := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: ns},
			}
			Expect(k8sClient.Create(ctx, nsObj)).To(Succeed())

			mockRedisReconciler.SetReconcileFunc(func(
				database *taskv1.RedisDatabase,
				config configv1.RedisConfig,
				logger logr.Logger,
			) (bool, error) {
				database.Status.State = taskv1.Complete
				return true, nil
			})

			dbObj = testutil.NewTestRedisDatabase(dbName, ns, "site1", "svc1", "nonexistent-env", 2)
			Expect(k8sClient.Create(ctx, dbObj)).To(Succeed())
		})

		AfterEach(func() {
			mockRedisReconciler.SetReconcileFunc(nil)
		})

		It("should not reach Complete status", func() {
			fetched := &taskv1.RedisDatabase{}
			Consistently(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(dbObj), fetched)).To(Succeed())
				g.Expect(fetched.Status.State).NotTo(Equal(taskv1.Complete))
			}, 2*time.Second, interval).Should(Succeed())
		})
	})

	Describe("when the mock reconciler returns an error", func() {
		var (
			ns        string
			envName   string
			dbName    string
			configObj *configv1.RedisConfig
			dbObj     *taskv1.RedisDatabase
		)

		BeforeEach(func() {
			ns = fmt.Sprintf("redis-err-%d", GinkgoParallelProcess())
			envName = "redis-env-err"
			dbName = "redis-db-err"

			nsObj := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: ns},
			}
			Expect(k8sClient.Create(ctx, nsObj)).To(Succeed())

			mockRedisReconciler.SetReconcileFunc(func(
				database *taskv1.RedisDatabase,
				config configv1.RedisConfig,
				logger logr.Logger,
			) (bool, error) {
				return false, fmt.Errorf("mock reconcile error")
			})

			configObj = testutil.NewTestRedisConfig(envName, ns)
			Expect(k8sClient.Create(ctx, configObj)).To(Succeed())

			dbObj = testutil.NewTestRedisDatabase(dbName, ns, "site1", "svc1", envName, 3)
			Expect(k8sClient.Create(ctx, dbObj)).To(Succeed())
		})

		AfterEach(func() {
			mockRedisReconciler.SetReconcileFunc(nil)
		})

		It("should not reach Complete status", func() {
			fetched := &taskv1.RedisDatabase{}
			Consistently(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(dbObj), fetched)).To(Succeed())
				g.Expect(fetched.Status.State).NotTo(Equal(taskv1.Complete))
			}, 2*time.Second, interval).Should(Succeed())
		})
	})
})
