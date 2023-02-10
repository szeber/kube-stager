package v1

import (
	"errors"
	"fmt"
	api "github.com/szeber/kube-stager/apis"
	configv1 "github.com/szeber/kube-stager/apis/config/v1"
	sitev1 "github.com/szeber/kube-stager/apis/site/v1"
	"github.com/szeber/kube-stager/helpers"
	"github.com/szeber/kube-stager/helpers/labels"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *DbInitJob) PopulateFomSite(
	site *sitev1.StagingSite,
	config *configv1.ServiceConfig,
	mysqlEnvironment string,
	mongoEnvironment string,
) error {
	siteService, ok := site.Spec.Services[config.Name]
	if !ok {
		return errors.New(fmt.Sprintf("Service %s is not in the site spec", config.Name))
	}

	r.ObjectMeta = metav1.ObjectMeta{
		Name:      helpers.ShortenHumanReadableValue(site.ObjectMeta.Name, 50) + "-" + config.Spec.ShortName,
		Namespace: site.ObjectMeta.Namespace,
		Labels: map[string]string{
			labels.Site:    site.ObjectMeta.Name,
			labels.Service: config.ObjectMeta.Name,
		},
		Annotations: map[string]string{},
	}
	r.Spec = DbInitJobSpec{
		SiteName:         site.Name,
		ServiceName:      config.ObjectMeta.Name,
		MysqlEnvironment: mysqlEnvironment,
		MongoEnvironment: mongoEnvironment,
		DbInitSource:     siteService.DbInitSourceEnvironmentName,
		DatabaseName:     api.MakeDatabaseName(site, config),
		Username:         api.MakeUsername(site, config),
		Password:         site.Spec.Password,
		DeadlineSeconds:  600,
	}
	r.Name = r.ObjectMeta.Name
	r.Namespace = r.ObjectMeta.Namespace
	r.Labels = r.ObjectMeta.Labels
	r.Annotations = r.ObjectMeta.Annotations

	return nil
}
