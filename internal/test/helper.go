package test_helper

import (
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"regexp"
	"strings"
	"testing"
)

type stubKubernetesManager struct {
	deletedYAMLs []string
	appliedYAMLs []string
	usedFilter   func(*unstructured.Unstructured) bool
}

type StubKubernetesManager interface {
	/* base Contains func's */
	DeletedContains(apiVersion, kind, appKubernetesIOName, appKubernetesIOComponent, metadataName string, otherChecks ...string) bool
	AppliedContains(apiVersion, kind, appKubernetesIOName, appKubernetesIOComponent, metadataName string, otherChecks ...string) bool

	// TODO: ProxyServiceContains, ...
	ConfigMapContains(checks ...string) bool
	ProxyDeploymentContains(checks ...string) bool
	NodeCollectorDaemonSetContains(checks ...string) bool
	ClusterCollectorDeploymentContains(checks ...string) bool

	// TODO: GetAppliedDaemonset, ...
	GetAppliedYAML(apiVersion, kind, appKubernetesIOName, appKubernetesIOComponent, metadataName string, otherChecks ...string) (*unstructured.Unstructured, error)
	GetAppliedDeployment(appKubernetesIOComponent, metadataName string) (v1.Deployment, error)

	// TODO: DeploymentPassesFilter, ...
	ObjectPassesFilter(object *unstructured.Unstructured) bool
	ServiceAccountPassesFilter(t *testing.T, err error) bool
}

func NewStubKubernetesManager() *stubKubernetesManager {
	return &stubKubernetesManager{}
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

func (skm stubKubernetesManager) ObjectPassesFilter(object *unstructured.Unstructured) bool {
	// TODO: filter returning true if filtered is confusing
	return !skm.usedFilter(object)
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

func (skm stubKubernetesManager) GetAppliedDeployment(appKubernetesIOComponent, metadataName string) (v1.Deployment, error) {
	deploymentYAMLUnstructured, err := skm.GetAppliedYAML(
		"apps/v1",
		"Deployment",
		"wavefront",
		appKubernetesIOComponent,
		metadataName,
	)
	if err != nil {
		return v1.Deployment{}, err
	}

	var deployment v1.Deployment
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(deploymentYAMLUnstructured.Object, &deployment)
	if err != nil {
		return v1.Deployment{}, err
	}

	return deployment, nil
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

func (skm stubKubernetesManager) ConfigMapContains(checks ...string) bool {
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

func (skm stubKubernetesManager) ServiceAccountPassesFilter(t *testing.T, err error) bool {
	serviceAccountYAML := `
apiVersion: v1
kind: ServiceAccount
metadata:
  labels:
    app.kubernetes.io/name: wavefront
    app.kubernetes.io/component: collector
  name: wavefront-collector
  namespace: wavefront
`
	var resourceDecoder = yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)

	serviceAccountObject := &unstructured.Unstructured{}
	_, _, err = resourceDecoder.Decode([]byte(serviceAccountYAML), nil, serviceAccountObject)
	assert.NoError(t, err)

	// TODO: NOTE: the filter is only based on app.kubernetes.io/component value
	// so I only tested one object
	return skm.ObjectPassesFilter(
		serviceAccountObject,
	)
}
