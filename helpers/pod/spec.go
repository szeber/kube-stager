package pod

import (
	configv1 "github.com/szeber/kube-stager/apis/config/v1"
	sitev1 "github.com/szeber/kube-stager/apis/site/v1"
	corev1 "k8s.io/api/core/v1"
	"sort"
)

func SetExtraEnvVarsOnPodSpec(
	spec corev1.PodSpec,
	site *sitev1.StagingSite,
	serviceConfig *configv1.ServiceConfig,
) corev1.PodSpec {

	extraEnvs := site.Spec.Services[serviceConfig.Name].ExtraEnvs

	if len(extraEnvs) > 0 {
		for i, v := range spec.Containers {
			for name, value := range extraEnvs {
				v.Env = append(
					v.Env, corev1.EnvVar{
						Name:  name,
						Value: value,
					},
				)
			}
			envs := v.Env
			sort.Slice(envs, func(i, j int) bool { return envs[i].Name < envs[j].Name })
			v.Env = envs
			spec.Containers[i] = v
		}
	}

	return spec
}

func UpdatePodSpecWithOverrides(
	spec corev1.PodSpec,
	site *sitev1.StagingSite,
	serviceConfig *configv1.ServiceConfig,
) corev1.PodSpec {
	service := site.Spec.Services[serviceConfig.Name]
	if len(service.ResourceOverrides) == 0 {
		return spec
	}

	for index, container := range spec.Containers {
		if override, ok := service.ResourceOverrides[container.Name]; ok {
			spec.Containers[index].Resources = override
		}
	}

	return spec
}
