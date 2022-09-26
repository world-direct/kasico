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

package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	"go.uber.org/zap/zapcore"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/cache"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	kasicov1 "github.com/world-direct/kasico/operator/api/v1"
	"github.com/world-direct/kasico/operator/controllers"
	corev1 "k8s.io/api/core/v1"

	"github.com/spf13/cobra"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(kasicov1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {

	var argDevelopment bool

	rootCmd := &cobra.Command{
		Use:   "kasico",
		Short: "Kasico stands for is KAmailio Sip Ingress COntroller",
	}

	// this flag is registered directly to the commandline flags by the controller-runtime
	// https://github.com/kubernetes-sigs/controller-runtime/blob/master/pkg/client/config/config.go#L39
	// we don't do anything with this flag here, it's just so that cobra knows about it, and generated
	// correct documentation with --help, or completion
	_ = rootCmd.Flags().String("kubeconfig", "", "Paths to a kubeconfig. Only required if out-of-cluster.")
	rootCmd.Flags().BoolVar(&argDevelopment, "development", false, "Enables development mode incl verbose logging")

	operatorCmd := &cobra.Command{
		Use:   "operator",
		Short: "Runs the kasico operator",
	}

	controllerCmd := &cobra.Command{
		Use:   "controller",
		Short: "Runs the kasico controller",
	}

	controllerWatchCmd := &cobra.Command{
		Use:   "watch",
		Short: "Runs the kasico controller in watch mode",
	}

	controllerGenerateCmd := &cobra.Command{
		Use:   "generate",
		Short: "Runs the kasico controller in generation mode",
	}

	controllerCmd.AddCommand(controllerWatchCmd)
	controllerCmd.AddCommand(controllerGenerateCmd)

	rootCmd.AddCommand(operatorCmd)
	rootCmd.AddCommand(controllerCmd)

	rootCmd.Execute()

	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var mode string

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false, "Enable leader election for controller manager. ")

	opts := zap.Options{
		Development: argDevelopment,
		TimeEncoder: zapcore.RFC3339TimeEncoder,
	}

	if argDevelopment {
		opts.Level = zapcore.Level(-2)
	}

	// we really don't want to provide all these options, as they don't bring really value
	// we will reduce this to a general "debug" argument
	// opts.BindFlags(flag.CommandLine)	// this is also not very compatible to cobra
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	if mode != "operator" {
		enableLeaderElection = false
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "kasico-lock.world-direct.at",
		// LeaderElectionReleaseOnCancel defines if the leader should step down voluntarily
		// when the Manager ends. This requires the binary to immediately end when the
		// Manager is stopped, otherwise, this setting is unsafe. Setting this significantly
		// speeds up voluntary leader transitions as the new leader don't have to wait
		// LeaseDuration time first.
		//
		// In the default scaffold provided, the program ends immediately after
		// the manager stops, so would be fine to enable this option. However,
		// if you are doing or is intended to do any operation such as perform cleanups
		// after the manager stops then its usage might be unsafe.
		// LeaderElectionReleaseOnCancel: true,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	switch mode {
	case "operator":
		main_operator(mgr)

	case "watcher":
		main_watcher(mgr)
	}
}

func main_generator(mgr ctrl.Manager) {
	setupLog.Info("Starting Kasico Generator")

}

func main_watcher(mgr ctrl.Manager) {
	setupLog.Info("Starting Kasico Watcher")
	ctx := ctrl.SetupSignalHandler()

	informer, err := mgr.GetCache().GetInformer(ctx, &corev1.ConfigMap{})
	if err != nil {
		setupLog.Error(err, "Unable to get informer")
	}

	onChange := func(obj *corev1.ConfigMap, changeType string) {
		fmt.Printf("onChange (%s): %v\n", changeType, obj.Name)
	}

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) {},
		UpdateFunc: func(_ interface{}, newObj interface{}) { onChange(newObj.(*corev1.ConfigMap), "update") },
		DeleteFunc: func(obj interface{}) {},
	})

	mgr.Start(ctx)

}

func main_operator(mgr ctrl.Manager) {

	var err error

	genenerator := controllers.NewGenerator(mgr.GetClient(), time.Second*5)

	if err = (&controllers.RouterInstanceReconciler{
		Client:    mgr.GetClient(),
		Scheme:    mgr.GetScheme(),
		Generator: genenerator,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "RouterInstance")
		os.Exit(1)
	}

	if err = (&controllers.IngressReconciler{
		Client:    mgr.GetClient(),
		Scheme:    mgr.GetScheme(),
		Generator: genenerator,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Ingress")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	mgr.Add(genenerator)

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
