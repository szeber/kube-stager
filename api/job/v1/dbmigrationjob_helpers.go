package v1

import (
	"errors"
	configv1 "github.com/szeber/kube-stager/api/config/v1"
	sitev1 "github.com/szeber/kube-stager/api/site/v1"
	"github.com/szeber/kube-stager/helpers"
	"github.com/szeber/kube-stager/helpers/labels"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *DbMigrationJob) PopulateFomSite(site *sitev1.StagingSite, config *configv1.ServiceConfig) error {
	if nil == config.Spec.MigrationJobPodSpec {
		return errors.New("no migration pod spec specified in the service config")
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
	r.Spec = DbMigrationJobSpec{
		SiteName:        site.Name,
		ServiceName:     config.ObjectMeta.Name,
		ImageTag:        site.Spec.Services[config.Name].ImageTag,
		DeadlineSeconds: 600,
	}
	r.Name = r.ObjectMeta.Name
	r.Namespace = r.ObjectMeta.Namespace
	r.Labels = r.ObjectMeta.Labels
	r.Annotations = r.ObjectMeta.Annotations

	return nil
}

func (r *DbMigrationJob) Matches(job *DbMigrationJob) bool {
	return r.Spec.SiteName == job.Spec.SiteName &&
		r.Spec.ServiceName == job.Spec.ServiceName &&
		r.Spec.ImageTag == job.Spec.ImageTag
}

func (r *DbMigrationJob) UpdateFrom(job *DbMigrationJob) {
	r.Spec = job.Spec
}
