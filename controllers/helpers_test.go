package controller

import (
	"context"
	"fmt"
	"testing"

	sitev1 "github.com/szeber/kube-stager/apis/site/v1"
	"github.com/szeber/kube-stager/internal/testutil"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestSaveStatusUpdatesIfObjectChanged(t *testing.T) {
	t.Run("not changed returns without update", func(t *testing.T) {
		site := testutil.NewTestStagingSite("test", "default", nil)
		site.Status.State = sitev1.StatePending
		c := testutil.NewFakeClient(site)
		result, err := SaveStatusUpdatesIfObjectChanged(false, c.Status(), context.Background(), site, ctrl.Result{}, nil)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if result != (ctrl.Result{}) {
			t.Error("should not requeue")
		}
	})

	t.Run("changed calls status update", func(t *testing.T) {
		site := testutil.NewTestStagingSite("test", "default", nil)
		c := testutil.NewFakeClient(site)
		site.Status.State = sitev1.StateComplete
		result, err := SaveStatusUpdatesIfObjectChanged(true, c.Status(), context.Background(), site, ctrl.Result{}, nil)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if result != (ctrl.Result{}) {
			t.Error("should not requeue")
		}
	})

	t.Run("original err preserved when status update would also error", func(t *testing.T) {
		site := testutil.NewTestStagingSite("test", "default", nil)
		c := testutil.NewFakeClient(site)
		origErr := fmt.Errorf("original error")
		_, err := SaveStatusUpdatesIfObjectChanged(true, c.Status(), context.Background(), site, ctrl.Result{}, origErr)
		if err != origErr {
			t.Errorf("expected original error, got: %v", err)
		}
	})

	t.Run("not changed preserves original error", func(t *testing.T) {
		site := testutil.NewTestStagingSite("test", "default", nil)
		c := testutil.NewFakeClient(site)
		origErr := fmt.Errorf("original error")
		_, err := SaveStatusUpdatesIfObjectChanged(false, c.Status(), context.Background(), site, ctrl.Result{}, origErr)
		if err != origErr {
			t.Errorf("expected original error, got: %v", err)
		}
	})
}
