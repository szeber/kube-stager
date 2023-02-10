package webhook

import (
	"context"
	"fmt"
	configv1 "github.com/szeber/kube-stager/apis/config/v1"
	sitev1 "github.com/szeber/kube-stager/apis/site/v1"
	"github.com/szeber/kube-stager/helpers/indexes"
	"github.com/szeber/kube-stager/helpers/labels"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type MongoConfigDeleteHandler struct {
	Client client.Client
}

func (r *MongoConfigDeleteHandler) Handle(ctx context.Context, req admission.Request) admission.Response {
	logger := log.FromContext(ctx)
	logger.Info("Validating deletion of mongo config " + req.Namespace + "/" + req.Name)

	var siteList sitev1.StagingSiteList
	err := r.Client.List(
		ctx,
		&siteList,
		client.InNamespace(req.Namespace),
		client.MatchingLabels{labels.MongoEnvironmentsPrefix + req.Name: "true"},
	)
	if nil != err {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	if len(siteList.Items) > 0 {
		var siteNames []string
		for _, v := range siteList.Items {
			siteNames = append(siteNames, v.Name)
		}
		logger.Info(
			fmt.Sprintf(
				"Denying delete request, because there are sites using this environment: %v",
				siteNames,
			),
		)
		return admission.Denied(fmt.Sprintf("There are sites using this environment: %v", siteNames))
	}

	var serviceList configv1.ServiceConfigList
	err = r.Client.List(
		ctx,
		&serviceList,
		client.InNamespace(req.Namespace),
		client.MatchingFields{indexes.DefaultMongoEnvironment: req.Name},
	)
	if nil != err {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	if len(serviceList.Items) > 0 {
		var serviceNames []string
		for _, v := range serviceList.Items {
			serviceNames = append(serviceNames, v.Name)
		}
		logger.Info(
			fmt.Sprintf(
				"Denying delete request, because there are services using this environment as a default: %v",
				serviceNames,
			),
		)
		return admission.Denied(fmt.Sprintf("There are services using this environment as a default: %v", serviceNames))
	}

	return admission.Allowed("")
}
