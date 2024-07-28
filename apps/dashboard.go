package apps

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/Masterminds/sprig"
	cassandraaxonopscomv1beta1 "github.com/axonops/axonops-developer-operator/api/v1beta1"
	"github.com/axonops/axonops-developer-operator/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
)

const defaultDashboardImage = "registry.axonops.com/axonops-public/axonops-docker/axon-dash"
const defaultDashboardTag = "latest"

const DashboardServiceTemplate = `
apiVersion: v1
kind: Service
metadata:
  name: ds-{{ .Name }}
  namespace: {{ .Namespace }}
spec:
  selector:
    app: ds-{{ .Name }}
  ports:
  - protocol: TCP
    port: 3000
    targetPort: 3000
    name: http
`

const DashboardTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ds-{{ .Name }}
  namespace: {{ .Namespace }}
spec:
  serviceName: ds-{{ .Name }}
  replicas: {{ .Replicas }}
  selector:
    matchLabels:
      app: ds-{{ .Name }}
  template:
    metadata:
      labels:
        app: ds-{{ .Name }}
    spec:
      containers:
      - name: axon-dash
        command:
        - /bin/sh
        - -c
        - "sed -i 's|private_endpoints.*|private_endpoints: http://as-{{ .Name }}:8080|' /etc/axonops/axon-dash.yml && /usr/share/axonops/axon-dash --appimage-extract-and-run"
        image: {{ .Image }}
        ports:
        - containerPort: 3000
          name: http
        env:
        - name: node.name
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        resources:
          limits:
            cpu: {{ .CpuLimit }}
            memory: {{ .MemoryLimit }}
          requests:
            cpu: {{ .CpuRequest }}
            memory: {{ .MemoryRequest }}
`

type DashboardServiceConfig struct {
	Name      string
	Namespace string
}

type DashboardConfig struct {
	Name          string
	Namespace     string
	Replicas      int
	Image         string
	CpuLimit      string
	MemoryLimit   string
	CpuRequest    string
	MemoryRequest string
}

func GenerateDashboardConfig(cfg cassandraaxonopscomv1beta1.AxonOpsCassandra) (*appsv1.Deployment, error) {
	config := DashboardConfig{
		Name:      cfg.GetName(),
		Namespace: cfg.GetNamespace(),
		Replicas:  1,
		Image: fmt.Sprintf("%s:%s",
			utils.ValueOrDefault(cfg.Spec.AxonOps.Dashboard.Image.Repository, defaultDashboardImage),
			utils.ValueOrDefault(cfg.Spec.AxonOps.Dashboard.Image.Tag, defaultDashboardTag),
		),
		CpuLimit:      "1000m",
		MemoryLimit:   "512Mi",
		CpuRequest:    "100m",
		MemoryRequest: "256Mi",
	}

	Deployment := &appsv1.Deployment{}
	b := bytes.NewBuffer(nil)
	tmpl, err := template.New("Dashboard").Funcs(sprig.FuncMap()).Parse(DashboardTemplate)
	if err != nil {
		return Deployment, err
	}

	err = tmpl.Execute(b, config)
	if err != nil {
		return Deployment, err
	}

	obj := &unstructured.Unstructured{}
	es := yaml.NewYAMLOrJSONDecoder(b, 500)
	if err := es.Decode(obj); err != nil {
		return Deployment, err
	}

	err = runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, Deployment)
	if err != nil {
		return Deployment, err
	}
	return Deployment, nil
}

func GenerateDashboardServiceConfig(cfg cassandraaxonopscomv1beta1.AxonOpsCassandra) (*corev1.Service, error) {
	config := DashboardServiceConfig{
		Name:      cfg.GetName(),
		Namespace: cfg.GetNamespace(),
	}

	svc := &corev1.Service{}
	b := bytes.NewBuffer(nil)
	tmpl, err := template.New("Dashboard").Funcs(sprig.FuncMap()).Parse(DashboardServiceTemplate)
	if err != nil {
		return svc, err
	}

	err = tmpl.Execute(b, config)
	if err != nil {
		return svc, err
	}

	obj := &unstructured.Unstructured{}
	es := yaml.NewYAMLOrJSONDecoder(b, 500)
	if err := es.Decode(obj); err != nil {
		return svc, err
	}

	err = runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, svc)
	if err != nil {
		return svc, err
	}
	return svc, nil
}
