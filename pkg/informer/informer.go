package informer

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

// MultiInformer watches an arbitrary list of resources and exposes
// thread-safe getters backed by the informers' local caches.
type MultiInformer struct {
	factory   dynamicinformer.DynamicSharedInformerFactory
	informers map[schema.GroupVersionResource]cache.SharedIndexInformer
	synced    []cache.InformerSynced
}

func NewMultiInformer(
	cfg *rest.Config,
	resync time.Duration,
	gvrs []schema.GroupVersionResource,
	namespace string,
	tweak dynamicinformer.TweakListOptionsFunc,
) (*MultiInformer, error) {
	dynamicClient, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(
		dynamicClient,
		resync,
		namespace,
		tweak,
	)

	mi := &MultiInformer{
		factory:   factory,
		informers: make(map[schema.GroupVersionResource]cache.SharedIndexInformer),
	}

	for _, gvr := range gvrs {
		inf := factory.ForResource(gvr).Informer()
		mi.informers[gvr] = inf
		mi.synced = append(mi.synced, inf.HasSynced)
	}

	mi.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			log.Debug().Msgf("Object added: %s", getObjectName(obj))
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			log.Debug().Msgf("Object updated: %s", getObjectName(newObj))
		},
		DeleteFunc: func(obj interface{}) {
			log.Debug().Msgf("Object deleted: %s", getObjectName(obj))
		},
	})

	return mi, nil
}

func (mi *MultiInformer) Start(ctx context.Context) {
	mi.factory.Start(ctx.Done())
	mi.WaitForCacheSync(ctx)
	<-ctx.Done() // Block until context is cancelled
}

func (mi *MultiInformer) WaitForCacheSync(ctx context.Context) bool {
	return cache.WaitForCacheSync(ctx.Done(), mi.synced...)
}

func (mi *MultiInformer) GetIndexer(gvr schema.GroupVersionResource) cache.Indexer {
	if inf, ok := mi.informers[gvr]; ok {
		return inf.GetIndexer()
	}
	return nil
}

func (mi *MultiInformer) AddEventHandler(handler cache.ResourceEventHandlerFuncs) error {
	for _, inf := range mi.informers {
		_, err := inf.AddEventHandler(handler)
		if err != nil {
			return err
		}
	}
	return nil
}

func getObjectName(obj any) string {
	if d, ok := obj.(metav1.Object); ok {
		return d.GetName()
	}
	return "unknown"
}
