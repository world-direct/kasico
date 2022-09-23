package controllers

import (
	"context"
	"time"

	"github.com/world-direct/kasico/controllers/debounce"
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
	client client.Client
	f      func(f func())
}

type Generator interface {
	Start(ctx context.Context) error
	OnObjectsChanged(ctx context.Context)
}

func NewGenerator(client client.Client, debounceTime time.Duration) Generator {
	generator := &generator{
		client: client,
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
	log.Info("GENERATOR!")
}
