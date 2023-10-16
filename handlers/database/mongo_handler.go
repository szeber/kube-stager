package database

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	configv1 "github.com/szeber/kube-stager/api/config/v1"
	taskv1 "github.com/szeber/kube-stager/api/task/v1"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"time"
)

type mongoReconcileTask struct {
	logger     logr.Logger
	connection *mongo.Client
	ctx        context.Context
	username   string
	password   string
	database   string
}

type mongoUserResult struct {
	User         string
	PasswordHash string
}

func ReconcileMongoDatabase(database *taskv1.MongoDatabase, config configv1.MongoConfig, logger logr.Logger) (
	bool,
	error,
) {
	client, ctx, cancel, err := getMongoConnection(config, logger)

	if nil != cancel {
		defer cancel()
	}

	if nil != err {
		return false, err
	}

	task := mongoReconcileTask{
		logger:     logger,
		connection: client,
		ctx:        ctx,
		username:   database.Spec.Username,
		password:   database.Spec.Password,
		database:   database.Spec.DatabaseName,
	}

	if err := task.reconcileTask(); nil != err {
		return false, err
	}

	isChanged := false
	if database.Status.State != taskv1.Complete {
		isChanged = true
		database.Status.State = taskv1.Complete
	}

	return isChanged, nil
}

func DeleteMongoDatabase(database *taskv1.MongoDatabase, config configv1.MongoConfig, logger logr.Logger) error {
	client, ctx, cancel, err := getMongoConnection(config, logger)

	if nil != cancel {
		defer cancel()
	}

	if nil != err {
		return err
	}

	task := mongoReconcileTask{
		logger:     logger,
		connection: client,
		ctx:        ctx,
		username:   database.Spec.Username,
		password:   database.Spec.Password,
		database:   database.Spec.DatabaseName,
	}

	return task.deleteTask()
}

func getMongoConnection(config configv1.MongoConfig, logger logr.Logger) (
	*mongo.Client,
	context.Context,
	context.CancelFunc,
	error,
) {
	logger.Info("Connecting to database " + config.Name)

	uri := fmt.Sprintf(
		"mongodb://%s:%s@%s:%d",
		config.Spec.Username,
		config.Spec.Password,
		config.Spec.Host1,
		config.Spec.Port,
	)

	client, err := mongo.NewClient(options.Client().ApplyURI(uri))

	if err != nil {
		return nil, nil, nil, err
	}

	connectionCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	ctx := context.Background()

	if err = client.Connect(connectionCtx); err != nil {
		return nil, nil, cancel, err
	}

	if err = client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, nil, cancel, err
	}

	logger.Info("Connected")

	return client, ctx, cancel, nil
}

func (r *mongoReconcileTask) reconcileTask() error {
	return r.reconcileUser()
}

func (r *mongoReconcileTask) deleteTask() error {
	r.logger.Info("Deleting task")

	if err := r.removeDatabase(); nil != err {
		return err
	}

	if err := r.removeUser(); nil != err {
		return err
	}

	r.logger.Info("Task successfully deleted")

	return nil
}

func (r *mongoReconcileTask) reconcileUser() error {
	userExists, err := r.checkUserExists()

	if err != nil {
		return err
	}

	if userExists {
		return r.updateUser()
	} else {
		return r.createUser()
	}

}

func (r *mongoReconcileTask) checkUserExists() (bool, error) {
	var result struct {
		Users []struct{} `json:"user"`
	}

	err := r.connection.Database("admin").RunCommand(
		r.ctx,
		bson.D{{Key: "usersInfo", Value: r.username}},
	).Decode(&result)

	if err != nil {
		return false, err
	}

	return len(result.Users) > 0, nil
}

func (r *mongoReconcileTask) updateUser() error {
	r.logger.Info("Updating user " + r.username)

	_, err := r.connection.Database("admin").RunCommand(
		r.ctx,
		bson.D{
			{Key: "updateUser", Value: r.username},
			{Key: "pwd", Value: r.password},
			{
				Key: "roles", Value: bson.A{
					bson.D{{Key: "role", Value: "readWrite"}, {Key: "db", Value: r.database}},
				},
			},
		},
	).DecodeBytes()

	return err
}

func (r *mongoReconcileTask) createUser() error {
	r.logger.Info("Creating user " + r.username)

	_, err := r.connection.Database("admin").RunCommand(
		r.ctx,
		bson.D{
			{Key: "createUser", Value: r.username},
			{Key: "pwd", Value: r.password},
			{
				Key: "roles", Value: bson.A{
					bson.D{{Key: "role", Value: "readWrite"}, {Key: "db", Value: r.database}},
				},
			},
		},
	).DecodeBytes()

	return err
}

func (r *mongoReconcileTask) removeDatabase() error {
	result, err := r.connection.ListDatabases(r.ctx, bson.M{"name": r.database})

	if err != nil {
		return err
	}

	if len(result.Databases) == 0 {
		r.logger.Info("The database " + r.database + " doesn't exist")

		return nil
	}

	r.logger.Info("Dropping database " + r.database)

	return r.connection.Database(r.database).Drop(r.ctx)

}

func (r *mongoReconcileTask) removeUser() error {
	if r.username == "" {
		r.logger.Info(fmt.Sprintf("User is empty. %+v", r))
		return nil
	}

	if userExists, err := r.checkUserExists(); nil != err {
		return err
	} else if !userExists {
		r.logger.Info("The user " + r.username + " doesn't exist")

		return nil
	}

	r.logger.Info("Deleting user " + r.username)

	_, err := r.connection.Database("admin").RunCommand(
		r.ctx,
		bson.D{{Key: "dropUser", Value: r.username}},
	).DecodeBytes()

	return err
}
