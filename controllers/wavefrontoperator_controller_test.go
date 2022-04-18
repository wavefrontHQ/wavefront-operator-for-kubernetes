package controllers_test

import (
	"context"
	"github.com/stretchr/testify/assert"
	wavefrontcomv1 "github.com/wavefrontHQ/wavefront-operator-for-kubernetes/api/v1"
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/controllers"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/scheme"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"testing"
)

//
//func TestKubernetesFilePaths(t *testing.T) {
//	emptyDir, err := ioutil.TempDir("", "")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	fileDir, err := ioutil.TempDir("", "temp")
//	if err != nil {
//		log.Fatal(err)
//	}
//	ioutil.TempFile(fileDir, "*.yaml")
//	ioutil.TempFile(fileDir, "*.txt")
//
//	defer os.RemoveAll(emptyDir)
//	defer os.RemoveAll(fileDir)
//
//	tests := []struct {
//		name string
//		dir  string
//		want int
//		err  error
//	}{
//		{"Invalid directory", "/invalidDir", 0, errors.New("no such file or directory")},
//		{"Empty directory", emptyDir, 0, nil},
//		{"Directory with txt and yaml files", fileDir, 1, nil},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			got, err := controllers.ResourceFiles(tt.dir)
//			assert.Equal(t, tt.want, len(got))
//			if err != nil {
//				assert.Contains(t, err.Error(), tt.err.Error())
//			}
//		})
//	}
//}
//
//func TestReadAndInterpolateResources(t *testing.T) {
//	t.Run("Interpolate multiple files", func(t *testing.T) {
//		spec := wavefrontcomv1.WavefrontOperatorSpec{
//			WavefrontUrl:    "fake-cluster-name",
//			WavefrontToken: "fake-token",
//		}
//		fakeFiles := fstest.MapFS{
//			"proxy.yaml": &fstest.MapFile{
//				Data:    []byte("whatIsNameProxy: {{.ClusterName}}"),
//				Mode:    fs.ModePerm,
//				ModTime: time.Now(),
//				Sys:     nil,
//			},
//			"config-map.yaml": &fstest.MapFile{
//				Data:    []byte("whatIsNameConfig: {{.ClusterName}}"),
//				Mode:    fs.ModePerm,
//				ModTime: time.Now(),
//				Sys:     nil,
//			},
//			"collector.yaml": &fstest.MapFile{
//				Data:    []byte("whatIsNameCollector: {{.ClusterName}}"),
//				Mode:    fs.ModePerm,
//				ModTime: time.Now(),
//				Sys:     nil,
//			},
//		}
//		resources, _ := controllers.ReadAndInterpolateResources(fakeFiles, spec, []string{"proxy.yaml", "config-map.yaml", "collector.yaml"})
//		assert.Equal(t, resources[0], "whatIsNameProxy: fake-cluster-name")
//		assert.Equal(t, resources[1], "whatIsNameConfig: fake-cluster-name")
//		assert.Equal(t, resources[2], "whatIsNameCollector: fake-cluster-name")
//	})
//
//	t.Run("Handles non-parsable templates", func(t *testing.T) {
//		spec := wavefrontcomv1.WavefrontOperatorSpec{
//			WavefrontUrl:    "fake-cluster-name",
//			WavefrontToken: "fake-token",
//		}
//		emptyFS := fstest.MapFS{}
//		_, err := controllers.ReadAndInterpolateResources(emptyFS, spec, []string{"some.yaml"})
//		assert.Error(t, err, "Expected template error")
//	})
//
//	t.Run("Handles non-executable templates", func(t *testing.T) {
//		spec := wavefrontcomv1.WavefrontOperatorSpec{
//			WavefrontUrl:    "fake-cluster-name",
//			WavefrontToken: "fake-token",
//		}
//		fakeFiles := fstest.MapFS{
//			"some.yaml": &fstest.MapFile{
//				Data:    []byte("someKey: {{.NonExistentField}}"),
//				Mode:    fs.ModePerm,
//				ModTime: time.Now(),
//				Sys:     nil,
//			},
//		}
//		_, err := controllers.ReadAndInterpolateResources(fakeFiles, spec, []string{"some.yaml"})
//		assert.Error(t, err, "Expected execution error")
//	})
//}

//func TestCreateKubernetesObjects(t *testing.T) {
//	t.Run("Create multiple Kubernetes objects from multiple resources", func(t *testing.T) {
//		resources := []string{"{\"kind\": \"Deployment\", \"resource\": \"resource-one\"}",
//			"{\"kind\": \"Deployment\", \"resource\": \"resource-two\"}"}
//		resourceOneObject := unstructured.Unstructured{
//			Object: map[string]interface{}{
//				"kind":     "Deployment",
//				"resource": "resource-one",
//			},
//		}
//		resourceTwoObject := unstructured.Unstructured{
//			Object: map[string]interface{}{
//				"kind":     "Deployment",
//				"resource": "resource-two",
//			},
//		}
//		actualObjects, _ := controllers.InitializeKubernetesObjects(resources)
//		assert.Contains(t, actualObjects, resourceOneObject)
//		assert.Contains(t, actualObjects, resourceTwoObject)
//	})
//	t.Run("Invalid resource json errors", func(t *testing.T) {
//		resources := []string{"{\"kind\":: \"Deployment\"}"}
//		_, err := controllers.InitializeKubernetesObjects(resources)
//		assert.Error(t, err, "Expecting json error")
//		t.Log(err)
//	})
//}

//func TestProvisionProxy(t *testing.T) {
//	createResource := func(client dynamic.ResourceInterface, objects []unstructured.Unstructured) (error) {
//		fmt.Printf("k8s objects :: %+v", objects)
//		assert.Equal(t,"wavefront-proxy", objects[0].GetName())
//		assert.Equal(t,"Deployment", objects[0].GetKind())
//		//TODO: Downcast object into deployment inorder to verify templatized spec
//		return nil
//	}
//	t.Run("Test wavefront proxy spec templating", func(t *testing.T) {
//		spec := wavefrontcomv1.WavefrontOperatorSpec{
//			WavefrontUrl:    "fake-cluster-name",
//			WavefrontToken: "fake-token",
//		}
//		r := controllers.WavefrontOperatorReconciler{FS:     os.DirFS("../deploy")}
//		err := r.ProvisionProxy(spec, createResource)
//		assert.NoError(t, err, "Expected no error")
//		t.Log(err)
//	})
//
//	t.Run("Test initializing kubernetes objects", func(t *testing.T) {
//		spec := wavefrontcomv1.WavefrontOperatorSpec{
//			WavefrontUrl:    "fake-cluster-name",
//			WavefrontToken: "fake-token",
//		}
//		r := controllers.WavefrontOperatorReconciler{FS:     os.DirFS("../deploy")}
//		err := r.ProvisionProxy(spec, createResource)
//		assert.NoError(t, err, "Expected no error")
//		t.Log(err)
//	})
//}
//
func TestWavefrontOperatorReconciler_Reconcile(t *testing.T) {
	wf := &wavefrontcomv1.WavefrontOperator{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{},
		Spec:       wavefrontcomv1.WavefrontOperatorSpec{WavefrontUrl: "testUrl", WavefrontToken: "testToken"},
		Status:     wavefrontcomv1.WavefrontOperatorStatus{},
	}
	s := scheme.Scheme
	s.AddKnownTypes(wavefrontcomv1.GroupVersion, wf)
	//proxy := &v1.Deployment{
	//	TypeMeta:   metav1.TypeMeta{},
	//	ObjectMeta: metav1.ObjectMeta{},
	//	Spec:       v1.DeploymentSpec{},
	//	Status:     v1.DeploymentStatus{},
	//}
	testRestMapper := meta.NewDefaultRESTMapper(
		[]schema.GroupVersion{
			{Group: "apps", Version: "v1"},
		})
	testRestMapper.Add(schema.GroupVersionKind{
		Group:   "apps",
		Version: "v1",
		Kind:    "Deployment",
	}, meta.RESTScopeNamespace)

	clientBuilder := fake.NewClientBuilder()
	clientBuilder = clientBuilder.WithScheme(s).WithObjects(wf).WithRESTMapper(testRestMapper)
	client := clientBuilder.Build()

	dynamicClient := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]interface{}{
				"name":      "testProxy",
				"namespace": "testNamespace",
			},
			"spec": map[string]interface{}{
				"testSpec": "3",
			},
		},
	})

	t.Run("basic", func(t *testing.T) {
		r := &controllers.WavefrontOperatorReconciler{
			Client:        client,
			Scheme:        nil,
			FS:            os.DirFS("../deploy"),
			DynamicClient: dynamicClient,
			RestMapper:    client.RESTMapper(),
		}
		results, err := r.Reconcile(context.Background(), reconcile.Request{})
		assert.Nil(t, err)
		assert.Equal(t, ctrl.Result{}, results)
	})
}
