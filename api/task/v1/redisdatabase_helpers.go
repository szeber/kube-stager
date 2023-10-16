package v1

import (
	"errors"
	configv1 "github.com/szeber/kube-stager/api/config/v1"
	sitev1 "github.com/szeber/kube-stager/api/site/v1"
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
	if nil == config {
		return errors.New("No service config provided")
	}

	r.ObjectMeta = metav1.ObjectMeta{
		Name:      helpers.ShortenHumanReadableValue(site.ObjectMeta.Name, 50) + "-" + config.Spec.ShortName,
		Namespace: site.ObjectMeta.Namespace,
		Labels: map[string]string{
			labels.Site:             site.ObjectMeta.Name,
			labels.Service:          config.ObjectMeta.Name,
			labels.RedisEnvironment: environmentName,
		},
		Annotations: map[string]string{},
	}
	r.Spec = RedisDatabaseSpec{
		EnvironmentConfig: EnvironmentConfig{
			ServiceName: config.ObjectMeta.Name,
			SiteName:    site.ObjectMeta.Name,
			Environment: environmentName,
		},
	}

	return nil
}

func (r *RedisDatabase) Matches(other RedisDatabase) bool {
	return reflect.DeepEqual(r.Spec.EnvironmentConfig, other.Spec.EnvironmentConfig) &&
		r.ObjectMeta.Name == other.ObjectMeta.Name &&
		r.ObjectMeta.Namespace == other.ObjectMeta.Namespace &&
		reflect.DeepEqual(r.ObjectMeta.Labels, other.ObjectMeta.Labels)
}

func (r *RedisDatabase) UpdateFromExpected(expected RedisDatabase) {
	r.Spec = expected.Spec
	r.ObjectMeta.Name = expected.ObjectMeta.Name
	r.ObjectMeta.Namespace = expected.ObjectMeta.Namespace
	r.ObjectMeta.Labels = expected.ObjectMeta.Labels
}
