package testhelper

import (
	"errors"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
)

type MockKubernetesManager struct {
	deletedYAMLs []string
	appliedYAMLs []string
	usedFilter   func(*unstructured.Unstructured) bool
}

func NewMockKubernetesManager() *MockKubernetesManager {
	return &MockKubernetesManager{}
}

func (skm *MockKubernetesManager) ApplyResources(resourceYAMLs []string, filterObject func(*unstructured.Unstructured) bool) error {
	skm.appliedYAMLs = resourceYAMLs
	skm.usedFilter = filterObject
	return nil
}

func (skm *MockKubernetesManager) DeleteResources(resourceYAMLs []string) error {
	skm.deletedYAMLs = resourceYAMLs
	return nil
}

func (skm MockKubernetesManager) AppliedContains(
	apiVersion,
	kind,
	appKubernetesIOName,
	appKubernetesIOComponent,
	metadataName string,
	otherChecks ...string,
) bool {
	return contains(
		skm.appliedYAMLs,
		apiVersion,
		kind,
		appKubernetesIOName,
		appKubernetesIOComponent,
		metadataName,
		otherChecks...,
	)
}

func (skm MockKubernetesManager) DeletedContains(
	apiVersion,
	kind,
	appKubernetesIOName,
	appKubernetesIOComponent,
	metadataName string,
	otherChecks ...string,
) bool {
	return contains(
		skm.deletedYAMLs,
		apiVersion,
		kind,
		appKubernetesIOName,
		appKubernetesIOComponent,
		metadataName,
		otherChecks...,
	)
}

func (skm MockKubernetesManager) GetAppliedYAML(
	apiVersion,
	kind,
	appKubernetesIOName,
	appKubernetesIOComponent,
	metadataName string,
	otherChecks ...string,
) (*unstructured.Unstructured, error) {
	for _, yamlStr := range skm.appliedYAMLs {
		object, err := unstructuredFromStr(yamlStr)
		if err != nil {
			return nil, err
		}

		if objectMatchesAll(
			object,
			apiVersion,
			kind,
			appKubernetesIOName,
			appKubernetesIOComponent,
			metadataName,
		) {
			for _, other := range otherChecks {
				if !strings.Contains(yamlStr, other) {
					return nil, errors.New("no YAML matched conditions passed")
				}
			}
			return object, err
		}
	}
	return nil, nil
}

func objectMatchesAll(
	object *unstructured.Unstructured,
	apiVersion string,
	kind string,
	appKubernetesIOName string,
	appKubernetesIOComponent string,
	metadataName string,
) bool {
	if object.Object["apiVersion"] != apiVersion {
		return false
	}

	if object.Object["kind"] != kind {
		return false
	}

	objectAppK8sIOName, found, err := unstructured.NestedString(object.Object, "metadata", "labels", "app.kubernetes.io/name")
	if objectAppK8sIOName != appKubernetesIOName || !found || err != nil {
		return false
	}

	objectAppK8sIOComponent, found, err := unstructured.NestedString(object.Object, "metadata", "labels", "app.kubernetes.io/component")
	if objectAppK8sIOComponent != appKubernetesIOComponent || !found || err != nil {
		return false
	}

	objectMetadataName, found, err := unstructured.NestedString(object.Object, "metadata", "name")
	if objectMetadataName != metadataName || !found || err != nil {
		return false
	}
	return true
}

func unstructuredFromStr(yamlStr string) (*unstructured.Unstructured, error) {
	object := &unstructured.Unstructured{}
	var resourceDecoder = yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	_, _, err := resourceDecoder.Decode([]byte(yamlStr), nil, object)
	return object, err
}

func (skm MockKubernetesManager) GetAppliedServiceAccount(appKubernetesIOComponent, metadataName string) (corev1.ServiceAccount, error) {
	yamlUnstructured, err := skm.GetAppliedYAML(
		"v1",
		"ServiceAccount",
		"wavefront",
		appKubernetesIOComponent,
		metadataName,
	)
	if err != nil {
		return corev1.ServiceAccount{}, err
	}

	var serviceAccount corev1.ServiceAccount
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(yamlUnstructured.Object, &serviceAccount)
	if err != nil {
		return corev1.ServiceAccount{}, err
	}

	return serviceAccount, nil
}

func (skm MockKubernetesManager) GetAppliedConfigMap(appKubernetesIOComponent, metadataName string) (corev1.ConfigMap, error) {
	yamlUnstructured, err := skm.GetAppliedYAML(
		"v1",
		"ConfigMap",
		"wavefront",
		appKubernetesIOComponent,
		metadataName,
	)
	if err != nil {
		return corev1.ConfigMap{}, err
	}

	var configMap corev1.ConfigMap
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(yamlUnstructured.Object, &configMap)
	if err != nil {
		return corev1.ConfigMap{}, err
	}

	return configMap, nil
}

func (skm MockKubernetesManager) GetAppliedDaemonSet(appKubernetesIOComponent, metadataName string) (appsv1.DaemonSet, error) {
	yamlUnstructured, err := skm.GetAppliedYAML(
		"apps/v1",
		"DaemonSet",
		"wavefront",
		appKubernetesIOComponent,
		metadataName,
	)
	if err != nil {
		return appsv1.DaemonSet{}, err
	}

	var daemonSet appsv1.DaemonSet
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(yamlUnstructured.Object, &daemonSet)
	if err != nil {
		return appsv1.DaemonSet{}, err
	}

	return daemonSet, nil
}

func (skm MockKubernetesManager) GetAppliedDeployment(appKubernetesIOComponent, metadataName string) (appsv1.Deployment, error) {
	yamlUnstructured, err := skm.GetAppliedYAML(
		"apps/v1",
		"Deployment",
		"wavefront",
		appKubernetesIOComponent,
		metadataName,
	)
	if err != nil {
		return appsv1.Deployment{}, err
	}

	var deployment appsv1.Deployment
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(yamlUnstructured.Object, &deployment)
	if err != nil {
		return appsv1.Deployment{}, err
	}

	return deployment, nil
}

func (skm MockKubernetesManager) GetAppliedService(appKubernetesIOComponent, metadataName string) (corev1.Service, error) {
	yamlUnstructured, err := skm.GetAppliedYAML(
		"v1",
		"Service",
		"wavefront",
		appKubernetesIOComponent,
		metadataName,
	)
	if err != nil {
		return corev1.Service{}, err
	}

	var service corev1.Service
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(yamlUnstructured.Object, &service)
	if err != nil {
		return corev1.Service{}, err
	}

	return service, nil
}

func (skm MockKubernetesManager) CollectorServiceAccountContains(checks ...string) bool {
	return contains(
		skm.appliedYAMLs,
		"v1",
		"ServiceAccount",
		"wavefront",
		"collector",
		"wavefront-collector",
		checks...,
	)
}

func (skm MockKubernetesManager) CollectorConfigMapContains(checks ...string) bool {
	return contains(
		skm.appliedYAMLs,
		"v1",
		"ConfigMap",
		"wavefront",
		"collector",
		"default-wavefront-collector-config",
		checks...,
	)
}

func (skm MockKubernetesManager) NodeCollectorDaemonSetContains(checks ...string) bool {
	return contains(
		skm.appliedYAMLs,
		"apps/v1",
		"DaemonSet",
		"wavefront",
		"node-collector",
		"wavefront-node-collector",
		checks...,
	)
}

func (skm MockKubernetesManager) LoggingDaemonSetContains(checks ...string) bool {
	return contains(
		skm.appliedYAMLs,
		"apps/v1",
		"DaemonSet",
		"wavefront",
		"logging",
		"wavefront-logging",
		checks...,
	)
}

func (skm MockKubernetesManager) LoggingConfigMapContains(checks ...string) bool {
	return contains(
		skm.appliedYAMLs,
		"v1",
		"ConfigMap",
		"wavefront",
		"logging",
		"wavefront-logging-config",
		checks...,
	)
}

func (skm MockKubernetesManager) ClusterCollectorDeploymentContains(checks ...string) bool {
	return contains(
		skm.appliedYAMLs,
		"apps/v1",
		"Deployment",
		"wavefront",
		"cluster-collector",
		"wavefront-cluster-collector",
		checks...,
	)
}

func (skm MockKubernetesManager) ProxyServiceContains(checks ...string) bool {
	return contains(
		skm.appliedYAMLs,
		"v1",
		"Service",
		"wavefront",
		"proxy",
		"wavefront-proxy",
		checks...,
	)
}

func (skm MockKubernetesManager) ProxyDeploymentContains(checks ...string) bool {
	return contains(
		skm.appliedYAMLs,
		"apps/v1",
		"Deployment",
		"wavefront",
		"proxy",
		"wavefront-proxy",
		checks...,
	)
}

func (skm MockKubernetesManager) GetUnstructuredCollectorServiceAccount() (*unstructured.Unstructured, error) {
	return skm.GetAppliedYAML(
		"v1",
		"ServiceAccount",
		"wavefront",
		"collector",
		"wavefront-collector",
	)
}

func (skm MockKubernetesManager) GetUnstructuredCollectorConfigMap() (*unstructured.Unstructured, error) {
	return skm.GetAppliedYAML(
		"v1",
		"ConfigMap",
		"wavefront",
		"collector",
		"default-wavefront-collector-config",
	)
}

func (skm MockKubernetesManager) GetUnstructuredNodeCollectorDaemonSet() (*unstructured.Unstructured, error) {
	return skm.GetAppliedYAML(
		"apps/v1",
		"DaemonSet",
		"wavefront",
		"collector",
		"wavefront-node-collector",
	)
}

func (skm MockKubernetesManager) GetUnstructuredClusterCollectorDeployment() (*unstructured.Unstructured, error) {
	return skm.GetAppliedYAML(
		"apps/v1",
		"Deployment",
		"wavefront",
		"collector",
		"wavefront-cluster-collector",
	)
}

func (skm MockKubernetesManager) GetUnstructuredProxyService() (*unstructured.Unstructured, error) {
	return skm.GetAppliedYAML(
		"v1",
		"Service",
		"wavefront",
		"proxy",
		"wavefront-proxy",
	)
}

func (skm MockKubernetesManager) GetUnstructuredProxyDeployment() (*unstructured.Unstructured, error) {
	return skm.GetAppliedYAML(
		"apps/v1",
		"Deployment",
		"wavefront",
		"proxy",
		"wavefront-proxy",
	)
}

func (skm MockKubernetesManager) GetUnstructuredLoggingDaemonset() (*unstructured.Unstructured, error) {
	return skm.GetAppliedYAML(
		"apps/v1",
		"DaemonSet",
		"wavefront",
		"logging",
		"wavefront-logging",
	)
}

func (skm MockKubernetesManager) GetCollectorServiceAccount() (corev1.ServiceAccount, error) {
	yamlUnstructured, err := skm.GetUnstructuredCollectorServiceAccount()
	if err != nil {
		return corev1.ServiceAccount{}, err
	}

	var serviceAccount corev1.ServiceAccount
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(yamlUnstructured.Object, &serviceAccount)
	if err != nil {
		return corev1.ServiceAccount{}, err
	}

	return serviceAccount, nil
}

func (skm MockKubernetesManager) GetCollectorConfigMap() (corev1.ConfigMap, error) {
	yamlUnstructured, err := skm.GetUnstructuredCollectorConfigMap()
	if err != nil {
		return corev1.ConfigMap{}, err
	}

	var configMap corev1.ConfigMap
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(yamlUnstructured.Object, &configMap)
	if err != nil {
		return corev1.ConfigMap{}, err
	}

	return configMap, nil
}

func (skm MockKubernetesManager) GetNodeCollectorDaemonSet() (appsv1.DaemonSet, error) {
	yamlUnstructured, err := skm.GetUnstructuredNodeCollectorDaemonSet()
	if err != nil {
		return appsv1.DaemonSet{}, err
	}

	var daemonSet appsv1.DaemonSet
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(yamlUnstructured.Object, &daemonSet)
	if err != nil {
		return appsv1.DaemonSet{}, err
	}

	return daemonSet, nil
}

func (skm MockKubernetesManager) GetClusterCollectorDeployment() (appsv1.Deployment, error) {
	yamlUnstructured, err := skm.GetUnstructuredClusterCollectorDeployment()
	if err != nil {
		return appsv1.Deployment{}, err
	}

	var deployment appsv1.Deployment
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(yamlUnstructured.Object, &deployment)
	if err != nil {
		return appsv1.Deployment{}, err
	}

	return deployment, nil
}

func (skm MockKubernetesManager) GetProxyService() (corev1.Service, error) {
	yamlUnstructured, err := skm.GetUnstructuredProxyService()
	if err != nil {
		return corev1.Service{}, err
	}

	var service corev1.Service
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(yamlUnstructured.Object, &service)
	if err != nil {
		return corev1.Service{}, err
	}

	return service, nil
}

func (skm MockKubernetesManager) GetProxyDeployment() (appsv1.Deployment, error) {
	yamlUnstructured, err := skm.GetUnstructuredProxyDeployment()
	if err != nil {
		return appsv1.Deployment{}, err
	}

	var deployment appsv1.Deployment
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(yamlUnstructured.Object, &deployment)
	if err != nil {
		return appsv1.Deployment{}, err
	}

	return deployment, nil
}

func (skm MockKubernetesManager) ObjectPassesFilter(object *unstructured.Unstructured) bool {
	// TODO: filter returning true if filtered is confusing
	return !skm.usedFilter(object)
}

func contains(
	yamls []string,
	apiVersion,
	kind,
	appKubernetesIOName,
	appKubernetesIOComponent,
	metadataName string,
	otherChecks ...string,
) bool {
	for _, yamlStr := range yamls {
		object, err := unstructuredFromStr(yamlStr)
		if err != nil {
			panic(err)
		}

		if objectMatchesAll(
			object,
			apiVersion,
			kind,
			appKubernetesIOName,
			appKubernetesIOComponent,
			metadataName,
		) {
			for _, other := range otherChecks {
				if !strings.Contains(yamlStr, other) {
					return false
				}
			}
			return true
		}
	}

	return false
}
