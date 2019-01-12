package v1alpha1

import "k8s.io/apimachinery/pkg/apis/meta/v1"
import corev1 "k8s.io/api/core/v1"

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

//Prometheus Resource
type Prometheus struct {
	v1.TypeMeta   `json:",inline"`
	v1.ObjectMeta `json:"metadata",omitempty`
	Spec          PrometheusSpec    `json:"spec"`
	Status        *PrometheusStatus `json:"status omitempty"`
}

// Prometheus Spec
type PrometheusSpec struct {
	StatefulSet *PrometheusStatefulSet `json:"statefulset"`
	Status      *PrometheusStatus      `json:"status omitempty"`
}

// Prometheus Statefulset
type PrometheusStatefulSet struct {
	Replicas        *int32             `json:""replicas`
	PrometheusImage string             `json:"image"`
	InitImage       string             `json:"initImage"`
	ImagePullPolicy corev1.PullPolicy  `json:"imagePullPolicy,omitempty"`
	ReloadImage     string             `json:"reloadImage"`
	Storage         *PrometheusStorage `json:storage`
}

// Prometheus StorageClass

type PrometheusStorage struct {
	VolumeClaimTemplate corev1.PersistentVolumeClaim `json:volumeClaimTemplate`
}

//Prometheus Status
type PrometheusStatus struct {
	AvailableReplicas int32 `json:"availableReplicas omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
//PrometheusList  is a list of Prometheus resources

type PrometheusList struct {
	v1.TypeMeta `json:",inline"`
	v1.ListMeta `json:"metadata"`
	Items       []Prometheus `json:"items"`
}
