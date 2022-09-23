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
	"k8s.io/apimachinery/pkg/api/meta"
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
	Scheme    *runtime.Scheme
	Generator Generator
}

//+kubebuilder:rbac:groups=kasico.world-direct.at,resources=routerinstances,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kasico.world-direct.at,resources=routerinstances/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kasico.world-direct.at,resources=routerinstances/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=daemonSet,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=service,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=configmap,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *RouterInstanceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	// based on https://github.com/operator-framework/operator-sdk/blob/latest/testdata/go/v3/memcached-operator/controllers/memcached_controller.go

	log := ctrllog.FromContext(ctx)
	log.V(2).Info("Reconcile RouterInstance")

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
	err = r.Get(ctx, types.NamespacedName{Name: Name_Daemonset, Namespace: routerInstance.Namespace}, daemonSet)
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
	err = r.Get(ctx, types.NamespacedName{Name: Name_Service, Namespace: routerInstance.Namespace}, service)
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

	// Check if the ConfigMap with the routing-data already exists, if not create a new one
	cmRoutingData := &corev1.ConfigMap{}
	err = r.Get(ctx, types.NamespacedName{Name: Name_ConfigMap, Namespace: routerInstance.Namespace}, cmRoutingData)
	if err != nil && errors.IsNotFound(err) {
		// Define a new ConfigMap
		cm := r.configMapForRouterInstance(routerInstance)
		log.Info("Creating a new ConfigMap", "ConfigMap.Namespace", cm.Namespace, "ConfigMap.Name", cm.Name)
		err = r.Create(ctx, cm)
		if err != nil {
			log.Error(err, "Failed to create new ConfigMap", "ConfigMap.Namespace", cm.Namespace, "ConfigMap.Name", cm.Name)
			return ctrl.Result{}, err
		}
		// ConfigMap created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get ConfigMap")
		return ctrl.Result{}, err
	}

	// check for the templates configmap, and calculate the hash
	cmTemplates := corev1.ConfigMap{}
	err = r.Get(ctx, types.NamespacedName{Name: routerInstance.Spec.TemplateConfigMapName, Namespace: routerInstance.Namespace}, &cmTemplates)
	if err != nil {
		meta.SetStatusCondition(&routerInstance.Status.Conditions, metav1.Condition{
			Type:    "templatesRead",
			Status:  metav1.ConditionFalse,
			Reason:  "getError",
			Message: err.Error(),
		})
	} else {
		hash := HashStringMap(cmTemplates.Data)

		if hash != routerInstance.Status.TemplatesHash {
			log.Info("The hash of the templates have been changed, ...", "oldHash", routerInstance.Status.TemplatesHash, "newHash", hash)
		}

		routerInstance.Status.TemplatesHash = hash
		meta.SetStatusCondition(&routerInstance.Status.Conditions, metav1.Condition{
			Type:   "templatesRead",
			Status: metav1.ConditionTrue,
			Reason: "done",
		})
	}

	// record the reconciliation as a condition
	meta.SetStatusCondition(&routerInstance.Status.Conditions, metav1.Condition{
		Type:    "reconciled",
		Status:  metav1.ConditionTrue,
		Reason:  "done",
		Message: "reconciliation done",
	})

	routerInstance.Status.ConfigurationGeneration = 1

	// update the status
	err = r.Status().Update(ctx, routerInstance)
	if err != nil {
		log.Error(err, "Unable to update the status")
	} else {
		log.Info("Status Update performed", "resourceVersion", routerInstance.ObjectMeta.ResourceVersion)
	}

	r.Generator.OnObjectsChanged(ctx)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RouterInstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kasicov1.RouterInstance{}).
		Owns(&appsv1.DaemonSet{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}

// configMapForRouterInstance returns the router-rules secret for the instance
func (r *RouterInstanceReconciler) configMapForRouterInstance(m *kasicov1.RouterInstance) *corev1.ConfigMap {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      Name_ConfigMap,
			Namespace: m.Namespace,
		},
	}

	// Set RouterInstance as the owner and controller
	ctrl.SetControllerReference(m, cm, r.Scheme)
	return cm
}

// daemonSetForRouterInstance returns a kasicoRouter DaemonSet object
func (r *RouterInstanceReconciler) daemonSetForRouterInstance(m *kasicov1.RouterInstance) *appsv1.DaemonSet {
	ls := labelsForDaemonSet(m)

	ports := []corev1.ContainerPort{}

	if m.Spec.RouterService.UDPPort != 0 {
		ports = append(ports, corev1.ContainerPort{
			ContainerPort: int32(m.Spec.RouterService.UDPPort),
			Protocol:      "UDP",
			Name:          "sip-udp",
		})
	}

	if m.Spec.RouterService.TCPPort != 0 {
		ports = append(ports, corev1.ContainerPort{
			ContainerPort: int32(m.Spec.RouterService.TCPPort),
			Protocol:      "TCP",
			Name:          "sip-tcp",
		})
	}

	kamailioContainer := corev1.Container{
		Image: "nginx",
		Name:  Name_Container_Kamailio,
		Ports: ports,
	}

	daemonSet := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      Name_Daemonset,
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
					Containers: []corev1.Container{kamailioContainer},
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

	ports := []corev1.ServicePort{}

	if m.Spec.RouterService.UDPPort != 0 {
		ports = append(ports, corev1.ServicePort{
			Port:     int32(m.Spec.RouterService.UDPPort),
			Protocol: "UDP",
			Name:     "sip-udp",
		})
	}

	if m.Spec.RouterService.TCPPort != 0 {
		ports = append(ports, corev1.ServicePort{
			Port:     int32(m.Spec.RouterService.TCPPort),
			Protocol: "TCP",
			Name:     "sip-tcp",
		})
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      Name_Service,
			Namespace: m.Namespace,
		},
		Spec: corev1.ServiceSpec{
			ExternalTrafficPolicy: "Local",
			Type:                  "LoadBalancer",
			Selector:              ls,
			Ports:                 ports,
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
