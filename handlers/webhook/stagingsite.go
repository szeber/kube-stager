package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	sitev1 "github.com/szeber/kube-stager/apis/site/v1"
	"github.com/szeber/kube-stager/helpers"
	"github.com/szeber/kube-stager/helpers/kubernetes"
	"github.com/szeber/kube-stager/helpers/labels"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"strings"
)

type StagingsiteHandler struct {
	Client  client.Client
	decoder admission.Decoder
}

func (r *StagingsiteHandler) Handle(ctx context.Context, req admission.Request) admission.Response {
	logger := log.FromContext(ctx)
	site := &sitev1.StagingSite{}
	var err error

	if err = r.decoder.Decode(req, site); nil != err {
		return admission.Errored(http.StatusBadRequest, err)
	}

	serviceConfigs, err := kubernetes.GetServiceConfigsInNamespace(site.Namespace, r.Client, ctx)
	if nil != err {
		logger.Error(err, "Failed to list the service configs")
		return admission.Errored(http.StatusInternalServerError, err)
	}
	mysqlEnvironments, err := kubernetes.GetMysqlEnvironmentsInNamespace(site.Namespace, r.Client, ctx)
	if nil != err {
		logger.Error(err, "Failed to list the service configs")
		return admission.Errored(http.StatusInternalServerError, err)
	}
	mongoEnvironments, err := kubernetes.GetMongoEnvironmentsInNamespace(site.Namespace, r.Client, ctx)
	if nil != err {
		logger.Error(err, "Failed to list the service configs")
		return admission.Errored(http.StatusInternalServerError, err)
	}
	redisEnvironments, err := kubernetes.GetRedisEnvironmentsInNamespace(site.Namespace, r.Client, ctx)
	if nil != err {
		logger.Error(err, "Failed to list the service configs")
		return admission.Errored(http.StatusInternalServerError, err)
	}

	if site.Spec.IncludeAllServices {
		logger.Info("Adding all services to the site")
		if 0 == len(site.Spec.Services) {
			site.Spec.Services = make(map[string]sitev1.StagingSiteService, len(serviceConfigs))
		}
		for _, serviceConfig := range serviceConfigs {
			serviceSpec := site.Spec.Services[serviceConfig.Name]
			if 0 == len(serviceSpec.CustomTemplateValues) {
				serviceSpec.CustomTemplateValues = make(map[string]string)
			}
			site.Spec.Services[serviceConfig.Name] = serviceSpec
		}
	}

	if 0 == len(site.Spec.Services) {
		return admission.Denied("There are no services defined in the site")
	}

	usedMongoEnvironmentNames := make(map[string]bool)
	usedMysqlEnvironmentNames := make(map[string]bool)
	usedRedisEnvironmentNames := make(map[string]bool)
	var serviceNames []string

	for name, serviceSpec := range site.Spec.Services {
		serviceNames = append(serviceNames, name)
		config, ok := serviceConfigs[name]
		if !ok {
			return admission.Denied("The service config '" + name + "' doesn't exist")
		}
		if "" == serviceSpec.ImageTag {
			serviceSpec.ImageTag = "latest"
		}
		if "" == serviceSpec.DbInitSourceEnvironmentName {
			serviceSpec.DbInitSourceEnvironmentName = "master"
		}
		if "" == serviceSpec.MongoEnvironment {
			serviceSpec.MongoEnvironment = config.Spec.DefaultMongoEnvironment
		}
		if "" == serviceSpec.MysqlEnvironment {
			serviceSpec.MysqlEnvironment = config.Spec.DefaultMysqlEnvironment
		}
		if "" == serviceSpec.RedisEnvironment {
			serviceSpec.RedisEnvironment = config.Spec.DefaultRedisEnvironment
		}
		if "" != serviceSpec.MongoEnvironment {
			if mongoEnvironments[serviceSpec.MongoEnvironment].Name == "" {
				return admission.Denied(
					fmt.Sprintf(
						"Invalid mongo environment '%s' in service '%s'",
						serviceSpec.MongoEnvironment,
						name,
					),
				)
			} else {
				usedMongoEnvironmentNames[serviceSpec.MongoEnvironment] = true
			}
		}
		if "" != serviceSpec.MysqlEnvironment {
			if mysqlEnvironments[serviceSpec.MysqlEnvironment].Name == "" {
				return admission.Denied(
					fmt.Sprintf(
						"Invalid mysql environment '%s' in service '%s'",
						serviceSpec.MysqlEnvironment,
						name,
					),
				)
			} else {
				usedMysqlEnvironmentNames[serviceSpec.MysqlEnvironment] = true
			}
		}
		if "" != serviceSpec.RedisEnvironment {
			if redisEnvironments[serviceSpec.RedisEnvironment].Name == "" {
				return admission.Denied(
					fmt.Sprintf(
						"Invalid redis environment '%s' in service '%s'",
						serviceSpec.RedisEnvironment,
						name,
					),
				)
			} else {
				usedRedisEnvironmentNames[serviceSpec.RedisEnvironment] = true
			}
		}
		site.Spec.Services[name] = serviceSpec
	}

	r.updatePrefixedLabels(
		site,
		labels.MongoEnvironmentsPrefix,
		helpers.GetKeysFromStringBoolMap(usedMongoEnvironmentNames),
	)
	r.updatePrefixedLabels(
		site,
		labels.MysqlEnvironmentsPrefix,
		helpers.GetKeysFromStringBoolMap(usedMysqlEnvironmentNames),
	)
	r.updatePrefixedLabels(
		site,
		labels.RedisEnvironmentsPrefix,
		helpers.GetKeysFromStringBoolMap(usedRedisEnvironmentNames),
	)
	r.updatePrefixedLabels(site, labels.ServicesPrefix, serviceNames)

	marshaledSite, err := json.Marshal(site)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledSite)
}

func (r *StagingsiteHandler) updatePrefixedLabels(site *sitev1.StagingSite, prefix string, values []string) {
	siteLabels := site.Labels
	if 0 == len(siteLabels) {
		siteLabels = make(map[string]string, len(values))
	}
	for k := range site.Labels {
		if strings.HasPrefix(k, prefix) {
			delete(siteLabels, k)
		}
	}
	for _, v := range values {
		if "" == v {
			continue
		}
		siteLabels[prefix+v] = "true"
	}
	site.Labels = siteLabels
}

func (r *StagingsiteHandler) InjectDecoder(d admission.Decoder) error {
	r.decoder = d
	return nil
}
