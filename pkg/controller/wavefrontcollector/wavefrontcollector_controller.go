// Copyright 2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package wavefrontcollector

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"

	wavefrontv1alpha1 "github.com/wavefronthq/wavefront-operator-for-kubernetes/pkg/apis/wavefront/v1alpha1"
	"github.com/wavefronthq/wavefront-operator-for-kubernetes/pkg/controller/util"

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

var log = logf.Log.WithName("controller_wavefrontcollector")

// Add creates a new WavefrontCollector Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileWavefrontCollector{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("wavefrontcollector-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource WavefrontCollector
	err = c.Watch(&source.Kind{Type: &wavefrontv1alpha1.WavefrontCollector{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource DaemonSets and requeue the owner WavefrontCollector
	err = c.Watch(&source.Kind{Type: &appsv1.DaemonSet{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &wavefrontv1alpha1.WavefrontCollector{},
	})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource Deployments and requeue the owner WavefrontCollector
	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &wavefrontv1alpha1.WavefrontCollector{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileWavefrontCollector implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileWavefrontCollector{}

// ReconcileWavefrontCollector reconciles a WavefrontCollector object
type ReconcileWavefrontCollector struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads the state of the cluster for a WavefrontCollector object and makes changes based on the state read
// and what is in the WavefrontCollector.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileWavefrontCollector) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling WavefrontCollector")

	// Fetch the WavefrontCollector instance
	instance := &wavefrontv1alpha1.WavefrontCollector{}
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

	desiredCRInstance := instance.DeepCopy()
	getLatestCollector(reqLogger, desiredCRInstance)

	var updateCR bool
	if instance.Spec.Daemon {
		updateCR, err = r.reconcileDaemonSet(reqLogger, desiredCRInstance)
	} else {
		updateCR, err = r.reconcileDeployment(reqLogger, desiredCRInstance)
	}
	if err != nil {
		return reconcile.Result{}, err
	} else if updateCR {
		err := r.updateCRStatus(reqLogger, instance, desiredCRInstance)
		if err != nil {
			reqLogger.Error(err, "Failed to update WavefrontCollector CR status")
			return reconcile.Result{}, err
		}
		reqLogger.Info("Updated WavefrontCollector CR Status.")
	}
	return reconcile.Result{RequeueAfter: 1 * time.Hour}, nil
}

func getLatestCollector(reqLogger logr.Logger, instance *wavefrontv1alpha1.WavefrontCollector) {
	imgSlice := strings.Split(instance.Spec.Image, ":")
	// Validate Image format and Auto Upgrade.
	if len(imgSlice) == 2 {
		instance.Status.Version = imgSlice[1]
		if instance.Spec.EnableAutoUpgrade {
			finalVer, err := util.GetLatestVersion(instance.Spec.Image, reqLogger)
			if err == nil && finalVer != "" {
				instance.Status.Version = finalVer
				instance.Spec.Image = imgSlice[0] + ":" + finalVer
			} else if err != nil {
				reqLogger.Error(err, "Auto Upgrade Error.")
			}
		}
	} else {
		reqLogger.Info("Cannot update CR's Status.version", "Un-recognized format for CR Image", instance.Spec.Image)
	}
}

func (r *ReconcileWavefrontCollector) reconcileDaemonSet(reqLogger logr.Logger, instance *wavefrontv1alpha1.WavefrontCollector) (bool, error) {
	updateCR := false
	ds := newDaemonSetForCR(instance)

	// Set WavefrontCollector instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, ds, r.scheme); err != nil {
		return updateCR, err
	}

	// delete a deployment if one exists matching the same name
	r.deleteCollector(reqLogger, instance, &appsv1.Deployment{})

	// Check if this DaemonSet already exists
	found := &appsv1.DaemonSet{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: ds.Name, Namespace: ds.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new DaemonSet")
		err = r.client.Create(context.TODO(), ds)
		if err != nil {
			return updateCR, err
		}
		// Update CR Status on Create.
		updateCR = true
		// DaemonSet created successfully - don't requeue
		return updateCR, nil
	} else if err != nil {
		reqLogger.Info("Return error")
		return updateCR, err
	}

	if specChanged(&found.Spec.Template, &instance.Spec) {
		reqLogger.Info("Updating the existing daemonset")
		err := r.client.Update(context.TODO(), ds)
		// Update CR Status on Update.
		updateCR = true
		if err != nil {
			return updateCR, err
		}
	}

	// Already exists - don't requeue
	return updateCR, nil
}

func newDaemonSetForCR(instance *wavefrontv1alpha1.WavefrontCollector) *appsv1.DaemonSet {
	selector := buildLabels(instance.Name)
	podSpec := newPodSpecForCR(instance)

	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
			Labels:    selector,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{MatchLabels: selector},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: selector},
				Spec:       podSpec,
			},
		},
	}
}

func (r *ReconcileWavefrontCollector) deleteCollector(reqLogger logr.Logger, instance *wavefrontv1alpha1.WavefrontCollector, found runtime.Object) {
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, found)
	if err != nil {
		if !errors.IsNotFound(err) {
			reqLogger.Error(err, "error looking up existing instance", instance.Name)
			return
		}
		return
	}

	// instance exists - delete it
	reqLogger.Info("terminating existing wavefront collector instance")
	err = r.client.Delete(context.TODO(), found)
	if err != nil {
		reqLogger.Error(err, "error deleting wavefront collector instance")
		return
	}
}

func newPodSpecForCR(instance *wavefrontv1alpha1.WavefrontCollector) corev1.PodSpec {
	daemon := "--daemon=false"
	if instance.Spec.Daemon {
		daemon = "--daemon=true"
	}
	configName := instance.Spec.ConfigName
	if configName == "" {
		configName = fmt.Sprintf("%s-config", instance.Name)
	}

	debug := "--log-level=info"
	if instance.Spec.EnableDebug {
		debug = "--log-level=debug"
	}

	volumeMounts := []corev1.VolumeMount{{
		Name:      "procfs",
		MountPath: "/host/proc",
		ReadOnly:  true,
	}}

	volumes := []corev1.Volume{{
		Name: "procfs",
		VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Path: "/proc",
			},
		},
	}}

	if !instance.Spec.UseOpenshiftDefaultConfig {
		configVM := corev1.VolumeMount{
			Name:      "collector-config",
			MountPath: "/etc/collector",
			ReadOnly:  true,
		}

		configV := corev1.Volume{
			Name: "collector-config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: configName},
				},
			},
		}

		volumeMounts = append(volumeMounts, configVM)
		volumes = append(volumes, configV)
	}

	return corev1.PodSpec{
		ServiceAccountName: instance.Name,
		Tolerations:        instance.Spec.Tolerations,

		Containers: []corev1.Container{{
			Name:            "wavefront-collector",
			Image:           instance.Spec.Image,
			ImagePullPolicy: corev1.PullAlways,
			Command:         []string{"/wavefront-collector", daemon, debug, "--config-file=/etc/collector/collector.yaml"},
			Env:             instance.Spec.Env,
			Resources:       instance.Spec.Resources,
			VolumeMounts:    volumeMounts,
		}},
		Volumes: volumes,
	}
}

func (r *ReconcileWavefrontCollector) reconcileDeployment(reqLogger logr.Logger, instance *wavefrontv1alpha1.WavefrontCollector) (bool, error) {
	updateCR := false
	deployment := newDeploymentForCR(instance)

	// Set WavefrontCollector instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, deployment, r.scheme); err != nil {
		return updateCR, err
	}

	// delete daemon set if one exists matching the same name
	r.deleteCollector(reqLogger, instance, &appsv1.DaemonSet{})

	// Check if this Deployment already exists
	found := &appsv1.Deployment{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new Deployment")
		err = r.client.Create(context.TODO(), deployment)
		if err != nil {
			return updateCR, err
		}
		// Update CR Status on Create.
		updateCR = true
		// Deployment created successfully - don't requeue
		return updateCR, nil
	} else if err != nil {
		reqLogger.Error(err, "error looking up deployment")
		return updateCR, err
	}

	// already exists. update if spec has changed
	if specChanged(&found.Spec.Template, &instance.Spec) {
		reqLogger.Info("Updating the existing deployment")
		err := r.client.Update(context.TODO(), deployment)
		if err != nil {
			return updateCR, err
		}
		// Update CR Status on Update.
		updateCR = true
	}

	return updateCR, nil
}

func newDeploymentForCR(instance *wavefrontv1alpha1.WavefrontCollector) *appsv1.Deployment {
	selector := buildLabels(instance.Name)
	podSpec := newPodSpecForCR(instance)

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
			Labels:    selector,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: selector},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: selector},
				Spec:       podSpec,
			},
		},
	}
}

// BuildLabels returns generic labels based on the name given for a Collector agent
func buildLabels(name string) map[string]string {
	return map[string]string{
		"k8s-app": "wavefront-collector",
		"name":    name,
	}
}

// updateCRStatus updates the status of the WavefrontProxy CR.
func (r *ReconcileWavefrontCollector) updateCRStatus(reqLogger logr.Logger, instance *wavefrontv1alpha1.WavefrontCollector, desiredCRInstance *wavefrontv1alpha1.WavefrontCollector) error {
	reqLogger.Info("Updating WavefrontCollector CR Status :")
	instance.Status = desiredCRInstance.Status
	instance.Status.UpdatedTimestamp = metav1.Now()

	return r.client.Status().Update(context.TODO(), instance)
}
