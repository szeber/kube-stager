package v1

import (
	"errors"
	configv1 "github.com/szeber/kube-stager/apis/config/v1"
	sitev1 "github.com/szeber/kube-stager/apis/site/v1"
	"github.com/szeber/kube-stager/helpers"
	"github.com/szeber/kube-stager/helpers/labels"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
)

func (r *RedisDatabase) PopulateFomSite(
	site *sitev1.StagingSite,
	config *configv1.ServiceConfig,
	environmentName string,
) error {
	if config == nil {
		return errors.New("no service config provided")
	}

	r.ObjectMeta = metav1.ObjectMeta{
		Name:      helpers.ShortenHumanReadableValue(site.Name, 50) + "-" + config.Spec.ShortName,
		Namespace: site.Namespace,
		Labels: map[string]string{
			labels.Site:             site.Name,
			labels.Service:          config.Name,
			labels.RedisEnvironment: environmentName,
		},
		Annotations: map[string]string{},
	}
	r.Spec = RedisDatabaseSpec{
		EnvironmentConfig: EnvironmentConfig{
			ServiceName: config.Name,
			SiteName:    site.Name,
			Environment: environmentName,
		},
	}

	return nil
}

func (r *RedisDatabase) Matches(other RedisDatabase) bool {
	return reflect.DeepEqual(r.Spec.EnvironmentConfig, other.Spec.EnvironmentConfig) &&
		r.Name == other.Name &&
		r.Namespace == other.Namespace &&
		reflect.DeepEqual(r.Labels, other.Labels)
}

func (r *RedisDatabase) UpdateFromExpected(expected RedisDatabase) {
	r.Spec = expected.Spec
	r.Name = expected.Name
	r.Namespace = expected.Namespace
	r.Labels = expected.Labels
}
