package errors

import (
	"fmt"
)

type EnvironmentConfig interface {
	GetSiteName() string
	GetServiceName() string
	GetEnvironment() string
}

type DatabaseCreationError struct {
	DatabaseType      DatabaseType
	EnvironmentConfig EnvironmentConfig
	Reason            string
}

type DatabaseInitError struct {
	SiteName    string
	ServiceName string
	Reason      string
}

type DatabaseMigrationError struct {
	SiteName    string
	ServiceName string
	Reason      string
}

type DatabaseType string

const (
	DatabaseTypeMongo DatabaseType = "Mongo"
	DatabaseTypeMysql DatabaseType = "Mysql"
	DatabaseTypeRedis DatabaseType = "Redis"
)

func (r DatabaseCreationError) Error() string {
	if "" == r.Reason {
		return fmt.Sprintf(
			"Failed to create %s database for site %s, service %s in environment %s",
			r.DatabaseType,
			r.EnvironmentConfig.GetSiteName(),
			r.EnvironmentConfig.GetServiceName(),
			r.EnvironmentConfig.GetEnvironment(),
		)
	} else {
		return fmt.Sprintf(
			"Failed to create %s database for site %s, service %s in environment %s. Reason: %s",
			r.DatabaseType,
			r.EnvironmentConfig.GetSiteName(),
			r.EnvironmentConfig.GetServiceName(),
			r.EnvironmentConfig.GetEnvironment(),
			r.Reason,
		)
	}
}

func (r DatabaseCreationError) IsFinal() bool {
	return true
}

func (r DatabaseInitError) Error() string {
	if "" == r.Reason {
		return fmt.Sprintf(
			"Database initialisation failed for site %s, service %s",
			r.SiteName,
			r.ServiceName,
		)
	} else {
		return fmt.Sprintf(
			"Database initialisation failed for site %s, service %s. Reason: %s",
			r.SiteName,
			r.ServiceName,
			r.Reason,
		)
	}
}

func (r DatabaseInitError) IsFinal() bool {
	return true
}

func (r DatabaseMigrationError) Error() string {
	if "" == r.Reason {
		return fmt.Sprintf(
			"Database migration failed for site %s, service %s",
			r.SiteName,
			r.ServiceName,
		)
	} else {
		return fmt.Sprintf(
			"Database migration failed for site %s, service %s. Reason: %s",
			r.SiteName,
			r.ServiceName,
			r.Reason,
		)
	}
}

func (r DatabaseMigrationError) IsFinal() bool {
	return true
}
