package controllers

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-logr/logr"
	kasicov1 "github.com/world-direct/kasico/api/v1"
	"github.com/world-direct/kasico/controllers/debounce"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
)

// The generator is reponsible for
//   * Reading all RouterInstance objects
//	 * Reading all Ingress objects
//	 * Map the RoutingData for each RouterInstance
//	 * Serialize the RoutingData into the routing-data configmap
//
//	We use this singleton instead putting this directly into the reconciler
//	basically to
//	  * Handle all Ingress objects at once
//	  * Avoid optimistic update errors because we modify objects outside of their reconcilers

type generator struct {
	Client client.Client
	f      func(f func())
}

type Generator interface {
	Start(ctx context.Context) error
	OnObjectsChanged(ctx context.Context)
}

func NewGenerator(client client.Client, debounceTime time.Duration) Generator {
	generator := &generator{
		Client: client,
		f:      debounce.New(debounceTime),
	}

	return generator
}

/// Here we implement 'manager.Runnable' to yield a seperate task to reconcile all Ingresses at once
func (r *generator) Start(ctx context.Context) error {

	log := ctrllog.FromContext(ctx)
	log.Info("Start Generator runnable")

	<-ctx.Done()
	log.Info("Stop Generator runnable")
	return nil
}

// Call this method whenever something has changed
// The Generator will debounce calls, and reevaluate the cluster state afterwards
func (generator *generator) OnObjectsChanged(ctx context.Context) {
	log := ctrllog.FromContext(ctx)
	log.V(2).Info("Generator notified OnObjectsChanged")

	generator.f(func() { generator.reconcile(context.Background()) })
}

func (generator *generator) reconcile(ctx context.Context) {
	log := ctrllog.FromContext(ctx)

	for retry := 0; retry < 5; retry++ {
		log.Info("Running Generator", "retry", retry)
		err := generator.reconcileImpl(ctx, log)
		if err == nil {
			log.Info("Generator finished")
			return
		}

		log.Error(err, "Error running Generator")
	}

}

func (generator *generator) reconcileImpl(ctx context.Context, log logr.Logger) error {

	var err error
	routers := &kasicov1.RouterInstanceList{}
	err = generator.Client.List(ctx, routers)
	if err != nil {
		return err
	}

	ingresses := &kasicov1.IngressList{}
	err = generator.Client.List(ctx, ingresses)
	if err != nil {
		return err
	}

	for _, router := range routers.Items {
		routingData := GetRoutingData(router, ingresses.Items)

		log = log.WithValues("ingressClassName", router.Spec.IngressClassName)

		routerDataJsonBytes, err := json.MarshalIndent(routingData, "", "  ")
		if err != nil {
			return err
		}

		routerDataMap := make(map[string]string)
		routerDataMap[Name_RouningDataJson] = string(routerDataJsonBytes)
		routerDataHash := HashStringMap(routerDataMap)

		cmRoutingData := &corev1.ConfigMap{}
		err = generator.Client.Get(ctx, types.NamespacedName{Name: Name_ConfigMap, Namespace: router.Namespace}, cmRoutingData)
		if err != nil {
			return err
		}

		existingHash := GetAnnotation(&cmRoutingData.ObjectMeta, Name_AnnotationRoutingDataHash)

		// check if the routerdata has been changed
		if routerDataHash != existingHash {
			log.Info("The hash of the data been changed, updating " + Name_ConfigMap)

			cmRoutingData.Data = routerDataMap
			SetAnnotation(&cmRoutingData.ObjectMeta, Name_AnnotationRoutingDataHash, routerDataHash)
			err = generator.Client.Update(ctx, cmRoutingData)

			if err != nil {
				log.Error(err, "Unable to update the routing-data configmap!")
				return err
			}

			log.Info("Successfully updated the ConfigMap")
		} else {
			log.Info("Nothing has changed")
		}
	}

	return nil

}
