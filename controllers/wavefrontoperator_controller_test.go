package controllers_test

import (
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	wavefrontcomv1 "github.com/wavefrontHQ/wavefront-operator-for-kubernetes/api/v1"
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/controllers"
	"io/fs"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"log"
	"os"
	"testing"
	"testing/fstest"
	"time"
)

func TestKubernetesFilePaths(t *testing.T) {
	emptyDir, err := ioutil.TempDir("", "")
	if err != nil {
		log.Fatal(err)
	}

	fileDir, err := ioutil.TempDir("", "temp")
	if err != nil {
		log.Fatal(err)
	}
	ioutil.TempFile(fileDir, "*.yaml")
	ioutil.TempFile(fileDir, "*.txt")

	defer os.RemoveAll(emptyDir)
	defer os.RemoveAll(fileDir)

	tests := []struct {
		name string
		dir  string
		want int
		err  error
	}{
		{"Invalid directory", "/invalidDir", 0, errors.New("no such file or directory")},
		{"Empty directory", emptyDir, 0, nil},
		{"Directory with txt and yaml files", fileDir, 1, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := controllers.ResourceFiles(tt.dir)
			assert.Equal(t, tt.want, len(got))
			if err != nil {
				assert.Contains(t, err.Error(), tt.err.Error())
			}
		})
	}
}

func TestReadAndInterpolateResources(t *testing.T) {
	t.Run("Interpolate multiple files", func(t *testing.T) {
		spec := wavefrontcomv1.WavefrontOperatorSpec{
			WavefrontUrl:    "fake-cluster-name",
			WavefrontToken: "fake-token",
		}
		fakeFiles := fstest.MapFS{
			"proxy.yaml": &fstest.MapFile{
				Data:    []byte("whatIsNameProxy: {{.ClusterName}}"),
				Mode:    fs.ModePerm,
				ModTime: time.Now(),
				Sys:     nil,
			},
			"config-map.yaml": &fstest.MapFile{
				Data:    []byte("whatIsNameConfig: {{.ClusterName}}"),
				Mode:    fs.ModePerm,
				ModTime: time.Now(),
				Sys:     nil,
			},
			"collector.yaml": &fstest.MapFile{
				Data:    []byte("whatIsNameCollector: {{.ClusterName}}"),
				Mode:    fs.ModePerm,
				ModTime: time.Now(),
				Sys:     nil,
			},
		}
		resources, _ := controllers.ReadAndInterpolateResources(fakeFiles, spec, []string{"proxy.yaml", "config-map.yaml", "collector.yaml"})
		assert.Equal(t, resources[0], "whatIsNameProxy: fake-cluster-name")
		assert.Equal(t, resources[1], "whatIsNameConfig: fake-cluster-name")
		assert.Equal(t, resources[2], "whatIsNameCollector: fake-cluster-name")
	})

	t.Run("Handles non-parsable templates", func(t *testing.T) {
		spec := wavefrontcomv1.WavefrontOperatorSpec{
			WavefrontUrl:    "fake-cluster-name",
			WavefrontToken: "fake-token",
		}
		emptyFS := fstest.MapFS{}
		_, err := controllers.ReadAndInterpolateResources(emptyFS, spec, []string{"some.yaml"})
		assert.Error(t, err, "Expected template error")
	})

	t.Run("Handles non-executable templates", func(t *testing.T) {
		spec := wavefrontcomv1.WavefrontOperatorSpec{
			WavefrontUrl:    "fake-cluster-name",
			WavefrontToken: "fake-token",
		}
		fakeFiles := fstest.MapFS{
			"some.yaml": &fstest.MapFile{
				Data:    []byte("someKey: {{.NonExistentField}}"),
				Mode:    fs.ModePerm,
				ModTime: time.Now(),
				Sys:     nil,
			},
		}
		_, err := controllers.ReadAndInterpolateResources(fakeFiles, spec, []string{"some.yaml"})
		assert.Error(t, err, "Expected execution error")
	})
}

func TestCreateKubernetesObjects(t *testing.T) {
	t.Run("Create multiple Kubernetes objects from multiple resources", func(t *testing.T) {
		resources := []string{"{\"kind\": \"Deployment\", \"resource\": \"resource-one\"}",
			"{\"kind\": \"Deployment\", \"resource\": \"resource-two\"}"}
		resourceOneObject := unstructured.Unstructured{
			Object: map[string]interface{}{
				"kind":     "Deployment",
				"resource": "resource-one",
			},
		}
		resourceTwoObject := unstructured.Unstructured{
			Object: map[string]interface{}{
				"kind":     "Deployment",
				"resource": "resource-two",
			},
		}
		actualObjects, _ := controllers.InitializeKubernetesObjects(resources)
		assert.Contains(t, actualObjects, resourceOneObject)
		assert.Contains(t, actualObjects, resourceTwoObject)
	})
	t.Run("Invalid resource json errors", func(t *testing.T) {
		resources := []string{"{\"kind\":: \"Deployment\"}"}
		_, err := controllers.InitializeKubernetesObjects(resources)
		assert.Error(t, err, "Expecting json error")
		t.Log(err)
	})
}

func TestProvisionProxy(t *testing.T) {
	createResource := func(objects []unstructured.Unstructured) (error) {
		fmt.Printf("k8s objects :: %+v", objects)
		assert.Equal(t,"wavefront-proxy", objects[0].GetName())
		assert.Equal(t,"Deployment", objects[0].GetKind())
		//TODO: Downcast object into deployment inorder to verify templatized spec
		return nil
	}
	t.Run("Test wavefront proxy spec templating", func(t *testing.T) {
		spec := wavefrontcomv1.WavefrontOperatorSpec{
			WavefrontUrl:    "fake-cluster-name",
			WavefrontToken: "fake-token",
		}
		r := controllers.WavefrontOperatorReconciler{FS:     os.DirFS("../deploy")}
		err := r.ProvisionProxy(spec, createResource)
		assert.NoError(t, err, "Expected no error")
		t.Log(err)
	})

	t.Run("Test initializing kubernetes objects", func(t *testing.T) {
		spec := wavefrontcomv1.WavefrontOperatorSpec{
			WavefrontUrl:    "fake-cluster-name",
			WavefrontToken: "fake-token",
		}
		r := controllers.WavefrontOperatorReconciler{FS:     os.DirFS("../deploy")}
		err := r.ProvisionProxy(spec, createResource)
		assert.NoError(t, err, "Expected no error")
		t.Log(err)
	})
}


