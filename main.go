package main

import (
	"fmt"
	coreV1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	kubeInformers "k8s.io/client-go/informers"
	"k8s.io/client-go/informers/internalinterfaces"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var faasLabels = map[string]string{
	"runtime": "wasm",
	"type":    "faas-wasm",
}

func main() {
	config, err := rest.InClusterConfig()

	if err != nil {
		fmt.Println("load k8s config error")
		os.Exit(1)
	}

	clientset, err := kubernetes.NewForConfig(config)

	if err != nil {
		fmt.Println("create k8s client error")
		os.Exit(1)
	}

	kubeInformerFactory := kubeInformers.NewSharedInformerFactoryWithOptions(
		clientset,
		time.Second*30,
		kubeInformers.WithNamespace(coreV1.NamespaceDefault),
		kubeInformers.WithTweakListOptions(internalinterfaces.TweakListOptionsFunc(filterLabel)),
	)

	configMapsInformer := kubeInformerFactory.Core().V1().ConfigMaps()

	configMapsInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(newConfigMap interface{}) {
			configmap := newConfigMap.(*coreV1.ConfigMap)
			fmt.Printf("%s\t%s\n", configmap.UID, configmap.Name)
		},
	})

	stopCh := setupSignalHandler()
	kubeInformerFactory.Start(stopCh)
	<-stopCh
}

func setupSignalHandler() (stopCh <-chan struct{}) {
	stop := make(chan struct{})
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		close(stop)
		<-c
		os.Exit(1) // second signal. Exit directly.
	}()

	return stop
}

func filterLabel(listOption *v1.ListOptions) {
	listOption.LabelSelector = labels.FormatLabels(faasLabels)
}
