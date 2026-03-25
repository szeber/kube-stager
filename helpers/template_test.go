package helpers

import (
	"strings"
	"testing"

	"github.com/szeber/kube-stager/helpers/errors"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
)

func TestReplaceTemplateVariablesInString(t *testing.T) {
	getter := StringMapTemplateValueGetter{StringMap: map[string]string{
		"site.name":   "mysite",
		"site.domain": "example.com",
	}}

	t.Run("single replacement", func(t *testing.T) {
		got := ReplaceTemplateVariablesInString("Hello ${site.name}", getter)
		if got != "Hello mysite" {
			t.Errorf("got %q, want %q", got, "Hello mysite")
		}
	})

	t.Run("multiple replacements", func(t *testing.T) {
		got := ReplaceTemplateVariablesInString("${site.name}.${site.domain}", getter)
		if got != "mysite.example.com" {
			t.Errorf("got %q, want %q", got, "mysite.example.com")
		}
	})

	t.Run("unknown template left intact", func(t *testing.T) {
		got := ReplaceTemplateVariablesInString("${unknown.var}", getter)
		if got != "${unknown.var}" {
			t.Errorf("got %q, want %q", got, "${unknown.var}")
		}
	})

	t.Run("multiple getters", func(t *testing.T) {
		getter2 := StringMapTemplateValueGetter{StringMap: map[string]string{
			"extra.val": "bonus",
		}}
		got := ReplaceTemplateVariablesInString("${site.name}-${extra.val}", getter, getter2)
		if got != "mysite-bonus" {
			t.Errorf("got %q, want %q", got, "mysite-bonus")
		}
	})
}

func TestGetTemplateVariables(t *testing.T) {
	t.Run("collects all keys", func(t *testing.T) {
		g1 := StringMapTemplateValueGetter{StringMap: map[string]string{"a": "1", "b": "2"}}
		g2 := StringMapTemplateValueGetter{StringMap: map[string]string{"c": "3"}}
		got := GetTemplateVariables(g1, g2)
		if len(got) != 3 {
			t.Errorf("got %d keys, want 3", len(got))
		}
	})

	t.Run("empty getters", func(t *testing.T) {
		got := GetTemplateVariables()
		if len(got) != 0 {
			t.Errorf("got %d keys, want 0", len(got))
		}
	})
}

func TestReplaceTemplateVariablesInStringMap(t *testing.T) {
	getter := StringMapTemplateValueGetter{StringMap: map[string]string{
		"site.name": "mysite",
	}}

	t.Run("all resolved", func(t *testing.T) {
		input := map[string]string{"key1": "val-${site.name}"}
		got, err := ReplaceTemplateVariablesInStringMap(input, "configmap", getter)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if got["key1"] != "val-mysite" {
			t.Errorf("got %q, want %q", got["key1"], "val-mysite")
		}
	})

	t.Run("unresolved returns error", func(t *testing.T) {
		input := map[string]string{"key1": "${unknown.var}"}
		_, err := ReplaceTemplateVariablesInStringMap(input, "configmap", getter)
		if err == nil {
			t.Error("expected error for unresolved template")
		}
		if _, ok := err.(errors.UnresolvedTemplatesError); !ok {
			t.Errorf("expected UnresolvedTemplatesError, got %T", err)
		}
	})
}

func TestGetUnresolvedTemplatesFromString(t *testing.T) {
	t.Run("finds templates", func(t *testing.T) {
		got := GetUnresolvedTemplatesFromString("${foo} and ${bar} and ${foo}")
		if len(got) != 2 {
			t.Errorf("got %d, want 2 (deduplicated)", len(got))
		}
	})

	t.Run("no templates", func(t *testing.T) {
		got := GetUnresolvedTemplatesFromString("no templates here")
		if got != nil {
			t.Errorf("got %v, want nil", got)
		}
	})
}

func TestReplaceTemplateVariablesInPodSpec(t *testing.T) {
	getter := StringMapTemplateValueGetter{StringMap: map[string]string{
		"site.imageTag": "v1.0",
		"site.name":     "mysite",
	}}

	t.Run("replaces in container image and env", func(t *testing.T) {
		spec := corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "app",
					Image: "myapp:${site.imageTag}",
					Env: []corev1.EnvVar{
						{Name: "SITE", Value: "${site.name}"},
					},
				},
			},
		}
		got, err := ReplaceTemplateVariablesInPodSpec(spec, getter)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Containers[0].Image != "myapp:v1.0" {
			t.Errorf("image = %q, want %q", got.Containers[0].Image, "myapp:v1.0")
		}
		if got.Containers[0].Env[0].Value != "mysite" {
			t.Errorf("env value = %q, want %q", got.Containers[0].Env[0].Value, "mysite")
		}
	})

	t.Run("error on unresolved", func(t *testing.T) {
		spec := corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "app", Image: "${unknown.image}"},
			},
		}
		_, err := ReplaceTemplateVariablesInPodSpec(spec, getter)
		if err == nil {
			t.Error("expected error for unresolved template")
		}
	})
}

func TestReplaceTemplateVariablesInServiceSpec(t *testing.T) {
	getter := StringMapTemplateValueGetter{StringMap: map[string]string{
		"site.name": "mysite",
	}}

	t.Run("replaces in selector", func(t *testing.T) {
		spec := corev1.ServiceSpec{
			Selector: map[string]string{"app": "${site.name}"},
			Ports:    []corev1.ServicePort{{Name: "http", Port: 80}},
		}
		got, err := ReplaceTemplateVariablesInServiceSpec(spec, getter)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Selector["app"] != "mysite" {
			t.Errorf("selector = %q, want %q", got.Selector["app"], "mysite")
		}
	})

	t.Run("error on unresolved", func(t *testing.T) {
		spec := corev1.ServiceSpec{
			Selector: map[string]string{"app": "${unknown.var}"},
		}
		_, err := ReplaceTemplateVariablesInServiceSpec(spec, getter)
		if err == nil {
			t.Error("expected error for unresolved template")
		}
	})
}

func TestReplaceTemplateVariablesInIngressSpec(t *testing.T) {
	getter := StringMapTemplateValueGetter{StringMap: map[string]string{
		"site.domain": "example.com",
	}}

	t.Run("replaces in host", func(t *testing.T) {
		pathType := networkingv1.PathTypePrefix
		spec := networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: "${site.domain}",
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pathType,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: "svc",
											Port: networkingv1.ServiceBackendPort{Number: 80},
										},
									},
								},
							},
						},
					},
				},
			},
		}
		got, err := ReplaceTemplateVariablesInIngressSpec(spec, getter)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Rules[0].Host != "example.com" {
			t.Errorf("host = %q, want %q", got.Rules[0].Host, "example.com")
		}
	})

	t.Run("error on unresolved", func(t *testing.T) {
		spec := networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{Host: "${unknown.host}"},
			},
		}
		_, err := ReplaceTemplateVariablesInIngressSpec(spec, getter)
		if err == nil {
			t.Error("expected error for unresolved template")
		}
		if !strings.Contains(err.Error(), "ingress spec") {
			t.Errorf("error should mention ingress spec: %v", err)
		}
	})
}

func TestStringMapTemplateValueGetter_GetTemplateValues(t *testing.T) {
	m := map[string]string{"a": "1", "b": "2"}
	getter := StringMapTemplateValueGetter{StringMap: m}
	got := getter.GetTemplateValues()
	if len(got) != 2 || got["a"] != "1" || got["b"] != "2" {
		t.Errorf("GetTemplateValues() = %v, want %v", got, m)
	}
}
