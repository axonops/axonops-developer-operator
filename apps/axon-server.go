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

const defaultServerImage = "registry.axonops.com/axonops-public/axonops-docker/axon-server"
const defaultServerTag = "latest"

const ServerServiceTemplate = `
apiVersion: v1
kind: Service
metadata:
  name: as-{{ .Name }}
  namespace: {{ .Namespace }}
spec:
  selector:
    app: as-{{ .Name }}
  ports:
  - protocol: TCP
    port: 8080
    targetPort: 8080
    name: api
  - protocol: TCP
    port: 1888
    targetPort: 1888
    name: agent
  - protocol: TCP
    port: 6060
    targetPort: 6060
    name: metrics
`

const ServerTemplate = `
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: as-{{ .Name }}
  namespace: {{ .Namespace }}
spec:
  serviceName: as-{{ .Name }}
  replicas: {{ .Replicas }}
  selector:
    matchLabels:
      app: as-{{ .Name }}
  template:
    metadata:
      labels:
        app: as-{{ .Name }}
    spec:
      containers:
      - name: axon-server
        image: {{ .Image }}
        ports:
        - name: api
          containerPort: 8080
        - name: agent
          containerPort: 1888
        - name: metrics
          containerPort: 6060
        env:
        - name: ELASTIC_HOSTS
          value: http://es-{{ .Name }}:9200
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

type ServerServiceConfig struct {
	Name      string
	Namespace string
}

type ServerConfig struct {
	Name          string
	Namespace     string
	Replicas      int
	Image         string
	CpuLimit      string
	MemoryLimit   string
	CpuRequest    string
	MemoryRequest string
}

func GenerateServerConfig(cfg cassandraaxonopscomv1beta1.AxonOpsCassandra) (*appsv1.StatefulSet, error) {
	config := ServerConfig{
		Name:      cfg.GetName(),
		Namespace: cfg.GetNamespace(),
		Replicas:  1,
		Image: fmt.Sprintf("%s:%s",
			utils.ValueOrDefault(cfg.Spec.AxonOps.Server.Image.Repository, defaultServerImage),
			utils.ValueOrDefault(cfg.Spec.AxonOps.Server.Image.Tag, defaultServerTag),
		),
		CpuLimit:      "1000m",
		MemoryLimit:   "512Mi",
		CpuRequest:    "100m",
		MemoryRequest: "256Mi",
	}

	StatefulSet := &appsv1.StatefulSet{}
	b := bytes.NewBuffer(nil)
	tmpl, err := template.New("Server").Funcs(sprig.FuncMap()).Parse(ServerTemplate)
	if err != nil {
		return StatefulSet, err
	}

	err = tmpl.Execute(b, config)
	if err != nil {
		return StatefulSet, err
	}

	obj := &unstructured.Unstructured{}
	es := yaml.NewYAMLOrJSONDecoder(b, 500)
	if err := es.Decode(obj); err != nil {
		return StatefulSet, err
	}

	err = runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, StatefulSet)
	if err != nil {
		return StatefulSet, err
	}
	return StatefulSet, nil
}

func GenerateServerServiceConfig(cfg cassandraaxonopscomv1beta1.AxonOpsCassandra) (*corev1.Service, error) {
	config := ServerServiceConfig{
		Name:      cfg.GetName(),
		Namespace: cfg.GetNamespace(),
	}

	svc := &corev1.Service{}
	b := bytes.NewBuffer(nil)
	tmpl, err := template.New("Server").Funcs(sprig.FuncMap()).Parse(ServerServiceTemplate)
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
