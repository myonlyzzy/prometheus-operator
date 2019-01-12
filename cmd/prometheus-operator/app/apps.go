package app

import (
	"github.com/myonlyzzy/prometheus-operator/pkg/client/clientset/versioned"
	promInformer "github.com/myonlyzzy/prometheus-operator/pkg/client/informers/externalversions"
	"github.com/myonlyzzy/prometheus-operator/pkg/controller"
	"github.com/spf13/cobra"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
	"k8s.io/klog/glog"
	"time"
)

func NewControllerManagerCommand() *cobra.Command {
	var stopChan <-chan struct{} = make(chan struct{})
	cmd := &cobra.Command{
		Use: "prometheus-operator-controller-manager",
		Run: func(cmd *cobra.Command, args []string) {
			Run(stopChan)
		},
	}
	return cmd

}

func Run(stopCh <-chan struct{}) error {
	conf, err := rest.InClusterConfig()
	if err != nil {
		glog.Fatalf("failed get config %v ", err)
		return err
	}
	cli, err := versioned.NewForConfig(conf)
	if err != nil {
		glog.Fatal("failed create client %v", err)
	}
	kubecli, err := kubernetes.NewForConfig(conf)
	if err != nil {
		klog.Fatal("failed creat client %v", err)
	}
	kubeFactory := informers.NewSharedInformerFactory(kubecli, time.Second*10)
	prometheusFactory := promInformer.NewSharedInformerFactory(cli, time.Second*10)
	controller := controller.NewPrometheusController(kubecli, cli, kubeFactory.Apps().V1().StatefulSets(), kubeFactory.Core().V1().Services(), prometheusFactory.Prometheus().V1alpha1().Prometheuses())
	kubeFactory.Start(stopCh)
	prometheusFactory.Start(stopCh)
	controller.Run(2, stopCh)

	return nil
}
