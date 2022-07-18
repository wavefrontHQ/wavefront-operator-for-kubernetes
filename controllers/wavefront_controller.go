/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"bytes"
	"context"
	"crypto/sha1"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/health"
	baseYaml "gopkg.in/yaml.v2"

	"k8s.io/client-go/kubernetes"
	typedappsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"

	wf "github.com/wavefrontHQ/wavefront-operator-for-kubernetes/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const DeployDir = "../deploy/internal"
const ProxyName = "wavefront-proxy"
const ClusterCollectorName = "wavefront-cluster-collector"
const NodeCollectorName = "wavefront-node-collector"

// WavefrontReconciler reconciles a Wavefront object
type WavefrontReconciler struct {
	client.Client

	wavefront         *wf.Wavefront
	Scheme            *runtime.Scheme
	FS                fs.FS
	Appsv1            typedappsv1.AppsV1Interface
	KubernetesManager KubernetesManager
}

type KubernetesManager struct {
	RestMapper    meta.RESTMapper
	DynamicClient dynamic.Interface
}

// +kubebuilder:rbac:groups=wavefront.com,namespace=wavefront,resources=wavefronts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=wavefront.com,namespace=wavefront,resources=wavefronts/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=wavefront.com,namespace=wavefront,resources=wavefronts/finalizers,verbs=update

// Permissions for creating Kubernetes resources from internal files.
// Possible point of confusion: the collector itself watches resources,
// but the operator doesn't need to... yet?
// +kubebuilder:rbac:groups=apps,namespace=wavefront,resources=deployments,verbs=get;create;update;patch;delete;watch;list
// +kubebuilder:rbac:groups="",namespace=wavefront,resources=services,verbs=get;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,namespace=wavefront,resources=daemonsets,verbs=get;create;update;patch;delete
// +kubebuilder:rbac:groups="",namespace=wavefront,resources=serviceaccounts,verbs=get;create;update;patch;delete
// +kubebuilder:rbac:groups="",namespace=wavefront,resources=configmaps,verbs=get;create;update;patch;delete
// +kubebuilder:rbac:groups="",namespace=wavefront,resources=secrets,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile

func (r *WavefrontReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// TODO: write separate story shut down collector and proxy if Operator is being shut down?

	r.wavefront = &wf.Wavefront{}
	err := r.Client.Get(ctx, req.NamespacedName, r.wavefront)
	if err != nil && !errors.IsNotFound(err) {
		log.Log.Error(err, "error getting wavefront operator crd")
		return ctrl.Result{}, err
	}

	if errors.IsNotFound(err) {
		err = withFSResources(r.FS, r.wavefront.Spec, r.deleteKubernetesObjects)
		//err = r.readAndDeleteResources(r.FS)
		return ctrl.Result{}, nil
	}

	err = r.preprocess(r.wavefront, ctx)
	if err != nil {
		log.Log.Error(err, "error preprocessing Wavefront Spec")
		return ctrl.Result{}, err
	}

	err = withFSResources(r.FS, r.wavefront.Spec, r.readAndCreateResources)
	if err != nil {
		log.Log.Error(err, "error creating resources")
		return ctrl.Result{}, err
	}

	err = r.reportHealthStatus(ctx, r.wavefront)
	if err != nil {
		log.Log.Error(err, "error report health status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{
		Requeue:      true,
		RequeueAfter: 30 * time.Second,
	}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *WavefrontReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&wf.Wavefront{}).
		Complete(r)
}

func NewWavefrontReconciler(client client.Client, scheme *runtime.Scheme) (operator *WavefrontReconciler, err error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	dc, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, err
	}
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(dc))

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	clientSet, err := kubernetes.NewForConfig(config)

	reconciler := &WavefrontReconciler{
		Client:        client,
		Scheme:        scheme,
		FS:            os.DirFS(DeployDir),
		Appsv1:        clientSet.AppsV1(),
		KubernetesManager: KubernetesManager{
			RestMapper:    mapper,
			DynamicClient: dynamicClient,
		},
	}

	return reconciler, nil
}

func withFSResources(FS fs.FS, spec wf.WavefrontSpec, resourceHandler func(resources []string) error) error {
	resources, err := readAndInterpolateResources(FS, spec)
	if err != nil {
		return err
	}

	return resourceHandler(resources)
}

// Read, Create, Update and Delete Resources.
func (r *WavefrontReconciler) readAndCreateResources(resources []string) error {
	controllerManagerUID, err := r.getControllerManagerUID()
	if err != nil {
		return err
	}
	r.wavefront.Spec.ControllerManagerUID = string(controllerManagerUID)

	err = r.createKubernetesObjects(resources, r.wavefront.Spec)
	if err != nil {
		return err
	}
	return nil
}

func readAndInterpolateResources(FS fs.FS, spec wf.WavefrontSpec) ([]string, error) {
	var resources []string

	resourceFiles, err := resourceFiles("yaml")
	if err != nil {
		return nil, err
	}

	for _, resourceFile := range resourceFiles {
		resourceTemplate, err := newTemplate(resourceFile).ParseFS(FS, resourceFile)
		if err != nil {
			return nil, err
		}
		buffer := bytes.NewBuffer(nil)
		err = resourceTemplate.Execute(buffer, spec)
		if err != nil {
			return nil, err
		}
		resources = append(resources, buffer.String())
	}
	return resources, nil
}

func (r *WavefrontReconciler) createKubernetesObjects(resources []string, wavefrontSpec wf.WavefrontSpec) error {
	var resourceDecoder = yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	for _, resource := range resources {
		object := &unstructured.Unstructured{}
		_, gvk, err := resourceDecoder.Decode([]byte(resource), nil, object)
		if err != nil {
			return err
		}

		mapping, err := r.KubernetesManager.RestMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return err
		}

		objLabels := object.GetLabels()
		if labelVal, _ := objLabels["app.kubernetes.io/component"]; labelVal == "collector" && !wavefrontSpec.DataCollection.Metrics.Enable {
			continue
		}
		if labelVal, _ := objLabels["app.kubernetes.io/component"]; labelVal == "proxy" && !wavefrontSpec.DataExport.WavefrontProxy.Enable {
			continue
		}
		if object.GetKind() == "ConfigMap" && wavefrontSpec.DataCollection.Metrics.CollectorConfigName != object.GetName() {
			continue
		}

		err = r.createResources(mapping, object)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *WavefrontReconciler) createResources(mapping *meta.RESTMapping, obj *unstructured.Unstructured) error {
	var dynamicClient dynamic.ResourceInterface
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		dynamicClient = r.KubernetesManager.DynamicClient.Resource(mapping.Resource).Namespace(obj.GetNamespace())
	} else {
		dynamicClient = r.KubernetesManager.DynamicClient.Resource(mapping.Resource)
	}

	_, err := dynamicClient.Get(context.TODO(), obj.GetName(), v1.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		_, err = dynamicClient.Create(context.TODO(), obj, v1.CreateOptions{})
		if err != nil {
			return err
		}
	} else if err == nil {
		data, err := json.Marshal(obj)
		if err != nil {
			return err
		}
		_, err = dynamicClient.Patch(context.TODO(), obj.GetName(), types.MergePatchType, data, v1.PatchOptions{})
		if err != nil {
			return err
		}
	}
	return err
}

func (r *WavefrontReconciler) getControllerManagerUID() (types.UID, error) {
	deployment, err := r.Appsv1.Deployments("wavefront").Get(context.Background(), "wavefront-controller-manager", v1.GetOptions{})
	if err != nil {
		return "", err
	}
	return deployment.UID, nil
}

func resourceFiles(suffix string) ([]string, error) {
	var files []string

	err := filepath.Walk(DeployDir,
		func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if strings.HasSuffix(path, suffix) {
				files = append(files, info.Name())
			}
			return nil
		},
	)

	return files, err
}

func (r *WavefrontReconciler) deleteKubernetesObjects(resources []string) error {
	var resourceDecoder = yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	for _, resource := range resources {
		object := &unstructured.Unstructured{}
		_, gvk, err := resourceDecoder.Decode([]byte(resource), nil, object)
		if err != nil {
			return err
		}

		mapping, err := r.KubernetesManager.RestMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return err
		}

		err = r.deleteResources(mapping, object)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *WavefrontReconciler) deleteResources(mapping *meta.RESTMapping, obj *unstructured.Unstructured) error {
	var dynamicClient dynamic.ResourceInterface
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		dynamicClient = r.KubernetesManager.DynamicClient.Resource(mapping.Resource).Namespace(obj.GetNamespace())
	} else {
		dynamicClient = r.KubernetesManager.DynamicClient.Resource(mapping.Resource)
	}
	_, err := dynamicClient.Get(context.TODO(), obj.GetName(), v1.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return dynamicClient.Delete(context.TODO(), obj.GetName(), v1.DeleteOptions{})
}

func newTemplate(resourceFile string) *template.Template {
	fMap := template.FuncMap{
		"toYaml": func(v interface{}) string {
			data, err := baseYaml.Marshal(v)
			if err != nil {
				log.Log.Error(err, "error in toYaml")
				return ""
			}
			return strings.TrimSuffix(string(data), "\n")
		},
		"indent": func(spaces int, v string) string {
			pad := strings.Repeat(" ", spaces)
			return pad + strings.Replace(v, "\n", "\n"+pad, -1)
		},
	}
	return template.New(resourceFile).Funcs(fMap)
}

func hashValue(bytes []byte) string {
	h := sha1.New()
	h.Write(bytes)
	return fmt.Sprintf("%x", h.Sum(nil))
}

// Preprocessing Wavefront Spec
func (r *WavefrontReconciler) preprocess(wavefront *wf.Wavefront, ctx context.Context) error {
	if wavefront.Spec.DataCollection.Metrics.Enable {
		if len(wavefront.Spec.DataCollection.Metrics.CustomConfig) == 0 {
			wavefront.Spec.DataCollection.Metrics.CollectorConfigName = "default-wavefront-collector-config"
		} else {
			wavefront.Spec.DataCollection.Metrics.CollectorConfigName = wavefront.Spec.DataCollection.Metrics.CustomConfig
		}
	}

	if wavefront.Spec.DataExport.WavefrontProxy.Enable {
		wavefront.Spec.DataExport.WavefrontProxy.ConfigHash = ""
		wavefront.Spec.DataCollection.Metrics.ProxyAddress = fmt.Sprintf("wavefront-proxy:%d", wavefront.Spec.DataExport.WavefrontProxy.MetricPort)
		err := r.parseHttpProxyConfigs(wavefront, ctx) // TODO: triple nested usage of func's with reconciler receiver...
		if err != nil {
			errInfo := fmt.Sprintf("Error setting up http proxy configuration: %s", err.Error())
			log.Log.Info(errInfo)
			return err
		}
	} else if len(wavefront.Spec.DataExport.ExternalWavefrontProxy.Url) != 0 {
		wavefront.Spec.DataCollection.Metrics.ProxyAddress = wavefront.Spec.DataExport.ExternalWavefrontProxy.Url
	}

	wavefront.Spec.DataExport.WavefrontProxy.Args = strings.ReplaceAll(wavefront.Spec.DataExport.WavefrontProxy.Args, "\r", "")
	wavefront.Spec.DataExport.WavefrontProxy.Args = strings.ReplaceAll(wavefront.Spec.DataExport.WavefrontProxy.Args, "\n", "")
	return nil
}

func (r *WavefrontReconciler) parseHttpProxyConfigs(wavefront *wf.Wavefront, ctx context.Context) error {
	if len(wavefront.Spec.DataExport.WavefrontProxy.HttpProxy.Secret) != 0 {
		httpProxySecret, err := r.findHttpProxySecret(wavefront, ctx)
		if err != nil {
			return err
		}
		err = setHttpProxyConfigs(httpProxySecret, wavefront)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *WavefrontReconciler) findHttpProxySecret(wavefront *wf.Wavefront, ctx context.Context) (*corev1.Secret, error) {
	secret := client.ObjectKey{
		Namespace: "wavefront",
		Name:      wavefront.Spec.DataExport.WavefrontProxy.HttpProxy.Secret,
	}
	httpProxySecret := &corev1.Secret{}
	err := r.Client.Get(ctx, secret, httpProxySecret)
	if err != nil {
		return nil, err
	}
	return httpProxySecret, nil
}

func setHttpProxyConfigs(httpProxySecret *corev1.Secret, wavefront *wf.Wavefront) error {
	httpProxySecretData := map[string]string{}
	for k, v := range httpProxySecret.Data {
		httpProxySecretData[k] = string(v)
	}

	httpUrl, err := url.Parse(httpProxySecretData["http-url"])
	if err != nil {
		return err
	}
	wavefront.Spec.DataExport.WavefrontProxy.HttpProxy.HttpProxyHost = httpUrl.Hostname()
	wavefront.Spec.DataExport.WavefrontProxy.HttpProxy.HttpProxyPort = httpUrl.Port()
	wavefront.Spec.DataExport.WavefrontProxy.HttpProxy.HttpProxyUser = httpProxySecretData["basic-auth-username"]
	wavefront.Spec.DataExport.WavefrontProxy.HttpProxy.HttpProxyPassword = httpProxySecretData["basic-auth-password"]

	configHashBytes, err := json.Marshal(wavefront.Spec.DataExport.WavefrontProxy.HttpProxy)
	if err != nil {
		return err
	}

	if len(httpProxySecretData["tls-root-ca-bundle"]) != 0 {
		wavefront.Spec.DataExport.WavefrontProxy.HttpProxy.UseHttpProxyCAcert = true
		configHashBytes = append(configHashBytes, httpProxySecret.Data["tls-root-ca-bundle"]...)
	}

	wavefront.Spec.DataExport.WavefrontProxy.ConfigHash = hashValue(configHashBytes)

	return nil
}

// Reporting Health Status
func (r *WavefrontReconciler) reportHealthStatus(ctx context.Context, wavefront *wf.Wavefront) error {
	deploymentStatuses := map[string]*wf.DeploymentStatus{}
	daemonSetStatuses := map[string]*wf.DaemonSetStatus{}

	if wavefront.Spec.DataExport.WavefrontProxy.Enable {
		deploymentStatuses[ProxyName] = &wavefront.Status.Proxy
	}

	if wavefront.Spec.DataCollection.Metrics.Enable {
		deploymentStatuses[ClusterCollectorName] = &wavefront.Status.ClusterCollector
		daemonSetStatuses[NodeCollectorName] = &wavefront.Status.NodeCollector
	}

	wavefront.Status.Warnings, wavefront.Status.Healthy, wavefront.Status.Message = health.UpdateComponentStatuses(r.Appsv1, deploymentStatuses, daemonSetStatuses, wavefront)

	return r.Status().Update(ctx, wavefront)
}
