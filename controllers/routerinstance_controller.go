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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"context"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	kasicov1 "github.com/world-direct/kasico/api/v1"
)

// RouterInstanceReconciler reconciles a RouterInstance object
type RouterInstanceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=kasico.world-direct.at,resources=routerinstances,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kasico.world-direct.at,resources=routerinstances/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kasico.world-direct.at,resources=routerinstances/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=daemonSet,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=service,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *RouterInstanceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	// based on https://github.com/operator-framework/operator-sdk/blob/latest/testdata/go/v3/memcached-operator/controllers/memcached_controller.go

	log := ctrllog.FromContext(ctx)

	// Fetch the RouterInstance instance
	routerInstance := &kasicov1.RouterInstance{}
	err := r.Get(ctx, req.NamespacedName, routerInstance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("RouterInstance resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get RouterInstance")
		return ctrl.Result{}, err
	}

	// Check if the DaemonSet already exists, if not create a new one
	daemonSet := &appsv1.DaemonSet{}
	err = r.Get(ctx, types.NamespacedName{Name: routerInstance.Name, Namespace: routerInstance.Namespace}, daemonSet)
	if err != nil && errors.IsNotFound(err) {
		// Define a new daemonSet
		daemonSet := r.daemonSetForRouterInstance(routerInstance)
		log.Info("Creating a new DaemonSet", "DaemonSet.Namespace", daemonSet.Namespace, "DaemonSet.Name", daemonSet.Name)
		err = r.Create(ctx, daemonSet)
		if err != nil {
			log.Error(err, "Failed to create new DaemonSet", "DaemonSet.Namespace", daemonSet.Namespace, "DaemonSet.Name", daemonSet.Name)
			return ctrl.Result{}, err
		}
		// DaemonSet created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get DaemonSet")
		return ctrl.Result{}, err
	}

	// Check if the Service already exists, if not create a new one
	service := &corev1.Service{}
	err = r.Get(ctx, types.NamespacedName{Name: routerInstance.Name, Namespace: routerInstance.Namespace}, service)
	if err != nil && errors.IsNotFound(err) {
		// Define a new service
		service := r.serviceForRouterInstance(routerInstance)
		log.Info("Creating a new Service", "Service.Namespace", service.Namespace, "Service.Name", service.Name)
		err = r.Create(ctx, service)
		if err != nil {
			log.Error(err, "Failed to create new Service", "Service.Namespace", service.Namespace, "Service.Name", service.Name)
			return ctrl.Result{}, err
		}
		// Service created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Service")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RouterInstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kasicov1.RouterInstance{}).
		Owns(&appsv1.DaemonSet{}).
		Owns(&corev1.Service{}).
		Complete(r)
}

// daemonSetForRouterInstance returns a kasicoRouter DaemonSet object
func (r *RouterInstanceReconciler) daemonSetForRouterInstance(m *kasicov1.RouterInstance) *appsv1.DaemonSet {
	ls := labelsForDaemonSet(m)

	daemonSet := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name,
			Namespace: m.Namespace,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: "nginx",
						Name:  "kamailio",
						Ports: []corev1.ContainerPort{{
							ContainerPort: 5060,
							Name:          "sip",
						}},
					}},
				},
			},
		},
	}

	// Set RouterInstance as the owner and controller
	ctrl.SetControllerReference(m, daemonSet, r.Scheme)
	return daemonSet
}

// serviceForRouterInstance returns a kasicoRouter Service object
func (r *RouterInstanceReconciler) serviceForRouterInstance(m *kasicov1.RouterInstance) *corev1.Service {
	ls := labelsForDaemonSet(m)

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name,
			Namespace: m.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: ls,
			Ports: []corev1.ServicePort{{
				Port: 5060,
				Name: "sip",
			}},
		},
	}

	// Set RouterInstance as the owner and controller
	ctrl.SetControllerReference(m, service, r.Scheme)
	return service
}

// labelsForDaemonSet returns the labels for selecting the resources
// belonging to the given kasico CR name.
func labelsForDaemonSet(m *kasicov1.RouterInstance) map[string]string {
	return map[string]string{"app": "kasico-router"}
}
