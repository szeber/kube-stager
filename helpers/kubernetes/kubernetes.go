package kubernetes

import (
	"context"
	configv1 "github.com/szeber/kube-stager/api/config/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetServiceConfigsInNamespace(
	namespace string,
	kubeClient client.Reader,
	ctx context.Context,
) (map[string]configv1.ServiceConfig, error) {
	result := make(map[string]configv1.ServiceConfig)
	var list configv1.ServiceConfigList

	for ok := true; ok; ok = (nil != list.RemainingItemCount && *list.RemainingItemCount > int64(0)) {
		listOptions := []client.ListOption{
			client.InNamespace(namespace),
		}
		if "" != list.Continue {
			listOptions = append(listOptions, client.Continue(list.Continue))
		}

		if err := kubeClient.List(ctx, &list, listOptions...); nil != err {
			return result, err
		}
		for _, config := range list.Items {
			result[config.Name] = config
		}
	}

	return result, nil
}

func GetMysqlEnvironmentsInNamespace(
	namespace string,
	kubeClient client.Reader,
	ctx context.Context,
) (map[string]configv1.MysqlConfig, error) {
	result := make(map[string]configv1.MysqlConfig)
	var list configv1.MysqlConfigList

	for ok := true; ok; ok = (nil != list.RemainingItemCount && *list.RemainingItemCount > int64(0)) {
		listOptions := []client.ListOption{
			client.InNamespace(namespace),
		}
		if "" != list.Continue {
			listOptions = append(listOptions, client.Continue(list.Continue))
		}
		if err := kubeClient.List(ctx, &list, listOptions...); nil != err {
			return result, err
		}
		for _, config := range list.Items {
			result[config.Name] = config
		}
	}

	return result, nil
}

func GetMongoEnvironmentsInNamespace(
	namespace string,
	kubeClient client.Reader,
	ctx context.Context,
) (map[string]configv1.MongoConfig, error) {
	result := make(map[string]configv1.MongoConfig)
	var list configv1.MongoConfigList

	for ok := true; ok; ok = (nil != list.RemainingItemCount && *list.RemainingItemCount > int64(0)) {
		listOptions := []client.ListOption{
			client.InNamespace(namespace),
		}
		if "" != list.Continue {
			listOptions = append(listOptions, client.Continue(list.Continue))
		}
		if err := kubeClient.List(ctx, &list, listOptions...); nil != err {
			return result, err
		}
		for _, config := range list.Items {
			result[config.Name] = config
		}
	}

	return result, nil
}

func GetRedisEnvironmentsInNamespace(
	namespace string,
	kubeClient client.Reader,
	ctx context.Context,
) (map[string]configv1.RedisConfig, error) {
	result := make(map[string]configv1.RedisConfig)
	var list configv1.RedisConfigList

	for ok := true; ok; ok = (nil != list.RemainingItemCount && *list.RemainingItemCount > int64(0)) {
		listOptions := []client.ListOption{
			client.InNamespace(namespace),
		}
		if "" != list.Continue {
			listOptions = append(listOptions, client.Continue(list.Continue))
		}
		if err := kubeClient.List(ctx, &list, listOptions...); nil != err {
			return result, err
		}
		for _, config := range list.Items {
			result[config.Name] = config
		}
	}

	return result, nil
}
