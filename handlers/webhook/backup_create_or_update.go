package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	jobv1 "github.com/szeber/kube-stager/api/job/v1"
	sitev1 "github.com/szeber/kube-stager/api/site/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"net/http"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type BackupCreateOrUpdateHandler struct {
	Client  client.Client
	Scheme  *runtime.Scheme
	decoder *admission.Decoder
}

func (r *BackupCreateOrUpdateHandler) Handle(ctx context.Context, req admission.Request) admission.Response {
	logger := log.FromContext(ctx)
	job := &jobv1.Backup{}
	var err error

	if err = r.decoder.Decode(req, job); nil != err {
		return admission.Errored(http.StatusBadRequest, err)
	}

	site := &sitev1.StagingSite{}
	if err = r.Client.Get(
		ctx,
		client.ObjectKey{Namespace: job.Namespace, Name: job.Spec.SiteName},
		site,
	); nil != client.IgnoreNotFound(err) {
		logger.Error(err, "Failed to load the site for the backup job")
		return admission.Errored(http.StatusInternalServerError, err)
	}

	if nil != err {
		return admission.Denied(
			fmt.Sprintf(
				"Staging site with name %s not found in namespace %s",
				job.Spec.SiteName,
				job.Namespace,
			),
		)
	}

	if nil == metav1.GetControllerOf(job) {
		err = ctrl.SetControllerReference(site, job, r.Scheme)
		if nil != err {
			return admission.Errored(http.StatusInternalServerError, err)
		}

		marshaledJob, err := json.Marshal(job)
		if err != nil {
			return admission.Errored(http.StatusInternalServerError, err)
		}

		return admission.PatchResponseFromRaw(req.Object.Raw, marshaledJob)
	}

	return admission.Allowed("")
}

func (r *BackupCreateOrUpdateHandler) InjectDecoder(d *admission.Decoder) error {
	r.decoder = d
	return nil
}
