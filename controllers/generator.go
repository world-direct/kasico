package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"text/template"
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
		err = generator.Client.Get(ctx, types.NamespacedName{Name: Name_ConfigMap_RoutingData, Namespace: router.Namespace}, cmRoutingData)
		if err != nil {
			return err
		}

		needGeneration := false

		// check if the routerdata has been changed
		if routerDataHash != GetAnnotation(&cmRoutingData.ObjectMeta, Name_AnnotationRoutingDataHash) {
			log.Info("The hash of the data been changed, updating " + Name_ConfigMap_RoutingData)
			needGeneration = true

			cmRoutingData.Data = routerDataMap
			SetAnnotation(&cmRoutingData.ObjectMeta, Name_AnnotationRoutingDataHash, routerDataHash)
			err = generator.Client.Update(ctx, cmRoutingData)
			if err != nil {
				log.Error(err, "Unable to update the routing-data configmap!")
				return err
			}
		}

		if HashStringMap(router.Spec.KamailioConfigTemplates) != router.Status.TemplatesHash {
			log.Info("The hash of the templates been changed, updating " + Name_ConfigMap_RoutingData)
			needGeneration = true

			router.Status.TemplatesHash = HashStringMap(router.Spec.KamailioConfigTemplates)
			err = generator.Client.Status().Update(ctx, &router)
			if err != nil {
				log.Error(err, "Unable to update the TemplatesHash of the router status!")
				return err
			}
		}

		if needGeneration {

			log.Info("Successfully updated the routing-data ConfigMap, running template generation")

			cmKamailioConfig := &corev1.ConfigMap{}
			err = generator.Client.Get(ctx, types.NamespacedName{Name: Name_ConfigMap_KamailioConfig, Namespace: router.Namespace}, cmKamailioConfig)
			if err != nil {
				return err
			}

			configs, err := GenerateTemplates(ctx, router.Spec.KamailioConfigTemplates, routingData)
			if err != nil {
				log.Error(err, "Error while rendering templates")
			}

			cmKamailioConfig.Data = configs
			err = generator.Client.Update(ctx, cmKamailioConfig)
			if err != nil {
				log.Error(err, "Error while updating kamailio config")
			}

		} else {
			log.Info("Nothing has changed")
		}
	}

	return nil

}

func GenerateTemplates(ctx context.Context, templates map[string]string, data *RoutingData) (output map[string]string, err error) {

	output = make(map[string]string)

	for name, templateSrc := range templates {

		t, err := template.New(name).Parse(templateSrc)
		if err != nil {
			return nil, err
		}

		var buf bytes.Buffer
		err = t.Execute(&buf, data)
		if err != nil {
			return nil, err
		}

		res := buf.String()
		output[name] = res
	}

	return output, nil
}
