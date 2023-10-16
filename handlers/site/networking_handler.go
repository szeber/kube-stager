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
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type NetworkingHandler struct {
	Reader client.Reader
	Writer client.Writer
	Scheme *runtime.Scheme
}

func (r NetworkingHandler) EnsureNetworkingObjectsAreUpToDate(site *sitev1.StagingSite, ctx context.Context) (
	bool,
	error,
) {
	previousComplete := site.Status.NetworkingObjectsAreCreated
	isComplete := true

	if complete, err := r.ensureServicesAreUpToDate(site, ctx); nil != err {
		return false, err
	} else {
		isComplete = isComplete && complete
	}

	if complete, err := r.ensureIngressesAreUpToDate(site, ctx); nil != err {
		return false, err
	} else {
		isComplete = isComplete && complete
	}

	site.Status.NetworkingObjectsAreCreated = isComplete

	return previousComplete != isComplete, nil
}

func (r NetworkingHandler) ensureServicesAreUpToDate(site *sitev1.StagingSite, ctx context.Context) (bool, error) {
	logger := log.FromContext(ctx)

	logger.V(0).Info("Retrieving service list")

	var list corev1.ServiceList
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

	if !site.Status.Enabled {
		for _, service := range list.Items {
			if err := r.Writer.Delete(ctx, &service); nil != err {
				return false, err
			}
		}
		return true, err
	}

	servicesToCreate := make(map[string]corev1.Service)
	servicesToDelete := make(map[string]corev1.Service)

	for name := range site.Spec.Services {
		config := &configv1.ServiceConfig{}
		if err := r.Reader.Get(ctx, client.ObjectKey{Namespace: site.Namespace, Name: name}, config); nil != err {
			return false, err
		}

		if nil == config.Spec.ServiceSpec {
			continue
		}

		servicesToCreate[name], err = r.createService(ctx, site, config)
		if nil != err {
			return false, err
		}
	}

	for _, service := range list.Items {
		serviceName := service.ObjectMeta.Labels[labels.Service]

		if _, ok := servicesToCreate[serviceName]; ok {
			delete(servicesToCreate, serviceName)
		} else {
			servicesToDelete[serviceName] = service
		}
	}

	for serviceName, database := range servicesToDelete {
		logger.V(1).Info("Deleting service for service " + serviceName)
		if err = r.Writer.Delete(ctx, &database); nil != err {
			return false, err
		}
	}
	for serviceName, database := range servicesToCreate {
		logger.V(1).Info("Creating service for service " + serviceName)
		if err = r.Writer.Create(ctx, &database); nil != err {
			return false, err
		}
	}

	logger.V(0).Info("Services created")

	return true, nil
}

func (r NetworkingHandler) ensureIngressesAreUpToDate(site *sitev1.StagingSite, ctx context.Context) (bool, error) {
	logger := log.FromContext(ctx)

	logger.V(0).Info("Retrieving ingress list")

	var list networkingv1.IngressList
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

	if !site.Status.Enabled {
		for _, service := range list.Items {
			if err := r.Writer.Delete(ctx, &service); nil != err {
				return false, err
			}
		}
		return true, err
	}

	ingressesToCreate := make(map[string]networkingv1.Ingress)
	ingressesToDelete := make(map[string]networkingv1.Ingress)

	for name := range site.Spec.Services {
		config := &configv1.ServiceConfig{}
		if err := r.Reader.Get(ctx, client.ObjectKey{Namespace: site.Namespace, Name: name}, config); nil != err {
			return false, err
		}

		if nil == config.Spec.IngressSpec {
			continue
		}

		ingress, err := r.createIngress(ctx, site, config)
		if nil != err {
			return false, err
		}
		ingressesToCreate[name] = ingress
	}

	for _, ingress := range list.Items {
		serviceName := ingress.ObjectMeta.Labels[labels.Service]

		if _, ok := ingressesToCreate[serviceName]; ok {
			delete(ingressesToCreate, serviceName)
		} else {
			ingressesToDelete[serviceName] = ingress
		}
	}

	for ingressName, database := range ingressesToDelete {
		logger.V(1).Info("Deleting ingress for service " + ingressName)
		if err = r.Writer.Delete(ctx, &database); nil != err {
			return false, err
		}
	}
	for ingressName, database := range ingressesToCreate {
		logger.V(1).Info("Creating ingress for service " + ingressName)
		if err = r.Writer.Create(ctx, &database); nil != err {
			return false, err
		}
	}

	logger.V(0).Info("Ingresses created")

	return true, nil
}

func (r NetworkingHandler) createService(
	ctx context.Context,
	site *sitev1.StagingSite,
	config *configv1.ServiceConfig,
) (corev1.Service, error) {
	siteTemplateHandler := template.NewSite(*site, *config)
	err := template.LoadConfigs(&siteTemplateHandler, ctx, r.Reader)
	if nil != err {
		return corev1.Service{}, err
	}
	replacedSpec, err := helpers.ReplaceTemplateVariablesInServiceSpec(*config.Spec.ServiceSpec, &siteTemplateHandler)
	if nil != err {
		return corev1.Service{}, err
	}
	replacedSpec.Selector = map[string]string{
		labels.Site:    site.Name,
		labels.Service: config.Name,
	}
	service := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      api.MakeServiceName(site, config),
			Namespace: site.Namespace,
			Labels: map[string]string{
				labels.Site:    site.Name,
				labels.Service: config.Name,
			},
		},
		Spec: replacedSpec,
	}

	if err := ctrl.SetControllerReference(site, &service, r.Scheme); nil != err {
		return service, err
	}

	return service, nil
}

func (r NetworkingHandler) createIngress(
	ctx context.Context,
	site *sitev1.StagingSite,
	config *configv1.ServiceConfig,
) (networkingv1.Ingress, error) {
	siteTemplateHandler := template.NewSite(*site, *config)
	err := template.LoadConfigs(&siteTemplateHandler, ctx, r.Reader)
	if nil != err {
		return networkingv1.Ingress{}, err
	}
	boolTrue := true
	replacedSpec, err := helpers.ReplaceTemplateVariablesInIngressSpec(
		*config.Spec.IngressSpec,
		&siteTemplateHandler,
		helpers.StringMapTemplateValueGetter{
			StringMap: map[string]string{
				"ingress.serviceName": api.MakeServiceName(
					site,
					config,
				),
			},
		},
	)
	if nil != err {
		return networkingv1.Ingress{}, err
	}
	if 0 == len(config.Spec.IngressAnnotations) {
		config.Spec.IngressAnnotations = make(map[string]string)
	}
	ingress := networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      api.MakeIngressName(site, config),
			Namespace: site.Namespace,
			Labels: map[string]string{
				labels.Site:    site.Name,
				labels.Service: config.Name,
			},
			Annotations: config.Spec.IngressAnnotations,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         site.APIVersion,
					Kind:               site.Kind,
					Name:               site.Name,
					UID:                site.UID,
					Controller:         &boolTrue,
					BlockOwnerDeletion: &boolTrue,
				},
			},
		},
		Spec: replacedSpec,
	}

	if err := ctrl.SetControllerReference(site, &ingress, r.Scheme); nil != err {
		return ingress, err
	}

	return ingress, nil
}
