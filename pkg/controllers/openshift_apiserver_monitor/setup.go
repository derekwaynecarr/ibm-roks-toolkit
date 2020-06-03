package openshift_apiserver_monitor

import (
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/openshift/ibm-roks-toolkit/pkg/cmd/cpoperator"
	"github.com/openshift/ibm-roks-toolkit/pkg/controllers"
)

func Setup(cfg *cpoperator.ControlPlaneOperatorConfig) error {
	apiextClient, err := apiextensionsclient.NewForConfig(cfg.TargetConfig())
	if err != nil {
		return err
	}
	crdInformer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(opt metav1.ListOptions) (runtime.Object, error) {
				return apiextClient.CustomResourceDefinitions().List(opt)
			},
			WatchFunc: func(opt metav1.ListOptions) (watch.Interface, error) {
				return apiextClient.CustomResourceDefinitions().Watch(opt)
			},
		},
		&apiextensionsv1.CustomResourceDefinition{},
		controllers.DefaultResync,
		cache.Indexers{},
	)
	cfg.Manager().Add(manager.RunnableFunc(func(stopCh <-chan struct{}) error {
		crdInformer.Run(stopCh)
		return nil
	}))
	reconciler := &OpenshiftAPIServerMonitor{
		KubeClient: cfg.KubeClient(),
		Namespace:  cfg.Namespace(),
		Log:        cfg.Logger().WithName("OpenshiftAPIServerMonitor"),
	}
	c, err := controller.New("openshift-apiserver-monitor", cfg.Manager(), controller.Options{Reconciler: reconciler})
	if err != nil {
		return err
	}
	if err := c.Watch(&source.Informer{Informer: crdInformer}, &handler.EnqueueRequestForObject{}); err != nil {
		return err
	}
	return nil
}
