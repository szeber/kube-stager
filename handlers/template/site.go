package template

import (
	"context"
	"fmt"
	api "github.com/szeber/kube-stager/api"
	configv1 "github.com/szeber/kube-stager/api/config/v1"
	sitev1 "github.com/szeber/kube-stager/api/site/v1"
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
		"site.name":               r.site.Name,
		"site.domainPrefix":       r.site.Spec.DomainPrefix,
		"site.imageTag":           r.siteServiceSpec.ImageTag,
		"database.username":       r.siteServiceStatus.Username,
		"database.name":           r.siteServiceStatus.DbName,
		"database.password":       r.site.Spec.Password,
		"database.redis.database": fmt.Sprintf("%d", r.siteServiceStatus.RedisDatabaseNumber),
		"database.initSource":     r.siteServiceSpec.DbInitSourceEnvironmentName,
	}

	if "" != r.siteServiceSpec.MysqlEnvironment {
		for k, v := range r.getMysqlConfigTemplateValues(r.mysqlConfigs[r.siteServiceSpec.MysqlEnvironment]) {
			result[k] = v
		}
	}

	if "" != r.siteServiceSpec.MongoEnvironment {
		for k, v := range r.getMongoConfigTemplateValues(r.mongoConfigs[r.siteServiceSpec.MongoEnvironment]) {
			result[k] = v
		}
	}

	if "" != r.siteServiceSpec.RedisEnvironment {
		for k, v := range r.getRedisConfigTemplateValues(r.redisConfigs[r.siteServiceSpec.RedisEnvironment]) {
			result[k] = v
		}
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
		if "" != r.site.Spec.Services[name].MysqlEnvironment {
			for k, v := range r.getMysqlConfigTemplateValues(r.mysqlConfigs[r.site.Spec.Services[name].MysqlEnvironment]) {
				result[fmt.Sprintf("service.%s.%s", name, k)] = v
			}
		}
		if "" != r.site.Spec.Services[name].MongoEnvironment {
			for k, v := range r.getMongoConfigTemplateValues(r.mongoConfigs[r.site.Spec.Services[name].MongoEnvironment]) {
				result[fmt.Sprintf("service.%s.%s", name, k)] = v
			}
		}
		if "" != r.site.Spec.Services[name].RedisEnvironment {
			for k, v := range r.getRedisConfigTemplateValues(r.redisConfigs[r.site.Spec.Services[name].RedisEnvironment]) {
				result[fmt.Sprintf("service.%s.%s", name, k)] = v
			}
		}
	}

	return result
}

func (r *SiteTemplateHandler) getMysqlConfigTemplateValues(mysqlConfig configv1.MysqlConfig) map[string]string {
	result := make(map[string]string)

	result["database.mysql.host"] = mysqlConfig.Spec.Host
	result["database.mysql.port"] = fmt.Sprintf("%d", mysqlConfig.Spec.Port)

	return result
}

func (r *SiteTemplateHandler) getMongoConfigTemplateValues(mongoConfig configv1.MongoConfig) map[string]string {
	result := make(map[string]string)

	result["database.mongo.host1"] = mongoConfig.Spec.Host1
	result["database.mongo.host2"] = mongoConfig.Spec.Host2
	result["database.mongo.host3"] = mongoConfig.Spec.Host3
	result["database.mongo.port"] = fmt.Sprintf("%d", mongoConfig.Spec.Port)

	return result
}

func (r *SiteTemplateHandler) getRedisConfigTemplateValues(redisConfig configv1.RedisConfig) map[string]string {
	result := make(map[string]string)

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
