package job

import (
	"context"
	"errors"
	"fmt"
	jobv1 "github.com/szeber/kube-stager/apis/job/v1"
	sitev1 "github.com/szeber/kube-stager/apis/site/v1"
	"github.com/szeber/kube-stager/helpers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"time"
)

type BackupHandler struct {
	Reader client.Reader
	Writer client.Writer
	Scheme *runtime.Scheme
}

func (r *BackupHandler) Create(
	site *sitev1.StagingSite,
	ctx context.Context,
	backupType jobv1.BackupType,
	now time.Time,
) error {
	backupName, err := r.makeBackupName(site.Name, backupType, now)
	if nil != err {
		return err
	}

	job, err := r.createJob(site, backupName, backupType)
	if nil != err {
		return err
	}

	err = r.Writer.Create(ctx, job)
	if nil != err {
		return err
	}

	return nil
}

func (r *BackupHandler) EnsureFinalBackupIsComplete(site *sitev1.StagingSite, ctx context.Context) (bool, error) {
	logger := log.FromContext(ctx)
	backupName, err := r.makeBackupName(site.Name, jobv1.BackupTypeFinal, time.Now())
	if nil != err {
		return false, err
	}

	logger.V(0).Info("Trying to load existing backup job")
	job := &jobv1.Backup{}
	err = r.Reader.Get(ctx, client.ObjectKey{Namespace: site.Namespace, Name: backupName}, job)
	if nil != client.IgnoreNotFound(err) {
		return false, err
	}

	if nil != err {
		logger.V(0).Info("No existing backup job found, creating one")
		if job, err = r.createJob(site, backupName, jobv1.BackupTypeFinal); nil != err {
			return false, err
		}

		err = r.Writer.Create(ctx, job)
		if nil != err {
			return false, err
		}
		logger.V(1).Info("Backup job created")
	} else {
		logger.V(0).Info("Existing backup job found", "job", job)
		if jobv1.Complete == job.Status.State {
			return true, nil
		} else if jobv1.Failed == job.Status.State {
			logger.Error(errors.New("The existing backup job is in a failed state"), "Backup failed", "job", job)
			return true, nil
		}
	}

	return false, nil
}

func (r *BackupHandler) makeBackupName(
	siteName string,
	backupType jobv1.BackupType,
	scheduledTimestamp time.Time,
) (string, error) {
	switch backupType {
	case jobv1.BackupTypeFinal:
		return helpers.ShortenHumanReadableValue(fmt.Sprintf("final-%s", siteName), 63), nil
	case jobv1.BackupTypeScheduled:
		return helpers.ShortenHumanReadableValue(
			fmt.Sprintf("sched-%s-%d", siteName, scheduledTimestamp.Unix()),
			63,
		), nil
	default:
		return "", errors.New(fmt.Sprintf("Unhandled backup type: %s", backupType))
	}
}

func (r *BackupHandler) createJob(
	site *sitev1.StagingSite,
	name string,
	backupType jobv1.BackupType,
) (*jobv1.Backup, error) {
	job := &jobv1.Backup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: site.Namespace,
		},
		Spec: jobv1.BackupSpec{
			SiteName:   site.Name,
			BackupType: backupType,
		},
	}

	if err := ctrl.SetControllerReference(site, job, r.Scheme); nil != err {
		return job, nil
	}

	return job, nil
}
