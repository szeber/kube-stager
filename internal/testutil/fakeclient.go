package testutil

import (
	jobv1 "github.com/szeber/kube-stager/apis/job/v1"
	sitev1 "github.com/szeber/kube-stager/apis/site/v1"
	taskv1 "github.com/szeber/kube-stager/apis/task/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// NewFakeClient creates a fake controller-runtime client with the full test scheme
// and status subresource support for all custom types that have status subresources.
func NewFakeClient(initObjs ...client.Object) client.Client {
	return fake.NewClientBuilder().
		WithScheme(NewTestScheme()).
		WithObjects(initObjs...).
		WithStatusSubresource(
			&sitev1.StagingSite{},
			&taskv1.MysqlDatabase{},
			&taskv1.MongoDatabase{},
			&taskv1.RedisDatabase{},
			&jobv1.DbInitJob{},
			&jobv1.DbMigrationJob{},
			&jobv1.Backup{},
		).
		Build()
}
