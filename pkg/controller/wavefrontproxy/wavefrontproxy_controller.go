package wavefrontproxy

import (
	"context"
	wavefrontv1alpha1 "github.com/wavefronthq/wavefront-operator/pkg/apis/wavefront/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_wavefrontproxy")

const (
	WavefrontProxyKind = "WavefrontProxy"
)

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new WavefrontProxy Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileWavefrontProxy{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("wavefrontproxy-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource WavefrontProxy
	err = c.Watch(&source.Kind{Type: &wavefrontv1alpha1.WavefrontProxy{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner WavefrontProxy
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &wavefrontv1alpha1.WavefrontProxy{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileWavefrontProxy implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileWavefrontProxy{}

// ReconcileWavefrontProxy reconciles a WavefrontProxy object
type ReconcileWavefrontProxy struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a WavefrontProxy object and makes changes based on the state read
// and what is in the WavefrontProxy.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileWavefrontProxy) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling WavefrontProxy")

	// Fetch the WavefrontProxy instance
	instance := &wavefrontv1alpha1.WavefrontProxy{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}
	// TODO: Handle Defaults? - r.scheme.Default(instance)
	// TODO: Handle Updates here.

	// Define a new Pod object
	if !instance.Spec.ProxyEnabled {
		reqLogger.Info("Skip reconcile since :", "ProxyEnabled", instance.Spec.ProxyEnabled)
		return reconcile.Result{}, nil
	}

	// Check if the deployment already exists, if not create a new one.
	found := &appsv1.Deployment{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		dep := r.newDeploymentForCR(instance)

		// Set WavefrontProxy instance as the owner and controller
		if err := controllerutil.SetControllerReference(instance, dep, r.scheme); err != nil {
			return reconcile.Result{}, err
		}

		reqLogger.Info("Creating a new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
		err = r.client.Create(context.TODO(), dep)
		if err != nil {
			reqLogger.Error(err, "Failed to create new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
			return reconcile.Result{}, err
		}
		//instance.Status.CreatedTimestamp = metav1.Now();
		//err := r.client.Status().Update(context.TODO(), instance)
		//if err != nil {
		//	reqLogger.Error(err, "Failed to update WavefrontProxy CreatedTimestamp status")
		//	return reconcile.Result{}, err
		//}
		//reqLogger.Info("CreatedTimestamp :: ", "instance.Status.CreatedTimestamp", instance.Status.CreatedTimestamp)
		// Deployment created successfully - return and requeue
		return reconcile.Result{Requeue: true}, nil
	} else if err != nil {
		reqLogger.Error(err, "Failed to get Deployment")
		return reconcile.Result{}, err
	}

	// Ensure the deployment size is the same as the spec
	size := instance.Spec.Size
	if *found.Spec.Replicas != size {
		found.Spec.Replicas = &size
		err = r.client.Update(context.TODO(), found)
		if err != nil {
			reqLogger.Error(err, "Failed to update Deployment", "Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
			return reconcile.Result{}, err
		}
		instance.Status.UpdatedTimestamp = metav1.Now()
		//err := r.client.Status().Update(context.TODO(), instance)
		//if err != nil {
		//	reqLogger.Error(err, "Failed to update WavefrontProxy UpdatedTimestamp status")
		//	return reconcile.Result{}, err
		//}
		//reqLogger.Info("UpdatedTimestamp :: ", "instance.Status.UpdatedTimestamp", instance.Status.UpdatedTimestamp)
		// Spec updated - return and requeue
		return reconcile.Result{Requeue: true}, nil
	}

	// TODO: Update Proxy status with appropriate details.

	// Deployment already exists - don't requeue
	reqLogger.Info("Skip reconcile: Deployment already exists", "Deployment.Namespace", found.Namespace,
		"Deployment.Name", found.Name)
	return reconcile.Result{}, nil
}

// deploymentForMemcached returns a memcached Deployment object
func (r *ReconcileWavefrontProxy) newDeploymentForCR(instance *wavefrontv1alpha1.WavefrontProxy) *appsv1.Deployment {
	labels := getLabelsForCR(instance)
	replicas := instance.Spec.Size

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: *(newPodSpecForCR(instance)),
			},
		},
	}

	return dep
}

// getLabelsForCR returns the labels for selecting the resources
// belonging to the given WavefrontProxy CR name.
func getLabelsForCR(instance *wavefrontv1alpha1.WavefrontProxy) map[string]string {
	// For consistency, labels assigned are as per suggested yaml at
	// https://github.com/wavefrontHQ/wavefront-kubernetes/blob/master/wavefront-proxy/wavefront.yaml
	// If any changes are made, make sure they are reflected in both places.
	return map[string]string{
		"app":  "wavefront-proxy",
		"name": instance.Name,
	}
}

func newPodSpecForCR(instance *wavefrontv1alpha1.WavefrontProxy) *corev1.PodSpec {
	envVar := constructEnvVars(instance)
	return &corev1.PodSpec{
		Containers: []corev1.Container{{
			Name:            "wavefront-proxy",
			Image:           instance.Spec.Image,
			ImagePullPolicy: corev1.PullAlways,
			Env:             envVar,
			Ports: []corev1.ContainerPort{{
				// TODO : Extract comma separated ports.
				Name:          "metric",
				ContainerPort: 2878,
			}},
		}},
	}
}

func constructEnvVars(instance *wavefrontv1alpha1.WavefrontProxy) []corev1.EnvVar {
	return []corev1.EnvVar{{
		Name:  "WAVEFRONT_URL",
		Value: instance.Spec.Url,
	}, {
		Name:  "WAVEFRONT_TOKEN",
		Value: instance.Spec.Token,
	}}
}
