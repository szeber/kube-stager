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
	SetMysql(config configv1.MysqlConfig)
	SetMongo(config configv1.MongoConfig)
	SetRedis(config configv1.RedisConfig)
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
	mysqlConfig          configv1.MysqlConfig
	mongoConfig          configv1.MongoConfig
	redisConfig          configv1.RedisConfig
}

func NewSite(site sitev1.StagingSite, serviceConfig configv1.ServiceConfig) SiteTemplateHandler {
	return SiteTemplateHandler{
		site:                 site,
		siteServiceSpec:      site.Spec.Services[serviceConfig.Name],
		siteServiceStatus:    site.Status.Services[serviceConfig.Name],
		currentServiceConfig: serviceConfig,
	}
}

func LoadConfigs(
	handler DatabaseHandler,
	ctx context.Context,
	reader client.Reader,
	mysqlConfigName string,
	mongoConfigName string,
	redisConfigName string,
) error {
	namespace := handler.getNamespace()

	if "" != mongoConfigName {
		config := configv1.MongoConfig{}
		if err := reader.Get(ctx, client.ObjectKey{Namespace: namespace, Name: mongoConfigName}, &config); nil != err {
			return err
		}
		handler.SetMongo(config)
	}
	if "" != mysqlConfigName {
		config := configv1.MysqlConfig{}
		if err := reader.Get(ctx, client.ObjectKey{Namespace: namespace, Name: mysqlConfigName}, &config); nil != err {
			return err
		}
		handler.SetMysql(config)
	}
	if "" != redisConfigName {
		config := configv1.RedisConfig{}
		if err := reader.Get(ctx, client.ObjectKey{Namespace: namespace, Name: redisConfigName}, &config); nil != err {
			return err
		}
		handler.SetRedis(config)
	}

	return LoadServiceConfigs(handler, ctx, reader)
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

func (r *SiteTemplateHandler) SetMysql(config configv1.MysqlConfig) {
	r.mysqlConfig = config
}

func (r *SiteTemplateHandler) SetMongo(config configv1.MongoConfig) {
	r.mongoConfig = config
}

func (r *SiteTemplateHandler) SetRedis(config configv1.RedisConfig) {
	r.redisConfig = config
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

	if "" != r.mysqlConfig.Name {
		result["database.mysql.host"] = r.mysqlConfig.Spec.Host
		result["database.mysql.port"] = fmt.Sprintf("%d", r.mysqlConfig.Spec.Port)
	}

	if "" != r.mongoConfig.Name {
		result["database.mongo.host1"] = r.mongoConfig.Spec.Host1
		result["database.mongo.host2"] = r.mongoConfig.Spec.Host2
		result["database.mongo.host3"] = r.mongoConfig.Spec.Host3
		result["database.mongo.port"] = fmt.Sprintf("%d", r.mongoConfig.Spec.Port)
	}

	if "" != r.redisConfig.Name {
		scheme := "tcp"
		if nil != r.redisConfig.Spec.IsTlsEnabled && *r.redisConfig.Spec.IsTlsEnabled {
			scheme = "tls"
		}
		result["database.redis.scheme"] = scheme
		result["database.redis.host"] = r.redisConfig.Spec.Host
		result["database.redis.port"] = fmt.Sprintf("%d", r.redisConfig.Spec.Port)
		result["database.redis.password"] = r.redisConfig.Spec.Password
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
	}

	return result
}
