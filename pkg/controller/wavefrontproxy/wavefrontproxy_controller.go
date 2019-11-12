package wavefrontproxy

import (
	"context"
	"github.com/go-logr/logr"
	wfv1 "github.com/wavefronthq/wavefront-operator/pkg/apis/wavefront/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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
	"time"

)

var log = logf.Log.WithName("controller_wavefrontproxy")

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
	err = c.Watch(&source.Kind{Type: &wfv1.WavefrontProxy{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource Deployments and requeue the owner WavefrontProxy
	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &wfv1.WavefrontProxy{},
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
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileWavefrontProxy) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name",
		request.Name)
	reqLogger.Info("Reconciling WavefrontProxy :::")

	// Fetch the WavefrontProxy instance
	instance := &wfv1.WavefrontProxy{}
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

	// Make desired InternalWavefrontProxyInstance.
	var desiredIp = &InternalWavefrontProxy{}
	desiredIp.initialize(instance.DeepCopy(), reqLogger)

	result, err := r.reconcileProxy(desiredIp, reqLogger)
	if err != nil {
		return result, err
	} else if desiredIp.updateCR {
		err := r.updateCRStatus(instance, desiredIp, reqLogger)
		reqLogger.Info("Updated WavefrontProxy CR Status.")
		if err != nil {
			reqLogger.Error(err, "Failed to update WavefrontProxy CR status")
			return reconcile.Result{}, err
		}
		return result, nil
	}

	return reconcile.Result{RequeueAfter: 1 * time.Hour}, nil
}

// reconcileProxy verifies whether the given deployment already exists, If not creates a new one.
// If exists, then brings it from current state -> desired state.
func (r *ReconcileWavefrontProxy) reconcileProxy(ip *InternalWavefrontProxy, reqLogger logr.Logger) (reconcile.Result, error) {
	desiredDep := newDeployment(ip)

	// Check if the deployment already exists, if not create a new one.
	existingDep := &appsv1.Deployment{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: ip.instance.Name, Namespace: ip.instance.Namespace}, existingDep)
	if err != nil && errors.IsNotFound(err) {
		// Set WavefrontProxy ip as the owner and controller
		if err := controllerutil.SetControllerReference(ip.instance, desiredDep, r.scheme); err != nil {
			return reconcile.Result{}, err
		}

		reqLogger.Info("Creating a new Deployment", "Deployment.Namespace", desiredDep.Namespace,
			"Deployment.Name", desiredDep.Name)
		err = r.client.Create(context.TODO(), desiredDep)
		if err != nil {
			reqLogger.Error(err, "Failed to create new Deployment", "Deployment.Namespace",
				desiredDep.Namespace, "Deployment.Name", desiredDep.Name)
			return reconcile.Result{}, err
		}
		// Update CR Status on Create.
		ip.updateCR = true
		// A new CR Deployment was successfull, now create a new CR Service.
		return r.reconcileProxySvc(ip, false, reqLogger)
	} else if err != nil {
		reqLogger.Error(err, "Failed to get Deployment")
		return reconcile.Result{}, err
	}

	reqLogger.Info("Check WavefrontProxy Updated :: ")
	// Deployment already exists, update if the desired spec has changed
	if specChanged(existingDep, desiredDep) {
		reqLogger.Info("Updating the existing deployment.")
		err := r.client.Update(context.TODO(), desiredDep)
		if err != nil {
			return reconcile.Result{}, err
		}
		// Update CR Status on Update.
		ip.updateCR = true
		// Update CR service if CR spec changed.
		if result, err := r.reconcileProxySvc(ip, true, reqLogger); err != nil {
			return result, err
		}
	}

	return reconcile.Result{}, nil
}

// updateCRStatus updates the status of the WavefrontProxy CR.
func (r *ReconcileWavefrontProxy) updateCRStatus(instance *wfv1.WavefrontProxy, desiredIp *InternalWavefrontProxy, reqLogger logr.Logger) error {
	reqLogger.Info("Updating WavefrontProxy CR Status :")
	instance.Status = desiredIp.instance.Status
	instance.Status.UpdatedTimestamp = metav1.Now()

	return r.client.Status().Update(context.TODO(), instance)
}


// reconcileProxySvc verifies whether the given service already exists, If not creates a new one.
// If exists, then brings it from current state -> desired state.
func (r *ReconcileWavefrontProxy) reconcileProxySvc(ip *InternalWavefrontProxy, isSpecChanged bool, reqLogger logr.Logger) (reconcile.Result, error) {
	// Check if the service already exists, if not create a new one.
	existingSvc := &corev1.Service{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: ip.instance.Name, Namespace: ip.instance.Namespace}, existingSvc)
	if err != nil && errors.IsNotFound(err) {
		svc := newService(ip)
		// Set WavefrontProxy instance as the owner and controller
		if err := controllerutil.SetControllerReference(ip.instance, svc, r.scheme); err != nil {
			return reconcile.Result{}, err
		}
		reqLogger.Info("Creating a new Service", "Service.Namespace", svc.Namespace, "Service.Name", svc.Name)
		err = r.client.Create(context.TODO(), svc)
		if err != nil {
			reqLogger.Error(err, "Failed to create new Service", "Service.Namespace", svc.Namespace, "Service.Name", svc.Name)
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	} else if err != nil {
		reqLogger.Error(err, "Failed to get Service")
		return reconcile.Result{}, err
	}

	// wavefront-proxy service exists and
	if isSpecChanged {
		desiredSvc := verifyAndModifySvc(*existingSvc, ip)
		if desiredSvc != nil {
			reqLogger.Info("Updating the wavefront-proxy service")
			err = r.client.Update(context.TODO(), desiredSvc)
			if err != nil {
				reqLogger.Error(err, "Failed to update the wavefront-proxy service")
				return reconcile.Result{}, err
			}
		}
	}

	return reconcile.Result{}, nil
}
