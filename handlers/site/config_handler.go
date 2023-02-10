package site

import (
	"context"
	api "github.com/szeber/kube-stager/apis"
	configv1 "github.com/szeber/kube-stager/apis/config/v1"
	sitev1 "github.com/szeber/kube-stager/apis/site/v1"
	"github.com/szeber/kube-stager/handlers/template"
	"github.com/szeber/kube-stager/helpers"
	"github.com/szeber/kube-stager/helpers/labels"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type ConfigHandler struct {
	Reader client.Reader
	Writer client.Writer
	Scheme *runtime.Scheme
}

func (r ConfigHandler) EnsureConfigsAreUpToDate(site *sitev1.StagingSite, ctx context.Context) (bool, error) {
	previousComplete := site.Status.ConfigsAreCreated
	isComplete := true

	if complete, err := r.ensureConfigmapsAreUpToDate(site, ctx); nil != err {
		return false, err
	} else {
		isComplete = isComplete && complete
	}

	site.Status.ConfigsAreCreated = isComplete

	return previousComplete != isComplete, nil
}

func (r ConfigHandler) ensureConfigmapsAreUpToDate(site *sitev1.StagingSite, ctx context.Context) (bool, error) {
	logger := log.FromContext(ctx)

	logger.V(0).Info("Retrieving dotenv list")

	var list corev1.ConfigMapList
	err := r.Reader.List(
		ctx,
		&list,
		client.InNamespace(site.Namespace),
		client.MatchingLabels{
			labels.Site: site.Name,
		},
	)
	if nil != err {
		return false, err
	}

	dotenvsToCreate := make(map[string]corev1.ConfigMap)
	dotenvsToUpdate := make(map[string]corev1.ConfigMap)
	dotenvsToDelete := make(map[string]corev1.ConfigMap)

	for name := range site.Spec.Services {
		config := &configv1.ServiceConfig{}
		if err := r.Reader.Get(ctx, client.ObjectKey{Namespace: site.Namespace, Name: name}, config); nil != err {
			return false, err
		}

		for key, values := range config.Spec.ConfigMaps {
			dotenv, err := r.createConfigMap(ctx, site, config, key, values)
			if nil != err {
				return false, err
			}
			dotenvsToCreate[r.makeConfigmapKey(config.Name, key)] = dotenv
		}
	}

	for _, configMap := range list.Items {
		configmapKey := r.makeConfigmapKey(
			configMap.ObjectMeta.Labels[labels.Service],
			configMap.ObjectMeta.Labels[labels.Type],
		)

		if dotenvToCreate, ok := dotenvsToCreate[configmapKey]; ok {
			if !reflect.DeepEqual(dotenvToCreate.Data, configMap.Data) {
				configMap.Data = dotenvToCreate.Data
				dotenvsToUpdate[configmapKey] = configMap
			}
			delete(dotenvsToCreate, configmapKey)
		} else {
			dotenvsToDelete[configmapKey] = configMap
		}
	}

	for serviceName, configMap := range dotenvsToDelete {
		configmapType := configMap.ObjectMeta.Labels[labels.Type]
		logger.V(1).Info("Deleting " + configmapType + " configmap for service " + serviceName)
		if err = r.Writer.Delete(ctx, &configMap); nil != err {
			return false, err
		}
	}
	for serviceName, configMap := range dotenvsToUpdate {
		configmapType := configMap.ObjectMeta.Labels[labels.Type]
		logger.V(1).Info("Updating " + configmapType + " configmap for service " + serviceName)
		if err = r.Writer.Update(ctx, &configMap); nil != err {
			return false, err
		}
	}
	for serviceName, configMap := range dotenvsToCreate {
		configmapType := configMap.ObjectMeta.Labels[labels.Type]
		logger.V(1).Info("Creating " + configmapType + " configmap for service " + serviceName)
		if err = r.Writer.Create(ctx, &configMap); nil != err {
			return false, err
		}
	}

	logger.V(0).Info("Configmaps created")

	return true, nil
}

func (r ConfigHandler) makeConfigmapKey(siteName string, configmapName string) string {
	return siteName + "." + configmapName
}

func (r ConfigHandler) createConfigMap(
	ctx context.Context,
	site *sitev1.StagingSite,
	config *configv1.ServiceConfig,
	typeName string,
	data map[string]string,
) (corev1.ConfigMap, error) {
	templateHandler := template.NewSite(*site, *config)
	err := template.LoadConfigs(
		&templateHandler,
		ctx,
		r.Reader,
		site.Spec.Services[config.Name].MysqlEnvironment,
		site.Spec.Services[config.Name].MongoEnvironment,
		site.Spec.Services[config.Name].RedisEnvironment,
	)
	if nil != err {
		return corev1.ConfigMap{}, err
	}
	replacedData, err := helpers.ReplaceTemplateVariablesInStringMap(data, "configmap data", &templateHandler)
	if nil != err {
		return corev1.ConfigMap{}, err
	}

	configMap := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      api.MakeConfigmapName(site, config, typeName),
			Namespace: site.Namespace,
			Labels: map[string]string{
				labels.Site:    site.Name,
				labels.Service: config.Name,
				labels.Type:    typeName,
			},
		},
		Data: replacedData,
	}

	err = ctrl.SetControllerReference(site, &configMap, r.Scheme)
	return configMap, err
}
