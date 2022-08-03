package test_helper

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
)

type stubKubernetesManager struct {
	deletedYAMLs []string
	appliedYAMLs []string
	usedFilter   func(*unstructured.Unstructured) bool
}

/*
** Note: interface sections and functions
** are in the same order as ${REPO_ROOT}/deploy/internal/
** for readability and ease of refactor / extension.
**/
type StubKubernetesManager interface {
	/* Check if YAML matching checks was applied */
	AppliedContains(apiVersion, kind, appKubernetesIOName, appKubernetesIOComponent, metadataName string, otherChecks ...string) bool
	DeletedContains(apiVersion, kind, appKubernetesIOName, appKubernetesIOComponent, metadataName string, otherChecks ...string) bool

	// TODO: GetDeletedYAML for consistency
	GetAppliedYAML(apiVersion, kind, appKubernetesIOName, appKubernetesIOComponent, metadataName string, otherChecks ...string) (*unstructured.Unstructured, error)
	GetAppliedDeployment(appKubernetesIOComponent, metadataName string) (appsv1.Deployment, error)

	/* Contains helpers */
	CollectorServiceAccountContains(checks ...string) bool
	CollectorConfigMapContains(checks ...string) bool

	NodeCollectorDaemonSetContains(checks ...string) bool
	ClusterCollectorDeploymentContains(checks ...string) bool

	ProxyServiceContains(checks ...string) bool
	ProxyDeploymentContains(checks ...string) bool

	/* Get *unstructured.Unstructured objects for filter testing, etc. */
	GetUnstructuredCollectorServiceAccount() (*unstructured.Unstructured, error)
	GetUnstructuredCollectorConfigMap() (*unstructured.Unstructured, error)
	GetUnstructuredNodeCollectorDaemonSet() (*unstructured.Unstructured, error)
	GetUnstructuredClusterCollectorDeployment() (*unstructured.Unstructured, error)
	GetUnstructuredProxyService() (*unstructured.Unstructured, error)
	GetUnstructuredProxyDeployment() (*unstructured.Unstructured, error)

	/* Object getters for specific property testing */
	GetCollectorServiceAccount() (corev1.ServiceAccount, error)
	GetCollectorConfigMap() (corev1.ConfigMap, error)

	GetNodeCollectorDaemonSet() (appsv1.DaemonSet, error)
	GetClusterCollectorDeployment() (appsv1.Deployment, error)

	GetProxyService() (corev1.Service, error)
	GetProxyDeployment() (appsv1.Deployment, error)

	// TODO: pull all object filters into single test
	ObjectPassesFilter(object *unstructured.Unstructured) bool

	// TODO: remove now that I have easy getters
	ServiceAccountPassesFilter(t *testing.T, err error) bool
}

func NewStubKubernetesManager() *stubKubernetesManager {
	return &stubKubernetesManager{}
}

func (skm *stubKubernetesManager) ApplyResources(resourceYAMLs []string, filterObject func(*unstructured.Unstructured) bool) error {
	skm.appliedYAMLs = resourceYAMLs
	skm.usedFilter = filterObject
	return nil
}

func (skm *stubKubernetesManager) DeleteResources(resourceYAMLs []string) error {
	skm.deletedYAMLs = resourceYAMLs
	return nil
}

func (skm stubKubernetesManager) AppliedContains(
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

func (skm stubKubernetesManager) DeletedContains(
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

func (skm stubKubernetesManager) GetAppliedYAML(
	apiVersion,
	kind,
	appKubernetesIOName,
	appKubernetesIOComponent,
	metadataName string,
	otherChecks ...string,
) (*unstructured.Unstructured, error) {
	reg, err := k8sYAMLHeader(apiVersion, kind, appKubernetesIOName, appKubernetesIOComponent, metadataName)
	if err != nil {
		return nil, err
	}

	for _, yamlStr := range skm.appliedYAMLs {
		if reg.MatchString(yamlStr) {
			for _, other := range otherChecks {
				if !strings.Contains(yamlStr, other) {
					return nil, errors.New("no YAML matched conditions passed")
				}
			}
			object := &unstructured.Unstructured{}
			var resourceDecoder = yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
			_, _, err := resourceDecoder.Decode([]byte(yamlStr), nil, object)
			return object, err
		}
	}
	return nil, nil
}

func (skm stubKubernetesManager) GetAppliedServiceAccount(appKubernetesIOComponent, metadataName string) (corev1.ServiceAccount, error) {
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

func (skm stubKubernetesManager) GetAppliedConfigMap(appKubernetesIOComponent, metadataName string) (corev1.ConfigMap, error) {
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

func (skm stubKubernetesManager) GetAppliedDaemonSet(appKubernetesIOComponent, metadataName string) (appsv1.DaemonSet, error) {
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

func (skm stubKubernetesManager) GetAppliedDeployment(appKubernetesIOComponent, metadataName string) (appsv1.Deployment, error) {
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

func (skm stubKubernetesManager) GetAppliedService(appKubernetesIOComponent, metadataName string) (corev1.Service, error) {
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

func (skm stubKubernetesManager) CollectorServiceAccountContains(checks ...string) bool {
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

func (skm stubKubernetesManager) CollectorConfigMapContains(checks ...string) bool {
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

func (skm stubKubernetesManager) NodeCollectorDaemonSetContains(checks ...string) bool {
	return contains(
		skm.appliedYAMLs,
		"apps/v1",
		"DaemonSet",
		"wavefront",
		"collector",
		"wavefront-node-collector",
		checks...,
	)
}

func (skm stubKubernetesManager) ClusterCollectorDeploymentContains(checks ...string) bool {
	return contains(
		skm.appliedYAMLs,
		"apps/v1",
		"Deployment",
		"wavefront",
		"collector",
		"wavefront-cluster-collector",
		checks...,
	)
}

func (skm stubKubernetesManager) ProxyServiceContains(checks ...string) bool {
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

func (skm stubKubernetesManager) ProxyDeploymentContains(checks ...string) bool {
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

func (skm stubKubernetesManager) GetUnstructuredCollectorServiceAccount() (*unstructured.Unstructured, error) {
	return skm.GetAppliedYAML(
		"v1",
		"ServiceAccount",
		"wavefront",
		"collector",
		"wavefront-collector",
	)
}

func (skm stubKubernetesManager) GetUnstructuredCollectorConfigMap() (*unstructured.Unstructured, error) {
	return skm.GetAppliedYAML(
		"v1",
		"ConfigMap",
		"wavefront",
		"collector",
		"default-wavefront-collector-config",
	)
}

func (skm stubKubernetesManager) GetUnstructuredNodeCollectorDaemonSet() (*unstructured.Unstructured, error) {
	return skm.GetAppliedYAML(
		"apps/v1",
		"DaemonSet",
		"wavefront",
		"collector",
		"wavefront-node-collector",
	)
}

func (skm stubKubernetesManager) GetUnstructuredClusterCollectorDeployment() (*unstructured.Unstructured, error) {
	return skm.GetAppliedYAML(
		"apps/v1",
		"Deployment",
		"wavefront",
		"collector",
		"wavefront-cluster-collector",
	)
}

func (skm stubKubernetesManager) GetUnstructuredProxyService() (*unstructured.Unstructured, error) {
	return skm.GetAppliedYAML(
		"v1",
		"Service",
		"wavefront",
		"proxy",
		"wavefront-proxy",
	)
}

func (skm stubKubernetesManager) GetUnstructuredProxyDeployment() (*unstructured.Unstructured, error) {
	return skm.GetAppliedYAML(
		"apps/v1",
		"Deployment",
		"wavefront",
		"proxy",
		"wavefront-proxy",
	)
}

func (skm stubKubernetesManager) GetCollectorServiceAccount() (corev1.ServiceAccount, error) {
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

func (skm stubKubernetesManager) GetCollectorConfigMap() (corev1.ConfigMap, error) {
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

func (skm stubKubernetesManager) GetNodeCollectorDaemonSet() (appsv1.DaemonSet, error) {
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

func (skm stubKubernetesManager) GetClusterCollectorDeployment() (appsv1.Deployment, error) {
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

func (skm stubKubernetesManager) GetProxyService() (corev1.Service, error) {
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

func (skm stubKubernetesManager) GetProxyDeployment() (appsv1.Deployment, error) {
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

func (skm stubKubernetesManager) ObjectPassesFilter(object *unstructured.Unstructured) bool {
	// TODO: filter returning true if filtered is confusing
	return !skm.usedFilter(object)
}

func (skm stubKubernetesManager) ServiceAccountPassesFilter(t *testing.T, err error) bool {
	serviceAccountObject, err := skm.GetUnstructuredCollectorServiceAccount()
	assert.NoError(t, err)

	// TODO: NOTE: the filter is only based on app.kubernetes.io/component value
	return skm.ObjectPassesFilter(
		serviceAccountObject,
	)
}

func k8sYAMLHeader(apiVersion string, kind string, appKubernetesIOName string, appKubernetesIOComponent string, metadataName string) (*regexp.Regexp, error) {
	headerMatchStr := fmt.Sprintf(
		`apiVersion: %s
kind: %s
metadata:
  labels:
    app.kubernetes.io/name: %s
    app.kubernetes.io/component: %s
  name: %s`,
		apiVersion, kind, appKubernetesIOName, appKubernetesIOComponent, metadataName)

	reg, err := regexp.Compile(headerMatchStr)
	return reg, err
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
	reg, err := k8sYAMLHeader(apiVersion, kind, appKubernetesIOName, appKubernetesIOComponent, metadataName)
	if err != nil {
		panic(err)
	}

	for _, yamlStr := range yamls {
		if reg.MatchString(yamlStr) {
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
