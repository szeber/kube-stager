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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Kubebuilder default values", func() {
	const (
		timeout  = 10 * time.Second
		interval = 200 * time.Millisecond
	)

	Describe("StagingSite defaults", func() {
		var (
			ns       string
			siteName string
		)

		BeforeEach(func() {
			ns = createNamespace()
			siteName = "site-defaults"
			testClock.SetNow(time.Now())

			sc := &configv1.ServiceConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "web",
					Namespace: ns,
				},
				Spec: configv1.ServiceConfigSpec{
					ShortName: "web",
					DeploymentPodSpec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Name: "nginx", Image: "nginx:latest"},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, sc)).To(Succeed())
		})

		It("should apply kubebuilder defaults for enabled, backupBeforeDelete, includeAllServices, imageTag, and includeInBackups", func() {
			// Use unstructured to omit fields so the API server applies defaults
			u := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "site.operator.kube-stager.io/v1",
					"kind":       "StagingSite",
					"metadata": map[string]interface{}{
						"name":      siteName,
						"namespace": ns,
					},
					"spec": map[string]interface{}{
						"domainPrefix": siteName,
						"dbName":       siteName,
						"username":     siteName,
						"password":     "testpassword",
						"services": map[string]interface{}{
							"web": map[string]interface{}{},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, u)).To(Succeed())

			fetched := &sitev1.StagingSite{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: siteName, Namespace: ns}, fetched)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			// StagingSite level defaults
			Expect(fetched.Spec.Enabled).To(BeTrue(), "enabled should default to true")
			Expect(fetched.Spec.BackupBeforeDelete).To(BeFalse(), "backupBeforeDelete should default to false")
			Expect(fetched.Spec.IncludeAllServices).To(BeFalse(), "includeAllServices should default to false")

			// StagingSiteService level defaults
			webService := fetched.Spec.Services["web"]
			Expect(webService.ImageTag).To(Equal("latest"), "imageTag should default to 'latest'")
			// Note: replicas marker (+kubebuilder:default:1) is missing the '=' so the CRD default is not generated.
			// The actual default is applied by the Go struct zero value handling in the controller, not the CRD schema.
			Expect(webService.IncludeInBackups).To(BeFalse(), "includeInBackups should default to false")
		})
	})

	Describe("StagingSite validation markers", func() {
		var (
			ns string
		)

		BeforeEach(func() {
			ns = createNamespace()
		})

		It("should reject username longer than 16 characters", func() {
			u := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "site.operator.kube-stager.io/v1",
					"kind":       "StagingSite",
					"metadata": map[string]interface{}{
						"name":      "site-val-user",
						"namespace": ns,
					},
					"spec": map[string]interface{}{
						"domainPrefix": "test",
						"dbName":       "testdb",
						"username":     "this_is_too_long_",
						"password":     "testpassword",
						"services": map[string]interface{}{
							"web": map[string]interface{}{},
						},
					},
				},
			}
			err := k8sClient.Create(ctx, u)
			Expect(err).To(HaveOccurred(), "expected rejection for username > 16 chars")
		})

		It("should reject password longer than 32 characters", func() {
			u := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "site.operator.kube-stager.io/v1",
					"kind":       "StagingSite",
					"metadata": map[string]interface{}{
						"name":      "site-val-pass",
						"namespace": ns,
					},
					"spec": map[string]interface{}{
						"domainPrefix": "test",
						"dbName":       "testdb",
						"username":     "testuser",
						"password":     "this_password_is_way_too_long_aaa",
						"services": map[string]interface{}{
							"web": map[string]interface{}{},
						},
					},
				},
			}
			err := k8sClient.Create(ctx, u)
			Expect(err).To(HaveOccurred(), "expected rejection for password > 32 chars")
		})
	})

	Describe("Backup defaults", func() {
		var (
			ns string
		)

		BeforeEach(func() {
			ns = createNamespace()

			site := &sitev1.StagingSite{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "backup-default-site",
					Namespace: ns,
				},
				Spec: sitev1.StagingSiteSpec{
					Enabled: true,
				},
			}
			Expect(k8sClient.Create(ctx, site)).To(Succeed())
		})

		It("should default backupType to Manual", func() {
			u := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "job.operator.kube-stager.io/v1",
					"kind":       "Backup",
					"metadata": map[string]interface{}{
						"name":      "backup-defaults",
						"namespace": ns,
					},
					"spec": map[string]interface{}{
						"siteName": "backup-default-site",
					},
				},
			}
			Expect(k8sClient.Create(ctx, u)).To(Succeed())

			fetched := &jobv1.Backup{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: "backup-defaults", Namespace: ns}, fetched)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			Expect(fetched.Spec.BackupType).To(Equal(jobv1.BackupTypeManual), "backupType should default to Manual")
		})
	})

	Describe("MongoConfig defaults", func() {
		var (
			ns string
		)

		BeforeEach(func() {
			ns = createNamespace()
		})

		It("should default port to 27017", func() {
			u := &unstructured.Unstructured{}
			u.SetGroupVersionKind(schema.GroupVersionKind{
				Group:   "config.operator.kube-stager.io",
				Version: "v1",
				Kind:    "MongoConfig",
			})
			u.SetName("mongo-defaults")
			u.SetNamespace(ns)
			Expect(unstructured.SetNestedField(u.Object, "mongo.example.com", "spec", "host1")).To(Succeed())
			Expect(unstructured.SetNestedField(u.Object, "admin", "spec", "username")).To(Succeed())
			Expect(unstructured.SetNestedField(u.Object, "secret", "spec", "password")).To(Succeed())

			Expect(k8sClient.Create(ctx, u)).To(Succeed())

			fetched := &configv1.MongoConfig{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: "mongo-defaults", Namespace: ns}, fetched)).To(Succeed())

			Expect(fetched.Spec.Port).To(Equal(uint16(27017)), "port should default to 27017")
		})
	})

	Describe("MysqlConfig defaults", func() {
		var (
			ns string
		)

		BeforeEach(func() {
			ns = createNamespace()
		})

		It("should default port to 3306", func() {
			u := &unstructured.Unstructured{}
			u.SetGroupVersionKind(schema.GroupVersionKind{
				Group:   "config.operator.kube-stager.io",
				Version: "v1",
				Kind:    "MysqlConfig",
			})
			u.SetName("mysql-defaults")
			u.SetNamespace(ns)
			Expect(unstructured.SetNestedField(u.Object, "mysql.example.com", "spec", "host")).To(Succeed())
			Expect(unstructured.SetNestedField(u.Object, "admin", "spec", "username")).To(Succeed())
			Expect(unstructured.SetNestedField(u.Object, "secret", "spec", "password")).To(Succeed())

			Expect(k8sClient.Create(ctx, u)).To(Succeed())

			fetched := &configv1.MysqlConfig{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: "mysql-defaults", Namespace: ns}, fetched)).To(Succeed())

			Expect(fetched.Spec.Port).To(Equal(uint16(3306)), "port should default to 3306")
		})
	})

	Describe("RedisConfig defaults", func() {
		var (
			ns string
		)

		BeforeEach(func() {
			ns = createNamespace()
		})

		It("should default port, availableDatabaseCount, isTlsEnabled, and verifyTlsServerCertificate", func() {
			u := &unstructured.Unstructured{}
			u.SetGroupVersionKind(schema.GroupVersionKind{
				Group:   "config.operator.kube-stager.io",
				Version: "v1",
				Kind:    "RedisConfig",
			})
			u.SetName("redis-defaults")
			u.SetNamespace(ns)
			Expect(unstructured.SetNestedField(u.Object, "redis.example.com", "spec", "host")).To(Succeed())

			Expect(k8sClient.Create(ctx, u)).To(Succeed())

			fetched := &configv1.RedisConfig{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: "redis-defaults", Namespace: ns}, fetched)).To(Succeed())

			Expect(fetched.Spec.Port).To(Equal(uint16(6379)), "port should default to 6379")
			Expect(fetched.Spec.AvailableDatabaseCount).To(Equal(uint32(16)), "availableDatabaseCount should default to 16")
			Expect(fetched.Spec.IsTlsEnabled).NotTo(BeNil(), "isTlsEnabled should not be nil")
			Expect(*fetched.Spec.IsTlsEnabled).To(BeFalse(), "isTlsEnabled should default to false")
			Expect(fetched.Spec.VerifyTlsServerCertificate).NotTo(BeNil(), "verifyTlsServerCertificate should not be nil")
			Expect(*fetched.Spec.VerifyTlsServerCertificate).To(BeTrue(), "verifyTlsServerCertificate should default to true")
		})
	})
})
