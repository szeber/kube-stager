package webhook

import (
	"context"
	"fmt"
	sitev1 "github.com/szeber/kube-stager/api/site/v1"
	"github.com/szeber/kube-stager/helpers/labels"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type ServiceConfigDeleteHandler struct {
	Client client.Client
}

func (r *ServiceConfigDeleteHandler) Handle(ctx context.Context, req admission.Request) admission.Response {
	logger := log.FromContext(ctx)
	logger.Info("Validating deletion of service " + req.Namespace + "/" + req.Name)

	var siteList sitev1.StagingSiteList
	err := r.Client.List(
		ctx,
		&siteList,
		client.InNamespace(req.Namespace),
		client.MatchingLabels{labels.ServicesPrefix + req.Name: "true"},
	)
	if nil != err {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	if len(siteList.Items) > 0 {
		var siteNames []string
		for _, v := range siteList.Items {
			siteNames = append(siteNames, v.Name)
		}
		logger.Info(fmt.Sprintf("Denying delete request, because there are sites using this service: %v", siteNames))
		return admission.Denied(fmt.Sprintf("There are sites using this service: %v", siteNames))
	}

	return admission.Allowed("")
}
