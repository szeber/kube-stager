package controller

import (
	"context"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func SaveStatusUpdatesIfObjectChanged(
	isChanged bool,
	writer client.StatusWriter,
	ctx context.Context,
	obj client.Object,
	result ctrl.Result,
	err error,
) (ctrl.Result, error) {
	if !isChanged {
		return result, err
	}

	newErr := writer.Update(ctx, obj)
	if err == nil {
		return result, newErr
	}
	return result, err
}
