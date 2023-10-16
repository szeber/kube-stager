package api

import (
	"fmt"
	configv1 "github.com/szeber/kube-stager/api/config/v1"
	sitev1 "github.com/szeber/kube-stager/api/site/v1"
	"github.com/szeber/kube-stager/helpers"
	"strings"
)

func MakeDatabaseName(site *sitev1.StagingSite, service *configv1.ServiceConfig) string {
	return strings.Replace(
		helpers.ShortenHumanReadableValue(
			fmt.Sprintf(
				"%s_%s",
				helpers.SanitiseDbValue(site.Spec.DbName),
				helpers.SanitiseDbValue(service.Spec.ShortName),
			),
			63,
		),
		"-",
		"_",
		-1,
	)
}

func MakeUsername(site *sitev1.StagingSite, service *configv1.ServiceConfig) string {
	return helpers.SanitiseAndShortenDbValue(fmt.Sprintf("%s_%s", site.Spec.Username, service.Spec.ShortName), 16)
}

func MakeConfigmapName(site *sitev1.StagingSite, service *configv1.ServiceConfig, typeName string) string {
	return helpers.MakeObjectName(site.Name, service.Spec.ShortName, string(typeName))
}

func MakeServiceName(site *sitev1.StagingSite, service *configv1.ServiceConfig) string {
	return helpers.MakeObjectName(site.Name, service.Spec.ShortName)
}

func MakeIngressName(site *sitev1.StagingSite, service *configv1.ServiceConfig) string {
	return helpers.MakeObjectName(site.Name, service.Spec.ShortName)
}

func MakeDeploymentName(site *sitev1.StagingSite, service *configv1.ServiceConfig) string {
	return helpers.MakeObjectName(site.Name, service.Spec.ShortName)
}

func MakeServiceUrl(site *sitev1.StagingSite, serviceName string) string {
	return fmt.Sprintf("%s.%s.svc.cluster.local", helpers.MakeObjectName(site.Name, serviceName), site.Namespace)
}
