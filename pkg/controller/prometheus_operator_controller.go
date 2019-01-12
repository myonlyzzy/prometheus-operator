package controller

import (
	"fmt"
	"github.com/myonlyzzy/prometheus-operator/pkg/apis/prometheus.io/v1alpha1"
	"github.com/myonlyzzy/prometheus-operator/pkg/client/clientset/versioned"
	pv1alpha1 "github.com/myonlyzzy/prometheus-operator/pkg/client/informers/externalversions/prometheus.io/v1alpha1"
	listers "github.com/myonlyzzy/prometheus-operator/pkg/client/listers/prometheus.io/v1alpha1"
	appsv1 "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	appsinformers "k8s.io/client-go/informers/apps/v1"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	apps "k8s.io/client-go/listers/apps/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
	"time"
)

const (
	ServiceName = "prometheus"
)

type PrometheusController struct {
	client           kubernetes.Interface
	clientSet        versioned.Interface
	setLister        apps.StatefulSetLister
	svcLister        corelisters.ServiceLister
	prometheusLister listers.PrometheusLister
	recorder         record.EventRecorder
	workqueue        workqueue.RateLimitingInterface
	setSynced        cache.InformerSynced
	prometheusSynced cache.InformerSynced
}

//NewPrometheusController return a Prometheus controller
func NewPrometheusController(client kubernetes.Interface, clientset versioned.Interface, setInformer appsinformers.StatefulSetInformer, svcInformers coreinformers.ServiceInformer, prometheusInformer pv1alpha1.PrometheusInformer) *PrometheusController {

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: client.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(v1alpha1.Scheme, corev1.EventSource{Component: "prometheus"})
	p := &PrometheusController{
		client:           client,
		clientSet:        clientset,
		recorder:         recorder,
		workqueue:        workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "prometheus"),
		setLister:        setInformer.Lister(),
		svcLister:        svcInformers.Lister(),
		prometheusLister: prometheusInformer.Lister(),
		prometheusSynced: prometheusInformer.Informer().HasSynced,
		setSynced:        setInformer.Informer().HasSynced,
	}

	prometheusInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: p.enqueue,
	})

	return p

}

func (p *PrometheusController) Run(workers int, stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer p.workqueue.ShutDown()
	klog.Info("Starting prometheus controller")
	klog.Info("Wating for informer  caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, p.setSynced, p.prometheusSynced); !ok {
		return
	}
	klog.Info("Starting workers")
	for i := 0; i < workers; i++ {
		go wait.Until(p.Worker, 10*time.Second, stopCh)
	}
	klog.Info("Started workers")
	<-stopCh
	klog.Info("Shutting down workers")

}
func (p *PrometheusController) Worker() {
	for p.processNextWorkItem() {
	}

}
func (p *PrometheusController) processNextWorkItem() bool {
	key, quit := p.workqueue.Get()
	if quit {
		return false
	}
	defer p.workqueue.Done(key)
	if err := p.sync(key.(string)); err != nil {
		utilruntime.HandleError(fmt.Errorf("Error syncing Prometheus %v,requeue:%v", key.(string), err))
		p.workqueue.AddRateLimited(key)
	} else {
		p.workqueue.Forget(key)
	}
	return true
}

//convert a prometheus resource to namespace/name and  add to workqueue
func (p *PrometheusController) enqueue(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	p.workqueue.AddRateLimited(key)
}

//sync prometheus reource
func (p *PrometheusController) sync(key string) error {
	startTime := time.Now()
	defer func() {
		klog.V(4).Infof("Finished syncing prometheus %q (%v)", key, time.Since(startTime))
	}()
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}
	prometheus, err := p.prometheusLister.Prometheuses(namespace).Get(name)
	if errors.IsNotFound(err) {
		klog.Infof("Prometheus has been deleted %v", key)
		return nil
	}

	klog.Infof("sync prometheus %v\n ", prometheus)
	if err := p.syncService(prometheus); err != nil {
		return err
	}
	if err := p.syncStatefulSet(prometheus); err != nil {
		return err
	}
	return nil
}

//sync  service
func (p *PrometheusController) syncService(prometheus *v1alpha1.Prometheus) error {
	namespace := prometheus.GetNamespace()
	name := prometheus.GetName()
	_, err := p.svcLister.Services(namespace).Get(name)
	if errors.IsNotFound(err) {
		err := p.CreateService(prometheus)
		if err != nil {
			return err
		}

	}
	return err
}

//sync  statefulset
func (p *PrometheusController) syncStatefulSet(prometheus *v1alpha1.Prometheus) error {
	namespace := prometheus.GetNamespace()
	name := prometheus.GetName()
	if prometheus.Spec.StatefulSet == nil {
		if err := p.client.AppsV1().StatefulSets(namespace).Delete(name, &metav1.DeleteOptions{}); err != nil {
			return err
		}
	}
	_, err := p.setLister.StatefulSets(namespace).Get(name)
	if errors.IsNotFound(err) {
		err := p.CreateStatefulset(prometheus)
		if err != nil {
			return err
		}
	}

	return err
}

//update prometheus statefulset
func (p *PrometheusController) UpdateStatefulSet(prometheus *v1alpha1.Prometheus) {

}

func (p *PrometheusController) CreateService(prometheus *v1alpha1.Prometheus) error {
	nameSpace := prometheus.GetNamespace()
	svc := p.NewPrometheusService(prometheus)
	_, err := p.client.CoreV1().Services(nameSpace).Create(svc)
	if apierrors.IsAlreadyExists(err) {
		return err
	}
	if err != nil {
		p.recorder.Event(prometheus, corev1.EventTypeNormal, "failed", fmt.Sprintln("Create prometheus service failed "))
	} else {
		p.recorder.Event(prometheus, corev1.EventTypeWarning, "success", fmt.Sprintln(" Successful create prometheus service"))
	}
	return err
}
func (p *PrometheusController) CreateStatefulset(prometheus *v1alpha1.Prometheus) error {
	nameSpace := prometheus.GetNamespace()
	set := p.NewPrometheusStatefulSet(prometheus)
	_, err := p.client.AppsV1beta1().StatefulSets(nameSpace).Create(set)
	if apierrors.IsAlreadyExists(err) {
		return err
	}
	if err != nil {
		p.recorder.Event(prometheus, corev1.EventTypeNormal, "success", fmt.Sprintln("Successful create prometheus service "))
	} else {
		p.recorder.Event(prometheus, corev1.EventTypeWarning, "failed", fmt.Sprintln(" Create prometheus service failed "))
	}
	return err
}

func (p *PrometheusController) NewPrometheusService(prometheus *v1alpha1.Prometheus) *corev1.Service {
	labels := make(map[string]string)
	labels["app"] = "prometheus"
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      prometheus.Name,
			Namespace: prometheus.Namespace,
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{
				metav1.OwnerReference{
					Name:       prometheus.GetName(),
					Kind:       prometheus.Kind,
					APIVersion: prometheus.APIVersion,
					UID:        prometheus.GetUID(),
				},
			},
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "None",
			Ports: []corev1.ServicePort{

				{
					Name:       "http",
					Port:       9090,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt(9090),
				},
			},
			Selector: map[string]string{
				"app": "prometheus",
			},
		},
	}
	return svc
}
func (p *PrometheusController) NewPrometheusStatefulSet(prometheus *v1alpha1.Prometheus) *appsv1.StatefulSet {
	labels := map[string]string{"app": "prometheus"}
	initVolumeMounts := []corev1.VolumeMount{
		corev1.VolumeMount{
			Name:      "prometheus-data",
			MountPath: "/data",
		},
	}
	reloadVolumeMounts := []corev1.VolumeMount{
		corev1.VolumeMount{
			Name:      "config-volume",
			MountPath: "/etc/config",
			ReadOnly:  true,
		},
	}
	prometheusVolumeMounts := []corev1.VolumeMount{
		corev1.VolumeMount{
			Name:      "config-volume",
			MountPath: "/etc/config",
		},
		corev1.VolumeMount{
			Name:      "prometheus-data",
			MountPath: "/data",
		},
	}
	var probe = &corev1.Probe{}
	probe.Handler = corev1.Handler{
		HTTPGet: &corev1.HTTPGetAction{
			Path: "/-/ready",
			Port: intstr.FromInt(9090),
		},
	}
	probe.InitialDelaySeconds = 30
	probe.TimeoutSeconds = 30
	var volume corev1.Volume
	volume.ConfigMap = &corev1.ConfigMapVolumeSource{}
	volume.ConfigMap.Name = "prometheus-config"
	volume.Name = "config-volume"
	/*	pvc := corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: "prometheus-data",
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			StorageClassName: func() *string { name := "standard"; return &name }(),
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.PersistentVolumeAccessMode(corev1.ReadWriteOnce),
			},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("16Gi"),
				},
			},
		},
	}*/
	setSpec := appsv1.StatefulSetSpec{
		ServiceName: prometheus.Name,
		Replicas:    prometheus.Spec.StatefulSet.Replicas,
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{"app": "prometheus"},
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{"app": "prometheus"},
			},
			Spec: corev1.PodSpec{
				InitContainers: []corev1.Container{
					corev1.Container{
						Name:            "init-chown-data",
						Image:           prometheus.Spec.StatefulSet.InitImage,
						ImagePullPolicy: prometheus.Spec.StatefulSet.ImagePullPolicy,
						Command:         []string{"chown", "-R", "65534:65534", "/data"},
						VolumeMounts:    initVolumeMounts,
					},
				},
				Containers: []corev1.Container{
					corev1.Container{
						Name:            "prometheus-server-configmap-reload",
						Image:           prometheus.Spec.StatefulSet.ReloadImage,
						ImagePullPolicy: prometheus.Spec.StatefulSet.ImagePullPolicy,
						Args: []string{
							"--volume-dir=/etc/config",
							"--webhook-url=http://localhost:9090/-/reload",
						},
						VolumeMounts: reloadVolumeMounts,
						Resources:    NewContainerResourceRequirements("10m", "10m", "10Mi", "10Mi"),
					},
					corev1.Container{
						Name:            "prometheus-server",
						ImagePullPolicy: prometheus.Spec.StatefulSet.ImagePullPolicy,
						Image:           prometheus.Spec.StatefulSet.PrometheusImage,
						Args: []string{
							"--config.file=/etc/config/prometheus.yml",
							"--storage.tsdb.path=/data",
							"--web.console.libraries=/etc/prometheus/console_libraries",
							"--web.console.templates=/etc/prometheus/consoles",
							"--web.enable-lifecycle",
						},
						Ports: []corev1.ContainerPort{
							corev1.ContainerPort{
								ContainerPort: 9090,
							},
						},
						Resources:      NewContainerResourceRequirements("200m", "200m", "1000Mi", "1000Mi"),
						VolumeMounts:   prometheusVolumeMounts,
						LivenessProbe:  probe,
						ReadinessProbe: probe,
					},
				},
				Volumes: []corev1.Volume{
					volume,
				},
			},
		},
		VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
			prometheus.Spec.StatefulSet.Storage.VolumeClaimTemplate,
		},
	}
	set := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      prometheus.Name,
			Namespace: prometheus.Namespace,
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{
				metav1.OwnerReference{
					Name:       prometheus.GetName(),
					Kind:       prometheus.Kind,
					APIVersion: prometheus.APIVersion,
					UID:        prometheus.GetUID(),
				},
			},
		},
		Spec: setSpec,
	}
	return set
}

func NewContainerResourceRequirements(cpuLimit, cpuRequest, memLimit, memRequest string) corev1.ResourceRequirements {
	r := corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse(cpuLimit),
			corev1.ResourceMemory: resource.MustParse(memLimit),
		},
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse(cpuRequest),
			corev1.ResourceMemory: resource.MustParse(memRequest),
		},
	}
	return r
}
