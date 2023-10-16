package task

import (
	"context"
	sitev1 "github.com/szeber/kube-stager/api/site/v1"
)

type TaskHandler interface {
	EnsureDatabasesAreCreated(site *sitev1.StagingSite, ctx context.Context) (bool, error)
	EnsureDatabasesAreReady(site *sitev1.StagingSite, ctx context.Context) (bool, error)
}
