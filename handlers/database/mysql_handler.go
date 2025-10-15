package database

import (
	"database/sql"
	"fmt"
	"github.com/go-logr/logr"
	_ "github.com/go-sql-driver/mysql"
	configv1 "github.com/szeber/kube-stager/apis/config/v1"
	taskv1 "github.com/szeber/kube-stager/apis/task/v1"
)

type mysqlReconcileTask struct {
	logger     logr.Logger
	connection *sql.DB
	host       string
	port       uint16
	username   string
	password   string
	database   string
}

type mysqlUserResult struct {
	User         string
	PasswordHash string
}

func ReconcileMysqlDatabase(database *taskv1.MysqlDatabase, config configv1.MysqlConfig, logger logr.Logger) (
	bool,
	error,
) {
	logger.Info("Connecting to database " + config.Name)

	dataSourceName := fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/mysql",
		config.Spec.Username,
		config.Spec.Password,
		config.Spec.Host,
		config.Spec.Port,
	)

	connection, err := sql.Open("mysql", dataSourceName)

	if err != nil {
		return false, err
	}

	defer connection.Close()

	logger.Info("Connected")

	task := mysqlReconcileTask{
		logger:     logger,
		connection: connection,
		host:       config.Spec.Host,
		port:       config.Spec.Port,
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

func DeleteMysqlDatabase(database *taskv1.MysqlDatabase, config configv1.MysqlConfig, logger logr.Logger) error {
	logger.Info("Connecting to database " + config.Name)

	dataSourceName := fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/mysql",
		config.Spec.Username,
		config.Spec.Password,
		config.Spec.Host,
		config.Spec.Port,
	)

	connection, err := sql.Open("mysql", dataSourceName)

	if err != nil {
		return err
	}

	defer connection.Close()

	logger.Info("Connected")

	task := mysqlReconcileTask{
		logger:     logger,
		connection: connection,
		username:   database.Spec.Username,
		password:   database.Spec.Password,
		database:   database.Spec.DatabaseName,
	}

	return task.deleteTask()
}

func (r *mysqlReconcileTask) reconcileTask() error {
	r.logger.Info("Reconciling task")

	if err := r.reconcileUser(); nil != err {
		return err
	}

	if err := r.reconcileDatabase(); nil != err {
		return err
	}

	if err := r.reconcilePermissions(); nil != err {
		return err
	}

	r.logger.Info("Task successfully reconciled")

	return nil
}

func (r *mysqlReconcileTask) deleteTask() error {
	r.logger.Info("Deleting task")

	if err := r.removeUser(); nil != err {
		return err
	}

	if err := r.revokePermissions(); nil != err {
		return err
	}

	if err := r.removeDatabase(); nil != err {
		return err
	}

	r.logger.Info("Task successfully deleted")

	return nil
}

func (r *mysqlReconcileTask) reconcileUser() error {
	r.logger.Info("Starting user reconciliation")

	var user mysqlUserResult
	if err := r.getUser(&user); err != nil {
		return err
	}

	r.logger.Info(fmt.Sprintf("Existing user: %v", user))

	if user.User == r.username {
		passwordCorrect := r.checkUserCanLogin()

		if passwordCorrect {
			r.logger.Info("the mysql user is up to date")

			return nil
		} else {
			r.logger.Info("the mysql password is incorrect for the user, changing password")

			if err := r.changePassword(); nil != err {
				return err
			}

			r.logger.Info("password updated")
		}
	} else {
		r.logger.Info("mysql user does not exist, creating it")

		if err := r.createUser(); nil != err {
			return err
		}

		r.logger.Info("user created")
	}

	return nil
}

func (r *mysqlReconcileTask) removeUser() error {
	if r.username == "" {
		return nil
	}

	var user mysqlUserResult
	if err := r.getUser(&user); err != nil {
		return err
	}

	if user.User == r.username {
		r.logger.Info("Removing user")

		_, err := r.connection.Exec(fmt.Sprintf("DROP USER '%s'@'%%'", r.username))

		return err
	}

	return nil
}

func (r *mysqlReconcileTask) reconcileDatabase() error {
	_, err := r.connection.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`", r.database))

	return err
}

func (r *mysqlReconcileTask) reconcilePermissions() error {
	havePerms, err := r.revokeNotRequestedPermissions()

	if nil != err {
		return nil
	}

	if !havePerms {
		r.logger.Info("Granting all privileges on db " + r.database)
		_, err = r.connection.Exec(fmt.Sprintf("GRANT ALL ON `%s`.* TO '%s'@'%%'", r.database, r.username))

		if err != nil {
			panic(err)
			//return err
		}
	}

	return nil
}

func (r *mysqlReconcileTask) getDatabasesWhereUserHasPermissions() ([]string, error) {
	var dbNames []string

	if r.username == "" {
		return dbNames, nil
	}

	result, err := r.connection.Query("SELECT Db FROM db WHERE User = ? AND Host = ?", r.username, "%")

	if err != nil {
		return dbNames, err
	}

	defer result.Close()

	var dbName string

	for result.Next() {
		dbNames = append(dbNames, dbName)
	}

	return dbNames, nil
}

func (r *mysqlReconcileTask) revokePermissionOnDatabase(dbName string) error {
	if "" == dbName {
		return nil
	}

	r.logger.Info("Revoking all privileges on db " + dbName)
	_, err := r.connection.Exec(fmt.Sprintf("REVOKE ALL ON `%s`.* FROM '%s'@'%%'", dbName, r.username))

	if err != nil {
		return err
	}

	return nil
}

func (r *mysqlReconcileTask) revokeNotRequestedPermissions() (bool, error) {
	hasPermissionsOnTaskDatabase := false
	dbNames, err := r.getDatabasesWhereUserHasPermissions()

	if nil != err {
		return hasPermissionsOnTaskDatabase, err
	}

	for _, dbName := range dbNames {
		if dbName == r.database {
			hasPermissionsOnTaskDatabase = true
		} else {
			if err = r.revokePermissionOnDatabase(dbName); nil != err {
				return hasPermissionsOnTaskDatabase, err
			}
		}
	}

	return hasPermissionsOnTaskDatabase, nil
}

func (r *mysqlReconcileTask) createUser() error {
	_, err := r.connection.Query(fmt.Sprintf("CREATE USER '%s'@'%%' IDENTIFIED BY '%s'", r.username, r.password))

	if nil != err {
		return err
	}

	_, err = r.connection.Query("FLUSH PRIVILEGES")

	return err
}

func (r *mysqlReconcileTask) getUser(user *mysqlUserResult) error {
	err := r.connection.QueryRow(
		"SELECT User, authentication_string from user WHERE User = ? AND Host = ?",
		r.username,
		"%",
	).Scan(&user.User, &user.PasswordHash)

	if err != nil && err != sql.ErrNoRows {
		return err
	}

	return nil
}

func (r *mysqlReconcileTask) changePassword() error {
	_, err := r.connection.Query(fmt.Sprintf("ALTER USER '%s'@'%%' IDENTIFIED BY '%s'", r.username, r.password))

	if nil != err {
		return err
	}

	_, err = r.connection.Query("FLUSH PRIVILEGES")

	return err
}

func (r *mysqlReconcileTask) removeDatabase() error {
	r.logger.Info("Dropping database if it exists")
	_, err := r.connection.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", r.database))

	return err
}

func (r *mysqlReconcileTask) revokePermissions() error {
	dbNames, err := r.getDatabasesWhereUserHasPermissions()

	if nil != err {
		return err
	}

	for _, dbName := range dbNames {
		if err := r.revokePermissionOnDatabase(dbName); nil != err {
			return err
		}
	}

	return nil
}

func (r *mysqlReconcileTask) checkUserCanLogin() bool {
	dataSourceName := fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/mysql",
		r.username,
		r.password,
		r.host,
		r.port,
	)

	_, err := sql.Open("mysql", dataSourceName)

	return nil == err
}
