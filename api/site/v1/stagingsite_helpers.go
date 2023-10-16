package v1

import (
	configv1 "github.com/szeber/kube-stager/api/config/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

func (r *StagingSite) GetServiceStatus(serviceName string) *StagingSiteServiceStatus {
	if site, ok := r.Status.Services[serviceName]; ok {
		return &site
	}

	return nil
}

func (r *StagingSite) isTimeIntervalEmpty(i TimeInterval) bool {
	return false == i.Never && 0 == i.Days && 0 == i.Hours && 0 == i.Minutes
}

func (r StagingSite) GetMongoConfigForService(serviceConfig configv1.ServiceConfig) string {
	if "" != r.Spec.Services[serviceConfig.Name].MongoEnvironment {
		return r.Spec.Services[serviceConfig.Name].MongoEnvironment
	}
	return serviceConfig.Spec.DefaultMongoEnvironment
}

func (r StagingSite) GetMysqlConfigForService(serviceConfig configv1.ServiceConfig) string {
	if "" != r.Spec.Services[serviceConfig.Name].MysqlEnvironment {
		return r.Spec.Services[serviceConfig.Name].MysqlEnvironment
	}
	return serviceConfig.Spec.DefaultMysqlEnvironment
}

func (r StagingSite) GetRedisConfigForService(serviceConfig configv1.ServiceConfig) string {
	if "" != r.Spec.Services[serviceConfig.Name].RedisEnvironment {
		return r.Spec.Services[serviceConfig.Name].RedisEnvironment
	}
	return serviceConfig.Spec.DefaultRedisEnvironment
}

func (r TimeInterval) ToDuration() time.Duration {
	return time.Minute*time.Duration(r.Minutes) + time.Hour*time.Duration(r.Hours) + time.Hour*24*time.Duration(r.Days)
}

func GetDummySite(serviceName string, namespace string) StagingSite {
	return StagingSite{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dummy",
			Namespace: namespace,
		},
		Spec: StagingSiteSpec{
			DomainPrefix:       "dummy",
			DbName:             "dummy",
			Username:           "dummy",
			Password:           "dummy",
			Enabled:            false,
			DisableAfter:       TimeInterval{Never: true},
			DeleteAfter:        TimeInterval{Never: true},
			BackupBeforeDelete: false,
			Services: map[string]StagingSiteService{
				serviceName: {
					ImageTag:                    "latest",
					Replicas:                    1,
					IncludeInBackups:            false,
					DbInitSourceEnvironmentName: "master",
				},
			},
			IncludeAllServices: false,
		},
	}
}
