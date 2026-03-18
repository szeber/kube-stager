package webhook

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	jobv1 "github.com/szeber/kube-stager/apis/job/v1"
	"github.com/szeber/kube-stager/internal/testutil"
)

func makeBackupAdmissionRequest(backup *jobv1.Backup) admission.Request {
	raw, _ := json.Marshal(backup)
	return admission.Request{
		AdmissionRequest: admissionv1.AdmissionRequest{
			Object:    runtime.RawExtension{Raw: raw},
			Namespace: backup.Namespace,
			Name:      backup.Name,
		},
	}
}

func TestBackupCreateOrUpdateHandler_ValidBackupWithExistingSite(t *testing.T) {
	const ns = "test-ns"
	const siteName = "mysite"

	site := testutil.NewTestStagingSite(siteName, ns, nil)
	backup := testutil.NewTestBackup("mybackup", ns, siteName, jobv1.BackupTypeManual)

	handler := &BackupCreateOrUpdateHandler{
		Client:  testutil.NewFakeClient(site),
		Scheme:  testutil.NewTestScheme(),
		Decoder: admission.NewDecoder(testutil.NewTestScheme()),
	}

	resp := handler.Handle(context.Background(), makeBackupAdmissionRequest(backup))

	if !resp.Allowed {
		t.Errorf("expected Allowed for backup with existing site, got Denied: %v", resp.Result)
	}

	// The handler sets a controller reference via a JSON patch when no owner is set
	if len(resp.Patches) == 0 {
		t.Error("expected patches to be set (controller reference), got none")
	}
}

func TestBackupCreateOrUpdateHandler_SiteNotFound(t *testing.T) {
	const ns = "test-ns"
	const siteName = "nonexistent-site"

	// No site registered in the fake client
	backup := testutil.NewTestBackup("mybackup", ns, siteName, jobv1.BackupTypeManual)

	handler := &BackupCreateOrUpdateHandler{
		Client:  testutil.NewFakeClient(),
		Scheme:  testutil.NewTestScheme(),
		Decoder: admission.NewDecoder(testutil.NewTestScheme()),
	}

	resp := handler.Handle(context.Background(), makeBackupAdmissionRequest(backup))

	if resp.Allowed {
		t.Error("expected Denied when site does not exist, got Allowed")
	}
	if resp.Result == nil || resp.Result.Code != http.StatusForbidden {
		t.Errorf("expected Forbidden (403), got: %v", resp.Result)
	}
}

func TestBackupCreateOrUpdateHandler_ScheduledBackupWithExistingSite(t *testing.T) {
	const ns = "test-ns"
	const siteName = "mysite"

	site := testutil.NewTestStagingSite(siteName, ns, nil)
	backup := testutil.NewTestBackup("scheduled-backup", ns, siteName, jobv1.BackupTypeScheduled)

	handler := &BackupCreateOrUpdateHandler{
		Client:  testutil.NewFakeClient(site),
		Scheme:  testutil.NewTestScheme(),
		Decoder: admission.NewDecoder(testutil.NewTestScheme()),
	}

	resp := handler.Handle(context.Background(), makeBackupAdmissionRequest(backup))

	if !resp.Allowed {
		t.Errorf("expected Allowed for scheduled backup with existing site, got Denied: %v", resp.Result)
	}
}

func TestBackupCreateOrUpdateHandler_FinalBackupWithExistingSite(t *testing.T) {
	const ns = "test-ns"
	const siteName = "mysite"

	site := testutil.NewTestStagingSite(siteName, ns, nil)
	backup := testutil.NewTestBackup("final-backup", ns, siteName, jobv1.BackupTypeFinal)

	handler := &BackupCreateOrUpdateHandler{
		Client:  testutil.NewFakeClient(site),
		Scheme:  testutil.NewTestScheme(),
		Decoder: admission.NewDecoder(testutil.NewTestScheme()),
	}

	resp := handler.Handle(context.Background(), makeBackupAdmissionRequest(backup))

	if !resp.Allowed {
		t.Errorf("expected Allowed for final backup with existing site, got Denied: %v", resp.Result)
	}
}

func TestBackupCreateOrUpdateHandler_SiteInDifferentNamespaceNotFound(t *testing.T) {
	const ns = "test-ns"
	const otherNs = "other-ns"
	const siteName = "mysite"

	// Site exists in a different namespace
	site := testutil.NewTestStagingSite(siteName, otherNs, nil)
	backup := testutil.NewTestBackup("mybackup", ns, siteName, jobv1.BackupTypeManual)

	handler := &BackupCreateOrUpdateHandler{
		Client:  testutil.NewFakeClient(site),
		Scheme:  testutil.NewTestScheme(),
		Decoder: admission.NewDecoder(testutil.NewTestScheme()),
	}

	resp := handler.Handle(context.Background(), makeBackupAdmissionRequest(backup))

	if resp.Allowed {
		t.Error("expected Denied when site is in a different namespace, got Allowed")
	}
}
