package main

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"

	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	corev1 "k8s.io/api/core/v1"
)

func main() {

	var kubeconfig *string

	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")

	}

	namespace := flag.String("namespace", "", "The namespace to watch")

	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Panic(err.Error())
	}

	// stop signal for the informer
	stopper := make(chan struct{})
	defer close(stopper)

	factory := informers.NewSharedInformerFactoryWithOptions(
		clientset,
		0,
		informers.WithNamespace(*namespace),
	)

	cmInformer := factory.Core().V1().ConfigMaps()
	informer := cmInformer.Informer()

	defer runtime.HandleCrash()

	// start informer ->
	go factory.Start(stopper)

	// start to sync and call list
	if !cache.WaitForCacheSync(stopper, informer.HasSynced) {
		runtime.HandleError(fmt.Errorf("Timed out waiting for caches to sync"))
		return
	}

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { onChange(obj.(*corev1.ConfigMap), "add") },
		UpdateFunc: func(_ interface{}, newObj interface{}) { onChange(newObj.(*corev1.ConfigMap), "update") },
		DeleteFunc: func(obj interface{}) { onChange(obj.(*corev1.ConfigMap), "delete") },
	})

	<-stopper
}

func onChange(obj *corev1.ConfigMap, changeType string) {
	fmt.Printf("onChange (%s): %v\n", changeType, obj.Name)
}
