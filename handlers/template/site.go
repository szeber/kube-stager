package template

import (
	"context"
	"fmt"
	api "github.com/szeber/kube-stager/apis"
	configv1 "github.com/szeber/kube-stager/apis/config/v1"
	sitev1 "github.com/szeber/kube-stager/apis/site/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DatabaseHandler interface {
	SetMysql(config map[string]configv1.MysqlConfig)
	SetMongo(config map[string]configv1.MongoConfig)
	SetRedis(config map[string]configv1.RedisConfig)
	GetMysql() map[string]configv1.MysqlConfig
	GetMongo() map[string]configv1.MongoConfig
	GetRedis() map[string]configv1.RedisConfig
	SetServiceConfigs(configs map[string]configv1.ServiceConfig)
	SetServiceConfig(name string, config configv1.ServiceConfig)
	getNamespace() string
}

type SiteTemplateHandler struct {
	site                 sitev1.StagingSite
	siteServiceSpec      sitev1.StagingSiteService
	siteServiceStatus    sitev1.StagingSiteServiceStatus
	currentServiceConfig configv1.ServiceConfig
	serviceConfigs       map[string]configv1.ServiceConfig
	mysqlConfigs         map[string]configv1.MysqlConfig
	mongoConfigs         map[string]configv1.MongoConfig
	redisConfigs         map[string]configv1.RedisConfig
}

func NewSite(site sitev1.StagingSite, serviceConfig configv1.ServiceConfig) SiteTemplateHandler {
	return SiteTemplateHandler{
		site:                 site,
		siteServiceSpec:      site.Spec.Services[serviceConfig.Name],
		siteServiceStatus:    site.Status.Services[serviceConfig.Name],
		currentServiceConfig: serviceConfig,
	}
}

func LoadConfigs(handler DatabaseHandler, ctx context.Context, reader client.Reader) error {
	namespace := handler.getNamespace()

	mongoConfigs, err := ListMongoConfigsInNamespace(namespace, ctx, reader)
	if nil != err {
		return err
	}
	handler.SetMongo(mongoConfigs)

	mysqlConfigs, err := ListMysqlConfigsInNamespace(namespace, ctx, reader)
	if nil != err {
		return err
	}
	handler.SetMysql(mysqlConfigs)

	redisConfigs, err := ListRedisConfigsInNamespace(namespace, ctx, reader)
	if nil != err {
		return err
	}
	handler.SetRedis(redisConfigs)

	return LoadServiceConfigs(handler, ctx, reader)
}

func ListMongoConfigsInNamespace(namespace string, ctx context.Context, reader client.Reader) (map[string]configv1.MongoConfig, error) {
	configs := make(map[string]configv1.MongoConfig)
	list := configv1.MongoConfigList{}

	if err := reader.List(ctx, &list, &client.ListOptions{Namespace: namespace}); nil != err {
		return configs, err
	}

	for _, config := range list.Items {
		configs[config.Name] = config
	}

	return configs, nil
}

func ListMysqlConfigsInNamespace(namespace string, ctx context.Context, reader client.Reader) (map[string]configv1.MysqlConfig, error) {
	list := configv1.MysqlConfigList{}
	configs := make(map[string]configv1.MysqlConfig)

	if err := reader.List(ctx, &list, &client.ListOptions{Namespace: namespace}); nil != err {
		return configs, err
	}

	for _, config := range list.Items {
		configs[config.Name] = config
	}
	return configs, nil
}

func ListRedisConfigsInNamespace(namespace string, ctx context.Context, reader client.Reader) (map[string]configv1.RedisConfig, error) {
	list := configv1.RedisConfigList{}
	configs := make(map[string]configv1.RedisConfig)

	if err := reader.List(ctx, &list, &client.ListOptions{Namespace: namespace}); nil != err {
		return configs, err
	}

	for _, config := range list.Items {
		configs[config.Name] = config
	}

	return configs, nil
}

func LoadServiceConfigs(handler DatabaseHandler, ctx context.Context, reader client.Reader) error {
	namespace := handler.getNamespace()

	configList := configv1.ServiceConfigList{}
	configs := make(map[string]configv1.ServiceConfig)
	if err := reader.List(ctx, &configList, client.InNamespace(namespace)); nil != err {
		return err
	}
	for _, v := range configList.Items {
		configs[v.Name] = v
	}
	handler.SetServiceConfigs(configs)

	return nil
}

func (r *SiteTemplateHandler) SetMysql(configs map[string]configv1.MysqlConfig) {
	r.mysqlConfigs = configs
}

func (r *SiteTemplateHandler) SetMongo(configs map[string]configv1.MongoConfig) {
	r.mongoConfigs = configs
}

func (r *SiteTemplateHandler) SetRedis(configs map[string]configv1.RedisConfig) {
	r.redisConfigs = configs
}

func (r *SiteTemplateHandler) GetMysql() map[string]configv1.MysqlConfig {
	return r.mysqlConfigs
}

func (r *SiteTemplateHandler) GetMongo() map[string]configv1.MongoConfig {
	return r.mongoConfigs
}

func (r *SiteTemplateHandler) GetRedis() map[string]configv1.RedisConfig {
	return r.redisConfigs
}

func (r *SiteTemplateHandler) SetServiceConfigs(configs map[string]configv1.ServiceConfig) {
	r.serviceConfigs = configs
}

func (r *SiteTemplateHandler) SetServiceConfig(name string, config configv1.ServiceConfig) {
	if 0 == len(r.serviceConfigs) {
		r.serviceConfigs = make(map[string]configv1.ServiceConfig)
	}
	r.serviceConfigs[name] = config
}

func (r *SiteTemplateHandler) getNamespace() string {
	return r.site.Namespace
}

func (r *SiteTemplateHandler) GetTemplateValues() map[string]string {
	result := map[string]string{
		"site.name":         r.site.Name,
		"site.domainPrefix": r.site.Spec.DomainPrefix,
		"site.imageTag":     r.siteServiceSpec.ImageTag,
	}

	for k, v := range r.getCommonDatabaseConfigTemplateValues(r.siteServiceStatus, r.siteServiceSpec) {
		result[k] = v
	}

	for k, v := range r.getMysqlConfigTemplateValues(r.mysqlConfigs, r.siteServiceSpec.MysqlEnvironment, r.currentServiceConfig.Spec.DefaultMysqlEnvironment) {
		result[k] = v
	}

	for k, v := range r.getMongoConfigTemplateValues(r.mongoConfigs, r.siteServiceSpec.MongoEnvironment, r.currentServiceConfig.Spec.DefaultMongoEnvironment) {
		result[k] = v
	}

	for k, v := range r.getRedisConfigTemplateValues(r.redisConfigs, r.siteServiceSpec.RedisEnvironment, r.currentServiceConfig.Spec.DefaultRedisEnvironment) {
		result[k] = v
	}

	for name := range r.currentServiceConfig.Spec.ConfigMaps {
		result["site.configmap."+name] = api.MakeConfigmapName(&r.site, &r.currentServiceConfig, name)
	}

	for name, value := range r.currentServiceConfig.Spec.CustomTemplateValues {
		result["site.custom."+name] = value
	}

	for name, value := range r.siteServiceSpec.CustomTemplateValues {
		result["site.custom."+name] = value
	}

	for name, config := range r.serviceConfigs {
		result["service."+name+".clusterUrl"] = api.MakeServiceUrl(&r.site, config.Spec.ShortName)

		for k, v := range r.getCommonDatabaseConfigTemplateValues(r.site.Status.Services[name], r.site.Spec.Services[name]) {
			result[fmt.Sprintf("service.%s.%s", name, k)] = v
		}

		for k, v := range r.getMysqlConfigTemplateValues(r.mysqlConfigs, r.site.Spec.Services[name].MysqlEnvironment, config.Spec.DefaultMysqlEnvironment) {
			result[fmt.Sprintf("service.%s.%s", name, k)] = v
		}
		for k, v := range r.getMongoConfigTemplateValues(r.mongoConfigs, r.site.Spec.Services[name].MongoEnvironment, config.Spec.DefaultMongoEnvironment) {
			result[fmt.Sprintf("service.%s.%s", name, k)] = v
		}
		for k, v := range r.getRedisConfigTemplateValues(r.redisConfigs, r.site.Spec.Services[name].RedisEnvironment, config.Spec.DefaultRedisEnvironment) {
			result[fmt.Sprintf("service.%s.%s", name, k)] = v
		}
	}

	return result
}

func (r *SiteTemplateHandler) getMysqlConfigTemplateValues(
	mysqlConfigs map[string]configv1.MysqlConfig,
	siteEnvironmentName string,
	serviceDefaultEnvironmentName string,
) map[string]string {
	result := make(map[string]string)
	var configName string

	if "" == siteEnvironmentName {
		if "" == serviceDefaultEnvironmentName {
			return result
		} else {
			configName = serviceDefaultEnvironmentName
		}
	} else {
		configName = siteEnvironmentName
	}

	mysqlConfig := mysqlConfigs[configName]
	result["database.mysql.host"] = mysqlConfig.Spec.Host
	result["database.mysql.port"] = fmt.Sprintf("%d", mysqlConfig.Spec.Port)

	return result
}

func (r *SiteTemplateHandler) getMongoConfigTemplateValues(
	mongoConfigs map[string]configv1.MongoConfig,
	siteEnvironmentName string,
	serviceDefaultEnvironmentName string,
) map[string]string {
	result := make(map[string]string)
	var configName string

	if "" == siteEnvironmentName {
		if "" == serviceDefaultEnvironmentName {
			return result
		} else {
			configName = serviceDefaultEnvironmentName
		}
	} else {
		configName = siteEnvironmentName
	}

	mongoConfig := mongoConfigs[configName]
	result["database.mongo.host1"] = mongoConfig.Spec.Host1
	result["database.mongo.host2"] = mongoConfig.Spec.Host2
	result["database.mongo.host3"] = mongoConfig.Spec.Host3
	result["database.mongo.port"] = fmt.Sprintf("%d", mongoConfig.Spec.Port)

	return result
}

func (r *SiteTemplateHandler) getRedisConfigTemplateValues(
	redisConfigs map[string]configv1.RedisConfig,
	siteEnvironmentName string,
	serviceDefaultEnvironmentName string,
) map[string]string {
	result := make(map[string]string)
	var configName string

	if "" == siteEnvironmentName {
		if "" == serviceDefaultEnvironmentName {
			return result
		} else {
			configName = serviceDefaultEnvironmentName
		}
	} else {
		configName = siteEnvironmentName
	}

	redisConfig := redisConfigs[configName]
	scheme := "tcp"
	if nil != redisConfig.Spec.IsTlsEnabled && *redisConfig.Spec.IsTlsEnabled {
		scheme = "tls"
	}
	result["database.redis.scheme"] = scheme
	result["database.redis.host"] = redisConfig.Spec.Host
	result["database.redis.port"] = fmt.Sprintf("%d", redisConfig.Spec.Port)
	result["database.redis.password"] = redisConfig.Spec.Password

	return result
}

func (r *SiteTemplateHandler) getCommonDatabaseConfigTemplateValues(serviceStatus sitev1.StagingSiteServiceStatus, serviceSpec sitev1.StagingSiteService) map[string]string {
	result := map[string]string{
		"database.username":       serviceStatus.Username,
		"database.name":           serviceStatus.DbName,
		"database.password":       r.site.Spec.Password,
		"database.redis.database": fmt.Sprintf("%d", serviceStatus.RedisDatabaseNumber),
		"database.initSource":     serviceSpec.DbInitSourceEnvironmentName,
	}

	return result
}
