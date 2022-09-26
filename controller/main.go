package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"text/template"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	corev1 "k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
)

var stopper chan struct{}

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func main() {

	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		HealthProbeBindAddress: probeAddr,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}

	var kubeconfig *string

	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}

	namespace := flag.String("namespace", "", "The namespace to watch")
	dataConfigMap := flag.String("cm-data", "routing-data", "The name of the routing-data configmap, default 'routing-data'")
	templateConfigMap := flag.String("cm-templates", "", "The name of the routing-data configmap")
	configDir := flag.String("configDirectory", "", "The name of the directory to emit the configuration")
	// mode := flag.String("mode", "", "init=generate one time and exit / watch=watch for changes in background")

	var cmTemplates *corev1.ConfigMap
	var cmData *corev1.ConfigMap

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
	stopper = make(chan struct{})
	defer close(stopper)

	factory := informers.NewSharedInformerFactoryWithOptions(
		clientset,
		0,
		informers.WithNamespace(*namespace),
	)

	cmInformer := factory.Core().V1().ConfigMaps()
	informer := cmInformer.Informer()

	defer utilruntime.HandleCrash()

	// start informer ->
	go factory.Start(stopper)

	// start to sync and call list
	if !cache.WaitForCacheSync(stopper, informer.HasSynced) {
		utilruntime.HandleError(fmt.Errorf("Timed out waiting for caches to sync"))
		return
	}

	onChange := func(obj *corev1.ConfigMap, changeType string) {
		fmt.Printf("onChange (%s): %v\n", changeType, obj.Name)

		relevantChange := false

		if obj.Name == *dataConfigMap {
			cmData = obj
			relevantChange = true
		}

		if obj.Name == *templateConfigMap {
			cmTemplates = obj
			relevantChange = true
		}

		if relevantChange && cmData != nil && cmTemplates != nil {
			generate(cmTemplates.Data, cmData.Data["routing-data.json"], *configDir)
		}
	}

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { onChange(obj.(*corev1.ConfigMap), "add") },
		UpdateFunc: func(_ interface{}, newObj interface{}) { onChange(newObj.(*corev1.ConfigMap), "update") },
		DeleteFunc: func(obj interface{}) { onChange(obj.(*corev1.ConfigMap), "delete") },
	})

	fmt.Println("## start read")
	<-stopper
	fmt.Println("## read done")
}

func generate(templates map[string]string, routingDataJson string, outputDirectory string) {
	fmt.Printf("Generating routing-data to %s\n", outputDirectory)

	var data interface{}
	var err error

	err = json.Unmarshal([]byte(routingDataJson), &data)
	if err != nil {
		fmt.Print(err)
	}

	for name, definition := range templates {
		path := filepath.Join(outputDirectory, name)
		file, err := os.Create(path)

		if err != nil {
			panic(err)
		}

		templ := template.Must(template.ParseFiles(definition))
		templ.Execute(file, data)

		fmt.Printf("	written %s\n", path)

	}
}
