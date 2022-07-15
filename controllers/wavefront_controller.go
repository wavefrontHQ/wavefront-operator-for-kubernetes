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
	"context"
	"crypto/sha1"
	"fmt"
	"net/url"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/health"
	baseYaml "gopkg.in/yaml.v2"

	wf "github.com/wavefrontHQ/wavefront-operator-for-kubernetes/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
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
	Scheme          *runtime.Scheme
	ResourceManager *ResourceManager
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

	wavefront := &wf.Wavefront{}
	err := r.Client.Get(ctx, req.NamespacedName, wavefront)
	if err != nil && !errors.IsNotFound(err) {
		log.Log.Error(err, "error getting wavefront operator crd")
		return ctrl.Result{}, err
	}

	if errors.IsNotFound(err) {
		err = r.ResourceManager.readAndDeleteResources()
		return ctrl.Result{}, nil
	}

	err = r.preprocess(wavefront, ctx)
	if err != nil {
		log.Log.Error(err, "error preprocessing Wavefront Spec")
		return ctrl.Result{}, err
	}

	err = r.ResourceManager.readAndCreateResources(wavefront.Spec)
	if err != nil {
		log.Log.Error(err, "error creating resources")
		return ctrl.Result{}, err
	}

	err = r.reportHealthStatus(ctx, wavefront)
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
		Client: client,
		Scheme: scheme,
		// why does a Reconciler own and create an FS and why is it not at least passed into New-?
		ResourceManager: NewResourceManager(os.DirFS(DeployDir), mapper, clientSet.AppsV1(), dynamicClient), // TODO: pass in ResourceManager
	}

	return reconciler, nil
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
		err := r.parseHttpProxyConfigs(wavefront, ctx)
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

	wavefront.Status.Healthy, wavefront.Status.Message = health.UpdateComponentStatuses(r.ResourceManager.Appsv1, deploymentStatuses, daemonSetStatuses)

	return r.Status().Update(ctx, wavefront)
}
