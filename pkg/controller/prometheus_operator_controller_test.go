package controller

import (
	"github.com/myonlyzzy/prometheus-operator/pkg/apis/prometheus.io/v1alpha1"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"testing"
)

func TestPrometheusNewStatefulSet(t *testing.T) {
	c := PrometheusController{}
	p := v1alpha1.Prometheus{}
	b, err := ioutil.ReadFile("../../manifests/prometheus.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if err := yaml.Unmarshal(b, &p); err != nil {
		t.Fatal(err)
	}
	get := c.NewPrometheusStatefulSet(&p)
	t.Logf("%v", get)

}
