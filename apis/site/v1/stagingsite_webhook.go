/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	"context"
	"time"

	"github.com/sethvargo/go-password/password"
	"github.com/szeber/kube-stager/helpers"
	"github.com/szeber/kube-stager/helpers/annotations"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var stagingsitelog = logf.Log.WithName("stagingsite-resource")

func (r *StagingSite) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		WithDefaulter(&StagingSiteDefaulter{}).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-site-operator-kube-stager-io-v1-stagingsite,mutating=true,failurePolicy=fail,sideEffects=None,groups=site.operator.kube-stager.io,resources=stagingsites,verbs=create;update,versions=v1,name=mstagingsite.kb.io,admissionReviewVersions={v1,v1beta1}

// StagingSiteDefaulter implements admission.CustomDefaulter for StagingSite
type StagingSiteDefaulter struct{}

var _ admission.CustomDefaulter = &StagingSiteDefaulter{}

// Default implements admission.CustomDefaulter so a webhook will be registered for the type
func (d *StagingSiteDefaulter) Default(ctx context.Context, obj runtime.Object) error {
	r, ok := obj.(*StagingSite)
	if !ok {
		return nil
	}

	stagingsitelog.Info("default", "name", r.Name)

	if 0 == len(r.ObjectMeta.Annotations) {
		r.ObjectMeta.Annotations = make(map[string]string)
	}

	r.ObjectMeta.Annotations[annotations.StagingSiteLastSpecChangeAt] = time.Now().Format(time.RFC3339)

	if "" == r.Spec.DomainPrefix {
		if len(r.Name) > 63 {
			r.Spec.DomainPrefix = r.Name[0:63]
		} else {
			r.Spec.DomainPrefix = r.Name
		}
	}

	if "" == r.Spec.DbName {
		r.Spec.DbName = helpers.SanitiseAndShortenDbValue(r.Name, 63)
	}

	if r.Spec.Username == "" {
		r.Spec.Username = helpers.SanitiseAndShortenDbValue(r.Spec.DbName, 16)
	}

	if r.Spec.Password == "" {
		randomPassword, err := password.Generate(25, 6, 0, false, true)
		if err != nil {
			return err
		}
		r.Spec.Password = helpers.SanitiseDbValue(randomPassword)
	}

	if r.isTimeIntervalEmpty(r.Spec.DisableAfter) {
		r.Spec.DisableAfter = TimeInterval{
			Never:   false,
			Days:    2,
			Hours:   0,
			Minutes: 0,
		}
	}

	if r.isTimeIntervalEmpty(r.Spec.DeleteAfter) {
		r.Spec.DeleteAfter = TimeInterval{
			Never:   false,
			Days:    7,
			Hours:   0,
			Minutes: 0,
		}
	}

	return nil
}
