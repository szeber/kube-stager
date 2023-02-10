package v1

import (
	"errors"
	api "github.com/szeber/kube-stager/apis"
	configv1 "github.com/szeber/kube-stager/apis/config/v1"
	sitev1 "github.com/szeber/kube-stager/apis/site/v1"
	"github.com/szeber/kube-stager/helpers"
	"github.com/szeber/kube-stager/helpers/labels"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
)

func (r *MysqlDatabase) PopulateFomSite(
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
			labels.MysqlEnvironment: environmentName,
		},
		Annotations: map[string]string{},
	}
	r.Spec = MysqlDatabaseSpec{
		EnvironmentConfig: EnvironmentConfig{
			ServiceName: config.ObjectMeta.Name,
			SiteName:    site.ObjectMeta.Name,
			Environment: environmentName,
		},
		DatabaseName: api.MakeDatabaseName(site, config),
		Username:     api.MakeUsername(site, config),
		Password:     site.Spec.Password,
	}
	r.Name = r.ObjectMeta.Name
	r.Namespace = r.ObjectMeta.Namespace
	r.Labels = r.ObjectMeta.Labels
	r.Annotations = r.ObjectMeta.Annotations

	return nil
}

func (r *MysqlDatabase) Matches(other MysqlDatabase) bool {
	return reflect.DeepEqual(r.Spec, other.Spec) &&
		r.ObjectMeta.Name == other.ObjectMeta.Name &&
		r.ObjectMeta.Namespace == other.ObjectMeta.Namespace &&
		reflect.DeepEqual(r.ObjectMeta.Labels, other.ObjectMeta.Labels)
}

func (r *MysqlDatabase) UpdateFromExpected(expected MysqlDatabase) {
	r.Spec = expected.Spec
	r.ObjectMeta.Name = expected.ObjectMeta.Name
	r.ObjectMeta.Namespace = expected.ObjectMeta.Namespace
	r.ObjectMeta.Labels = expected.ObjectMeta.Labels
}
