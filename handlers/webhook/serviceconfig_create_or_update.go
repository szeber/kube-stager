package webhook

import (
	"context"
	"errors"
	"fmt"
	configv1 "github.com/szeber/kube-stager/api/config/v1"
	sitev1 "github.com/szeber/kube-stager/api/site/v1"
	"github.com/szeber/kube-stager/handlers/template"
	"github.com/szeber/kube-stager/helpers"
	errorshelpers "github.com/szeber/kube-stager/helpers/errors"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type ServiceConfigCreateOrUpdateHandler struct {
	Client  client.Client
	decoder *admission.Decoder
}

func (r *ServiceConfigCreateOrUpdateHandler) Handle(ctx context.Context, req admission.Request) admission.Response {
	logger := log.FromContext(ctx)
	config := &configv1.ServiceConfig{}
	var err error

	if err = r.decoder.Decode(req, config); nil != err {
		return admission.Errored(http.StatusBadRequest, err)
	}

	templateHandler := template.NewSite(sitev1.GetDummySite(config.Name, config.Namespace), *config)

	if err := template.LoadServiceConfigs(&templateHandler, ctx, r.Client); nil != err {
		logger.Error(err, "Failed to load the service configs for the site")
		return admission.Errored(http.StatusInternalServerError, err)
	}

	if err := template.LoadConfigs(&templateHandler, ctx, r.Client); nil != err {
		logger.Error(err, "Failed to load the database configs for the site")
		return admission.Errored(http.StatusInternalServerError, err)
	}

	templateHandler.SetServiceConfig(config.Name, *config)

	logger.Info("Validating if the shortname is unique")
	configList := &configv1.ServiceConfigList{}
	if err = r.Client.List(
		ctx,
		configList,
		client.InNamespace(config.Namespace),
		client.MatchingFields{".spec.shortName": config.Spec.ShortName},
	); nil != err {
		logger.Error(err, "Failed to list the service configs")
		return admission.Errored(http.StatusInternalServerError, err)
	}

	if len(configList.Items) > 1 || (len(configList.Items) == 1 && configList.Items[0].Name != config.Name) {
		return admission.Errored(
			http.StatusUnprocessableEntity,
			errors.New(
				fmt.Sprintf(
					"The short name %s is not unique in the namespace %s",
					config.Spec.ShortName,
					config.Namespace,
				),
			),
		)
	}

	if "" != config.Spec.DefaultMongoEnvironment {
		logger.Info("Validating default mongo environment: " + config.Spec.DefaultMongoEnvironment)
		if _, ok := templateHandler.GetMongo()[config.Spec.DefaultMongoEnvironment]; !ok {
			return admission.Denied("Invalid mongo environment: " + config.Spec.DefaultMongoEnvironment)
		}
	}

	if "" != config.Spec.DefaultMysqlEnvironment {
		logger.Info("Validating default mysql environment: " + config.Spec.DefaultMysqlEnvironment)
		if _, ok := templateHandler.GetMongo()[config.Spec.DefaultMysqlEnvironment]; !ok {
			return admission.Denied("Invalid mysql environment: " + config.Spec.DefaultMysqlEnvironment)
		}
	}

	if "" != config.Spec.DefaultRedisEnvironment {
		logger.Info("Validating default redis environment: " + config.Spec.DefaultRedisEnvironment)
		if _, ok := templateHandler.GetMongo()[config.Spec.DefaultRedisEnvironment]; !ok {
			return admission.Denied("Invalid redis environment: " + config.Spec.DefaultRedisEnvironment)
		}
	}

	logger.Info("Validating templates")
	err = r.validateTemplates(*config, &templateHandler)
	if errorshelpers.IsControllerError(err) {
		return admission.Denied(err.Error())
	} else if nil != err {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.Allowed("")
}

func (r *ServiceConfigCreateOrUpdateHandler) validateTemplates(
	config configv1.ServiceConfig,
	templateHandler *template.SiteTemplateHandler,
) error {
	spec := config.Spec

	if nil != spec.DbInitPodSpec {
		if _, err := helpers.ReplaceTemplateVariablesInPodSpec(*spec.DbInitPodSpec, templateHandler); nil != err {
			return err
		}
	}
	if nil != spec.MigrationJobPodSpec {
		if _, err := helpers.ReplaceTemplateVariablesInPodSpec(*spec.MigrationJobPodSpec, templateHandler); nil != err {
			return err
		}
	}
	if nil != spec.BackupPodSpec {
		if _, err := helpers.ReplaceTemplateVariablesInPodSpec(*spec.BackupPodSpec, templateHandler); nil != err {
			return err
		}
	}
	if _, err := helpers.ReplaceTemplateVariablesInPodSpec(spec.DeploymentPodSpec, templateHandler); nil != err {
		return err
	}
	if nil != spec.ServiceSpec {
		if _, err := helpers.ReplaceTemplateVariablesInServiceSpec(*spec.ServiceSpec, templateHandler); nil != err {
			return err
		}
	}
	if nil != spec.IngressSpec {
		ingressTemplateValues := make(map[string]string)
		if nil != spec.ServiceSpec {
			ingressTemplateValues["ingress.serviceName"] = "dummy"
		}
		if _, err := helpers.ReplaceTemplateVariablesInIngressSpec(
			*spec.IngressSpec,
			templateHandler,
			&helpers.StringMapTemplateValueGetter{StringMap: ingressTemplateValues},
		); nil != err {
			return err
		}
	}
	for name, v := range spec.ConfigMaps {
		if _, err := helpers.ReplaceTemplateVariablesInStringMap(v, name+" configmap", templateHandler); nil != err {
			return err
		}
	}

	return nil
}

func (r *ServiceConfigCreateOrUpdateHandler) InjectDecoder(d *admission.Decoder) error {
	r.decoder = d
	return nil
}
