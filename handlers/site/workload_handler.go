package site

import (
	"context"
	api "github.com/szeber/kube-stager/apis"
	configv1 "github.com/szeber/kube-stager/apis/config/v1"
	sitev1 "github.com/szeber/kube-stager/apis/site/v1"
	"github.com/szeber/kube-stager/handlers/template"
	"github.com/szeber/kube-stager/helpers"
	"github.com/szeber/kube-stager/helpers/labels"
	"github.com/szeber/kube-stager/helpers/pod"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type WorkloadHandler struct {
	Reader client.Reader
	Writer client.Writer
	Scheme *runtime.Scheme
}

type deploymentUpdate struct {
	existing appsv1.Deployment
	patch    client.Patch
}

func (r WorkloadHandler) EnsureWorkloadObjectsAreUpToDate(site *sitev1.StagingSite, ctx context.Context) (bool, error) {
	previousComplete := site.Status.WorkloadsAreCreated
	previousHealth := site.Status.WorkloadHealth
	isComplete := true

	if complete, err := r.ensureDeploymentsAreUpToDate(site, ctx); nil != err {
		return false, err
	} else {
		isComplete = isComplete && complete
	}

	site.Status.WorkloadsAreCreated = isComplete

	return previousComplete != isComplete || site.Status.WorkloadHealth != previousHealth, nil
}

func (r WorkloadHandler) ensureDeploymentsAreUpToDate(site *sitev1.StagingSite, ctx context.Context) (bool, error) {
	logger := log.FromContext(ctx)

	logger.V(0).Info("Retrieving deployment list")

	var list appsv1.DeploymentList
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

	deploymentsToCreate := make(map[string]appsv1.Deployment)
	deploymentsToUpdate := make(map[string]deploymentUpdate)
	deploymentsToDelete := make(map[string]appsv1.Deployment)

	for name := range site.Spec.Services {
		config := &configv1.ServiceConfig{}
		if err := r.Reader.Get(ctx, client.ObjectKey{Namespace: site.Namespace, Name: name}, config); nil != err {
			return false, err
		}

		if deployment, err := r.createDeployment(site, config, ctx); nil != err {
			return false, err
		} else {
			deploymentsToCreate[name] = *deployment
		}
	}

	isEverythingHealthy := true
	for _, existingDeployment := range list.Items {
		serviceName := existingDeployment.ObjectMeta.Labels[labels.Service]

		if _, ok := deploymentsToCreate[serviceName]; ok {
			patch := client.MergeFrom(existingDeployment.DeepCopy())
			r.updateDeploymentFromOther(&existingDeployment, deploymentsToCreate[serviceName])
			deploymentsToUpdate[serviceName] = deploymentUpdate{
				existing: existingDeployment,
				patch:    patch,
			}

			serviceStatus := site.Status.Services[serviceName]
			serviceStatus.DeploymentStatus = *existingDeployment.Status.DeepCopy()
			site.Status.Services[serviceName] = serviceStatus

			isEverythingHealthy = isEverythingHealthy &&
				(*existingDeployment.Spec.Replicas == existingDeployment.Status.ReadyReplicas)

			delete(deploymentsToCreate, serviceName)
		} else {
			deploymentsToDelete[serviceName] = existingDeployment
		}
	}

	if isEverythingHealthy {
		site.Status.WorkloadHealth = sitev1.WorkloadHealthHealthy
	} else {
		site.Status.WorkloadHealth = sitev1.WorkloadHealthUnhealthy
	}

	for serviceName, deployment := range deploymentsToDelete {
		logger.V(1).Info("Deleting deployment for service " + serviceName)
		if err = r.Writer.Delete(ctx, &deployment); nil != err {
			return false, err
		}
	}
	for serviceName, deployment := range deploymentsToUpdate {
		logger.V(1).Info("Updating deployment for service", "serviceName", serviceName)
		if err := r.Writer.Patch(
			ctx,
			&deployment.existing,
			deployment.patch,
		); nil != err {
			return false, err
		}
	}
	for serviceName, deployment := range deploymentsToCreate {
		logger.V(1).Info("Creating deployment for service " + serviceName)
		if err = r.Writer.Create(ctx, &deployment); nil != err {
			return false, err
		}
	}

	logger.V(0).Info("Deployments created")

	return true, nil
}

func (r WorkloadHandler) createDeployment(
	site *sitev1.StagingSite,
	serviceConfig *configv1.ServiceConfig,
	ctx context.Context,
) (*appsv1.Deployment, error) {
	templateHandler := template.NewSite(*site, *serviceConfig)
	err := template.LoadConfigs(
		&templateHandler,
		ctx,
		r.Reader,
		site.GetMysqlConfigForService(*serviceConfig),
		site.GetMongoConfigForService(*serviceConfig),
		site.GetRedisConfigForService(*serviceConfig),
	)
	if nil != err {
		return nil, err
	}
	labelsMap := map[string]string{
		labels.Site:    site.Name,
		labels.Service: serviceConfig.Name,
	}

	replicas := site.Spec.Services[serviceConfig.Name].Replicas
	if replicas < 1 {
		replicas = int32(1)
	}

	podSpec, err := helpers.ReplaceTemplateVariablesInPodSpec(serviceConfig.Spec.DeploymentPodSpec, &templateHandler)
	if nil != err {
		return nil, err
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      api.MakeDeploymentName(site, serviceConfig),
			Namespace: site.Namespace,
			Labels:    labelsMap,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: labelsMap},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labelsMap,
				},
				Spec: pod.SetExtraEnvVarsOnPodSpec(
					pod.UpdatePodSpecWithOverrides(podSpec, site, serviceConfig),
					site,
					serviceConfig,
				),
			},
		},
	}

	if err := ctrl.SetControllerReference(site, deployment, r.Scheme); nil != err {
		return nil, err
	}

	return deployment, nil
}

func (r WorkloadHandler) updateDeploymentFromOther(a *appsv1.Deployment, b appsv1.Deployment) {
	a.ObjectMeta.Labels = b.ObjectMeta.Labels
	a.Spec.Template.ObjectMeta.Labels = b.Spec.Template.ObjectMeta.Labels
	a.Spec.Template.Spec = b.Spec.Template.Spec
	a.Spec.Replicas = b.Spec.Replicas
}
