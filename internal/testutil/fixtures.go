package testutil

import (
	configv1 "github.com/szeber/kube-stager/apis/config/v1"
	jobv1 "github.com/szeber/kube-stager/apis/job/v1"
	sitev1 "github.com/szeber/kube-stager/apis/site/v1"
	taskv1 "github.com/szeber/kube-stager/apis/task/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewTestStagingSite(name, namespace string, services map[string]sitev1.StagingSiteService) *sitev1.StagingSite {
	if services == nil {
		services = map[string]sitev1.StagingSiteService{}
	}
	return &sitev1.StagingSite{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: sitev1.StagingSiteSpec{
			DomainPrefix:       name,
			DbName:             name,
			Username:           name,
			Password:           "testpassword",
			Enabled:            true,
			DisableAfter:       sitev1.TimeInterval{Days: 2},
			DeleteAfter:        sitev1.TimeInterval{Days: 7},
			BackupBeforeDelete: false,
			Services:           services,
			IncludeAllServices: false,
		},
	}
}

func NewTestServiceConfig(name, namespace, shortName string) *configv1.ServiceConfig {
	return &configv1.ServiceConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: configv1.ServiceConfigSpec{
			ShortName: shortName,
			DeploymentPodSpec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "nginx",
						Image: "nginx:latest",
					},
				},
			},
		},
	}
}

func NewTestServiceConfigWithDefaults(name, namespace, shortName string) *configv1.ServiceConfig {
	sc := NewTestServiceConfig(name, namespace, shortName)
	sc.Spec.DbInitPodSpec = &corev1.PodSpec{
		Containers: []corev1.Container{
			{
				Name:  "init",
				Image: "init:latest",
			},
		},
	}
	sc.Spec.MigrationJobPodSpec = &corev1.PodSpec{
		Containers: []corev1.Container{
			{
				Name:  "migrate",
				Image: "migrate:latest",
			},
		},
	}
	sc.Spec.BackupPodSpec = &corev1.PodSpec{
		Containers: []corev1.Container{
			{
				Name:  "backup",
				Image: "backup:latest",
			},
		},
	}
	port80 := int32(80)
	sc.Spec.ServiceSpec = &corev1.ServiceSpec{
		Ports: []corev1.ServicePort{
			{
				Name: "http",
				Port: port80,
			},
		},
	}
	return sc
}

func NewTestMysqlConfig(name, namespace string) *configv1.MysqlConfig {
	return &configv1.MysqlConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: configv1.MysqlConfigSpec{
			Host:     "mysql.example.com",
			Username: "admin",
			Password: "adminpass",
			Port:     3306,
		},
	}
}

func NewTestMongoConfig(name, namespace string) *configv1.MongoConfig {
	return &configv1.MongoConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: configv1.MongoConfigSpec{
			Host1:    "mongo1.example.com",
			Username: "admin",
			Password: "adminpass",
			Port:     27017,
		},
	}
}

func NewTestRedisConfig(name, namespace string) *configv1.RedisConfig {
	return &configv1.RedisConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: configv1.RedisConfigSpec{
			Host:                   "redis.example.com",
			AvailableDatabaseCount: 16,
			Port:                   6379,
		},
	}
}

func NewTestMysqlDatabase(name, namespace, siteName, serviceName, environment string) *taskv1.MysqlDatabase {
	return &taskv1.MysqlDatabase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: taskv1.MysqlDatabaseSpec{
			EnvironmentConfig: taskv1.EnvironmentConfig{
				ServiceName: serviceName,
				SiteName:    siteName,
				Environment: environment,
			},
			DatabaseName: name,
			Username:     "testuser",
			Password:     "testpassword",
		},
	}
}

func NewTestMongoDatabase(name, namespace, siteName, serviceName, environment string) *taskv1.MongoDatabase {
	return &taskv1.MongoDatabase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: taskv1.MongoDatabaseSpec{
			EnvironmentConfig: taskv1.EnvironmentConfig{
				ServiceName: serviceName,
				SiteName:    siteName,
				Environment: environment,
			},
			DatabaseName: name,
			Username:     "testuser",
			Password:     "testpassword",
		},
	}
}

func NewTestRedisDatabase(name, namespace, siteName, serviceName, environment string, dbNumber uint32) *taskv1.RedisDatabase {
	return &taskv1.RedisDatabase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: taskv1.RedisDatabaseSpec{
			EnvironmentConfig: taskv1.EnvironmentConfig{
				ServiceName: serviceName,
				SiteName:    siteName,
				Environment: environment,
			},
			DatabaseNumber: dbNumber,
		},
	}
}

func NewTestDbInitJob(name, namespace, siteName, serviceName string) *jobv1.DbInitJob {
	return &jobv1.DbInitJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: jobv1.DbInitJobSpec{
			SiteName:        siteName,
			ServiceName:     serviceName,
			DatabaseName:    "testdb",
			Username:        "testuser",
			Password:        "testpassword",
			DeadlineSeconds: 600,
		},
	}
}

func NewTestDbMigrationJob(name, namespace, siteName, serviceName, imageTag string) *jobv1.DbMigrationJob {
	return &jobv1.DbMigrationJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: jobv1.DbMigrationJobSpec{
			SiteName:        siteName,
			ServiceName:     serviceName,
			ImageTag:        imageTag,
			DeadlineSeconds: 600,
		},
	}
}

func NewTestBackup(name, namespace, siteName string, backupType jobv1.BackupType) *jobv1.Backup {
	return &jobv1.Backup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: jobv1.BackupSpec{
			SiteName:   siteName,
			BackupType: backupType,
		},
	}
}
