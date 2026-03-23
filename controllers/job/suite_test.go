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

package job

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	controllerconfigv1 "github.com/szeber/kube-stager/apis/controller-config/v1"
	"github.com/szeber/kube-stager/internal/testutil"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment
var ctx context.Context
var cancel context.CancelFunc

var testClock *testutil.MockClock

func TestAPIs(t *testing.T) {
	testutil.SafeTestMain(t)
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(
	func() {
		logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

		ctx, cancel = context.WithCancel(context.TODO())

		By("bootstrapping test environment")
		useExistingCluster := false
		testEnv = &envtest.Environment{
			CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
			ErrorIfCRDPathMissing: true,
			UseExistingCluster:    &useExistingCluster,
		}

		var err error
		cfg, err = testEnv.Start()
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg).NotTo(BeNil())

		testScheme := testutil.NewTestScheme()

		//+kubebuilder:scaffold:scheme

		k8sClient, err = client.New(cfg, client.Options{Scheme: testScheme})
		Expect(err).NotTo(HaveOccurred())
		Expect(k8sClient).NotTo(BeNil())

		testClock = &testutil.MockClock{}
		testClock.SetNow(time.Now())

		mgr, err := ctrl.NewManager(cfg, ctrl.Options{
			Scheme: testScheme,
		})
		Expect(err).NotTo(HaveOccurred())

		err = (&BackupReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
			Config: controllerconfigv1.ProjectConfig{},
			Clock:  testClock,
		}).SetupWithManager(mgr)
		Expect(err).NotTo(HaveOccurred())

		err = (&DbInitJobReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
			Config: controllerconfigv1.ProjectConfig{},
		}).SetupWithManager(mgr)
		Expect(err).NotTo(HaveOccurred())

		err = (&DbMigrationJobReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
			Config: controllerconfigv1.ProjectConfig{},
		}).SetupWithManager(mgr)
		Expect(err).NotTo(HaveOccurred())

		go func() {
			defer GinkgoRecover()
			if err := mgr.Start(ctx); err != nil {
				Expect(err).NotTo(HaveOccurred())
			}
		}()

		// Wait for the manager's cache to sync before running tests
		Expect(mgr.GetCache().WaitForCacheSync(ctx)).To(BeTrue())
	},
)

var _ = AfterSuite(
	func() {
		cancel()
		By("tearing down the test environment")
		err := testEnv.Stop()
		Expect(err).NotTo(HaveOccurred())
	},
)
