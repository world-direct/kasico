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
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/go-logr/logr"
	kasicov1 "github.com/world-direct/kasico/api/v1"
)

// IngressReconciler reconciles a Ingress object
type IngressReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	Generator Generator
}

//+kubebuilder:rbac:groups=kasico.world-direct.at,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kasico.world-direct.at,resources=ingresses/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kasico.world-direct.at,resources=ingresses/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Ingress object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *IngressReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	log := ctrllog.FromContext(ctx)
	log.V(2).Info("Reconcile Ingress")

	// at startup all ingress objects are reconciled. This would need to be debounced to avoid state
	// configuration on startup.
	// to resolve this, this controller just inserts a reference to itself into the routerinstance.status.ingresses field.
	// this is reconciliated later on, with debouncing

	// https://sdk.operatorframework.io/docs/building-operators/golang/advanced-topics/#handle-cleanup-on-deletion

	// Fetch the Ingress
	ingress := &kasicov1.Ingress{}
	err := r.Get(ctx, req.NamespacedName, ingress)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("Ingress resource not found. Ignoring since object must be deleted.")
			r.Generator.OnObjectsChanged(ctx)
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get Ingress")
		return ctrl.Result{}, err
	}

	// routerInstance, err := r.getRouterInstance(ctx, log, ingress)
	// if err != nil {
	// 	meta.SetStatusCondition(&ingress.Status.Conditions, metav1.Condition{
	// 		Type:    "routerReconciled",
	// 		Status:  metav1.ConditionFalse,
	// 		Reason:  "failed",
	// 		Message: err.Error(),
	// 	})

	// 	err = r.Status().Update(ctx, ingress)
	// 	return ctrl.Result{}, err
	// }

	r.Generator.OnObjectsChanged(ctx)

	// // label the routerinstance so that this will trigger it to be reconciled
	// log.Info("Label the routerinstance to trigger reconciliation", "routerInstance.ingressClassName", routerInstance.Spec.IngressClassName)
	// SetLabel(&routerInstance.ObjectMeta, "reconciled-by", ingress.Namespace+"."+ingress.Name)
	// err = r.Update(ctx, routerInstance)
	// if err != nil {
	// 	return ctrl.Result{Requeue: true, RequeueAfter: 5 * time.Second}, err
	// }

	// meta.SetStatusCondition(&ingress.Status.Conditions, metav1.Condition{
	// 	Type:    "routerReconciled",
	// 	Status:  metav1.ConditionTrue,
	// 	Reason:  "done",
	// 	Message: "reconciliation done",
	// })

	// err = r.Status().Update(ctx, ingress)
	// if err != nil {
	// 	return ctrl.Result{}, err
	// }

	return ctrl.Result{}, nil
}

func (r *IngressReconciler) getRouterInstance(ctx context.Context, log logr.Logger, ingress *kasicov1.Ingress) (*kasicov1.RouterInstance, error) {

	list := &kasicov1.RouterInstanceList{}
	err := r.Client.List(ctx, list)
	if err != nil {
		return nil, err
	}

	for _, routerInstance := range list.Items {
		if routerInstance.Spec.IngressClassName == ingress.Spec.IngressClassName {
			return &routerInstance, nil
		}
	}

	return nil, fmt.Errorf("Unable to find a RouterInstance with ingressClassName='%s'", ingress.Spec.IngressClassName)
}

// SetupWithManager sets up the controller with the Manager.
func (r *IngressReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kasicov1.Ingress{}).
		Complete(r)
}
