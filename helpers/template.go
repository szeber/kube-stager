package helpers

import (
	"fmt"
	"github.com/szeber/kube-stager/helpers/errors"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"regexp"
	"sigs.k8s.io/yaml"
	"strings"
)

type TemplateValueGetter interface {
	GetTemplateValues() map[string]string
}

type StringMapTemplateValueGetter struct {
	StringMap map[string]string
}

func (r StringMapTemplateValueGetter) GetTemplateValues() map[string]string {
	return r.StringMap
}

func ReplaceTemplateVariablesInString(s string, templates ...TemplateValueGetter) string {
	for _, i := range templates {
		for name, value := range i.GetTemplateValues() {
			s = strings.Replace(s, fmt.Sprintf("${%s}", name), value, -1)
		}
	}

	return s
}

func ReplaceTemplateVariablesInStringMap(
	stringMap map[string]string,
	entityType string,
	templates ...TemplateValueGetter,
) (map[string]string, error) {
	for k, s := range stringMap {
		replaced := ReplaceTemplateVariablesInString(s, templates...)
		unresolvedTemplates := GetUnresolvedTemplatesFromString(replaced)
		stringMap[k] = replaced

		if len(unresolvedTemplates) > 0 {
			return stringMap, errors.UnresolvedTemplatesError{
				TemplateVariables: unresolvedTemplates,
				EntityType:        entityType,
			}
		}

	}

	return stringMap, nil
}

func GetUnresolvedTemplatesFromString(s string) []string {
	slice := regexp.MustCompile(`\${[-_a-zA-Z0-9.]+}`).FindAllString(s, -1)

	// Make list unique
	keys := make(map[string]bool)
	var list []string
	for _, entry := range slice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

func ReplaceTemplateVariablesInPodSpec(spec corev1.PodSpec, templates ...TemplateValueGetter) (corev1.PodSpec, error) {
	pod := corev1.Pod{Spec: spec}

	data, err := yaml.Marshal(pod)
	if nil != err {
		return spec, err
	}

	replacedMarshalledSpec := ReplaceTemplateVariablesInString(string(data), templates...)
	unresolvedTemplates := GetUnresolvedTemplatesFromString(replacedMarshalledSpec)

	if len(unresolvedTemplates) > 0 {
		return spec, errors.UnresolvedTemplatesError{TemplateVariables: unresolvedTemplates, EntityType: "pod spec"}
	}

	err = yaml.Unmarshal([]byte(replacedMarshalledSpec), &pod)
	if nil != err {
		return spec, err
	}

	return pod.Spec, nil
}

func ReplaceTemplateVariablesInServiceSpec(
	spec corev1.ServiceSpec,
	templates ...TemplateValueGetter,
) (corev1.ServiceSpec, error) {
	service := corev1.Service{Spec: spec}

	data, err := yaml.Marshal(service)
	if nil != err {
		return spec, err
	}

	replacedMarshalledSpec := ReplaceTemplateVariablesInString(string(data), templates...)
	unresolvedTemplates := GetUnresolvedTemplatesFromString(replacedMarshalledSpec)

	if len(unresolvedTemplates) > 0 {
		return spec, errors.UnresolvedTemplatesError{TemplateVariables: unresolvedTemplates, EntityType: "service spec"}
	}

	err = yaml.Unmarshal([]byte(replacedMarshalledSpec), &service)
	if nil != err {
		return spec, err
	}

	return service.Spec, nil
}

func ReplaceTemplateVariablesInIngressSpec(
	spec networkingv1.IngressSpec,
	templates ...TemplateValueGetter,
) (networkingv1.IngressSpec, error) {
	ingress := networkingv1.Ingress{Spec: spec}

	data, err := yaml.Marshal(ingress)
	if nil != err {
		return spec, err
	}

	replacedMarshalledSpec := ReplaceTemplateVariablesInString(string(data), templates...)
	unresolvedTemplates := GetUnresolvedTemplatesFromString(replacedMarshalledSpec)

	if len(unresolvedTemplates) > 0 {
		return spec, errors.UnresolvedTemplatesError{TemplateVariables: unresolvedTemplates, EntityType: "ingress spec"}
	}

	err = yaml.Unmarshal([]byte(replacedMarshalledSpec), &ingress)
	if nil != err {
		return spec, err
	}

	return ingress.Spec, nil
}
