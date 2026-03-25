package job

import (
	"context"
	"fmt"
	"testing"
	"time"

	jobv1 "github.com/szeber/kube-stager/apis/job/v1"
	"github.com/szeber/kube-stager/internal/testutil"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func newBackupHandler(objs ...client.Object) *BackupHandler {
	c := testutil.NewFakeClient(objs...)
	return &BackupHandler{
		Reader: c,
		Writer: c,
		Scheme: testutil.NewTestScheme(),
	}
}

func TestBackupHandler_Create_FinalBackup(t *testing.T) {
	ctx := context.Background()
	site := testutil.NewTestStagingSite("mysite", "test-ns", nil)
	handler := newBackupHandler(site)
	now := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)

	err := handler.Create(site, ctx, jobv1.BackupTypeFinal, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	backupList := &jobv1.BackupList{}
	err = handler.Reader.List(ctx, backupList, client.InNamespace("test-ns"))
	if err != nil {
		t.Fatalf("unexpected error listing backups: %v", err)
	}
	if len(backupList.Items) != 1 {
		t.Fatalf("expected 1 backup, got %d", len(backupList.Items))
	}

	backup := backupList.Items[0]
	if backup.Spec.SiteName != "mysite" {
		t.Errorf("SiteName = %q, want %q", backup.Spec.SiteName, "mysite")
	}
	if backup.Spec.BackupType != jobv1.BackupTypeFinal {
		t.Errorf("BackupType = %q, want %q", backup.Spec.BackupType, jobv1.BackupTypeFinal)
	}
	if backup.Namespace != "test-ns" {
		t.Errorf("Namespace = %q, want %q", backup.Namespace, "test-ns")
	}
	expectedName := "final-mysite"
	if backup.Name != expectedName {
		t.Errorf("Name = %q, want %q", backup.Name, expectedName)
	}
}

func TestBackupHandler_Create_ScheduledBackup(t *testing.T) {
	ctx := context.Background()
	site := testutil.NewTestStagingSite("mysite", "test-ns", nil)
	handler := newBackupHandler(site)
	now := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)

	err := handler.Create(site, ctx, jobv1.BackupTypeScheduled, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	backupList := &jobv1.BackupList{}
	err = handler.Reader.List(ctx, backupList, client.InNamespace("test-ns"))
	if err != nil {
		t.Fatalf("unexpected error listing backups: %v", err)
	}
	if len(backupList.Items) != 1 {
		t.Fatalf("expected 1 backup, got %d", len(backupList.Items))
	}

	backup := backupList.Items[0]
	if backup.Spec.SiteName != "mysite" {
		t.Errorf("SiteName = %q, want %q", backup.Spec.SiteName, "mysite")
	}
	if backup.Spec.BackupType != jobv1.BackupTypeScheduled {
		t.Errorf("BackupType = %q, want %q", backup.Spec.BackupType, jobv1.BackupTypeScheduled)
	}
	expectedName := fmt.Sprintf("sched-mysite-%d", now.Unix())
	if backup.Name != expectedName {
		t.Errorf("Name = %q, want %q", backup.Name, expectedName)
	}
}

func TestBackupHandler_Create_UnknownType(t *testing.T) {
	ctx := context.Background()
	site := testutil.NewTestStagingSite("mysite", "test-ns", nil)
	handler := newBackupHandler(site)
	now := time.Now()

	err := handler.Create(site, ctx, jobv1.BackupType("Unknown"), now)
	if err == nil {
		t.Error("expected error for unknown backup type, got nil")
	}
}

func TestBackupHandler_EnsureFinalBackupIsComplete_CreatesBackupWhenNotFound(t *testing.T) {
	ctx := context.Background()
	site := testutil.NewTestStagingSite("mysite", "test-ns", nil)
	handler := newBackupHandler(site)

	complete, err := handler.EnsureFinalBackupIsComplete(site, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if complete {
		t.Error("expected complete=false when backup was just created")
	}

	backupList := &jobv1.BackupList{}
	err = handler.Reader.List(ctx, backupList, client.InNamespace("test-ns"))
	if err != nil {
		t.Fatalf("unexpected error listing backups: %v", err)
	}
	if len(backupList.Items) != 1 {
		t.Fatalf("expected 1 backup to be created, got %d", len(backupList.Items))
	}

	backup := backupList.Items[0]
	if backup.Spec.BackupType != jobv1.BackupTypeFinal {
		t.Errorf("BackupType = %q, want %q", backup.Spec.BackupType, jobv1.BackupTypeFinal)
	}
	if backup.Spec.SiteName != "mysite" {
		t.Errorf("SiteName = %q, want %q", backup.Spec.SiteName, "mysite")
	}
}

func TestBackupHandler_EnsureFinalBackupIsComplete_PendingBackupReturnsFalse(t *testing.T) {
	ctx := context.Background()
	site := testutil.NewTestStagingSite("mysite", "test-ns", nil)
	existingBackup := testutil.NewTestBackup("final-mysite", "test-ns", "mysite", jobv1.BackupTypeFinal)
	existingBackup.Status.State = jobv1.Pending
	handler := newBackupHandler(site, existingBackup)

	complete, err := handler.EnsureFinalBackupIsComplete(site, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if complete {
		t.Error("expected complete=false for pending backup")
	}
}

func TestBackupHandler_EnsureFinalBackupIsComplete_CompleteBackupReturnsTrue(t *testing.T) {
	ctx := context.Background()
	site := testutil.NewTestStagingSite("mysite", "test-ns", nil)
	existingBackup := testutil.NewTestBackup("final-mysite", "test-ns", "mysite", jobv1.BackupTypeFinal)
	existingBackup.Status.State = jobv1.Complete
	handler := newBackupHandler(site, existingBackup)

	complete, err := handler.EnsureFinalBackupIsComplete(site, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !complete {
		t.Error("expected complete=true for completed backup")
	}
}

func TestBackupHandler_EnsureFinalBackupIsComplete_FailedBackupReturnsTrue(t *testing.T) {
	ctx := context.Background()
	site := testutil.NewTestStagingSite("mysite", "test-ns", nil)
	existingBackup := testutil.NewTestBackup("final-mysite", "test-ns", "mysite", jobv1.BackupTypeFinal)
	existingBackup.Status.State = jobv1.Failed
	handler := newBackupHandler(site, existingBackup)

	// A failed backup still returns true (considered done, even if failed)
	complete, err := handler.EnsureFinalBackupIsComplete(site, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !complete {
		t.Error("expected complete=true for failed backup")
	}
}

func TestBackupHandler_EnsureFinalBackupIsComplete_RunningBackupReturnsFalse(t *testing.T) {
	ctx := context.Background()
	site := testutil.NewTestStagingSite("mysite", "test-ns", nil)
	existingBackup := testutil.NewTestBackup("final-mysite", "test-ns", "mysite", jobv1.BackupTypeFinal)
	existingBackup.Status.State = jobv1.Running
	handler := newBackupHandler(site, existingBackup)

	complete, err := handler.EnsureFinalBackupIsComplete(site, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if complete {
		t.Error("expected complete=false for running backup")
	}
}
