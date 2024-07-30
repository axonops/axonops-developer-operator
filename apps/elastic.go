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

const defaultElasticsearchImage = "docker.elastic.co/elasticsearch/elasticsearch"
const defaultElasticsearchTag = "7.17.0"

const elasticsearchServiceTemplate = `
apiVersion: v1
kind: Service
metadata:
  name: es-{{ .Name }}
  namespace: {{ .Namespace }}
spec:
  selector:
    app: es-{{ .Name }}
  ports:
  - protocol: TCP
    port: 9200
    targetPort: 9200
    name: rest
  - protocol: TCP
    port: 9300
    targetPort: 9300
    name: inter-node
`

const elasticsearchTemplate = `
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: es-{{ .Name }}
  namespace: {{ .Namespace }}
spec:
  serviceName: es-{{ .Name }}
  replicas: {{ .Replicas }}
  selector:
    matchLabels:
      app: es-{{ .Name }}
  template:
    metadata:
      labels:
        app: es-{{ .Name }}
    spec:
      initContainers:
      - name: sysctl
        image: busybox:stable
        command: ['sh', '-c', 'sysctl -w vm.max_map_count=262144']
        securityContext:
          privileged: true
          runAsUser: 0
      containers:
      - name: elasticsearch
        image: {{ .Image }}
        ports:
        - containerPort: 9200
          name: rest
        - containerPort: 9300
          name: inter-node
        env:
        - name: cluster.name
          value: {{ .ClusterName }}
        - name: node.name
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: ES_JAVA_OPTS
          value: "{{ .JavaOpts }}"
        - name: discovery.type
          value: single-node
        resources:
          limits:
            cpu: {{ .CpuLimit }}
            memory: {{ .MemoryLimit }}
          requests:
            cpu: {{ .CpuRequest }}
            memory: {{ .MemoryRequest }}
{{- if ne .StorageSize "" }}
        volumeMounts:
        - name: data
          mountPath: /usr/share/elasticsearch/data
  volumeClaimTemplates:
  - metadata:
      name: data
    spec:
      accessModes: ["ReadWriteOnce"]
      {{- if ne .StorageClass "" }}
      storageClassName: {{ .StorageClass }}
      {{- end }}
      resources:
        requests:
          storage: {{ .StorageSize }}
{{- end }}
`

type ElasticsearchServiceConfig struct {
	Name      string
	Namespace string
}

type ElasticsearchConfig struct {
	Name               string
	Namespace          string
	Replicas           int
	Image              string
	ClusterName        string
	SeedHosts          string
	InitialMasterNodes string
	JavaOpts           string
	CpuLimit           string
	MemoryLimit        string
	CpuRequest         string
	MemoryRequest      string
	StorageSize        string
	StorageClass       string
	Persistent         bool
}

func GenerateElasticsearchConfig(cfg cassandraaxonopscomv1beta1.AxonOpsCassandra) (*appsv1.StatefulSet, error) {
	config := ElasticsearchConfig{
		Name:      cfg.GetName(),
		Namespace: cfg.GetNamespace(),
		Replicas:  1,
		Image: fmt.Sprintf("%s:%s",
			utils.ValueOrDefault(cfg.Spec.AxonOps.Elasticsearch.Image.Repository, defaultElasticsearchImage),
			utils.ValueOrDefault(cfg.Spec.AxonOps.Elasticsearch.Image.Tag, defaultElasticsearchTag),
		),
		ClusterName:   utils.ValueOrDefault(cfg.Spec.AxonOps.Elasticsearch.ClusterName, cfg.GetName()),
		JavaOpts:      utils.ValueOrDefault(cfg.Spec.AxonOps.Elasticsearch.JavaOpts, "-Xms512m -Xmx512m"),
		CpuLimit:      "1000m",
		MemoryLimit:   "2Gi",
		CpuRequest:    "100m",
		MemoryRequest: "1Gi",
		StorageSize:   utils.ValueOrDefault(cfg.Spec.AxonOps.Elasticsearch.PersistentVolume.Size, ""),
		StorageClass:  utils.ValueOrDefault(cfg.Spec.AxonOps.Elasticsearch.PersistentVolume.StorageClass, ""),
	}

	statefulSet := &appsv1.StatefulSet{}
	b := bytes.NewBuffer(nil)
	tmpl, err := template.New("elasticsearch").Funcs(sprig.FuncMap()).Parse(elasticsearchTemplate)
	if err != nil {
		return statefulSet, err
	}

	err = tmpl.Execute(b, config)
	if err != nil {
		return statefulSet, err
	}

	obj := &unstructured.Unstructured{}
	es := yaml.NewYAMLOrJSONDecoder(b, 500)
	if err := es.Decode(obj); err != nil {
		return statefulSet, err
	}

	err = runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, statefulSet)
	if err != nil {
		return statefulSet, err
	}
	return statefulSet, nil
}

func GenerateElasticsearchServiceConfig(cfg cassandraaxonopscomv1beta1.AxonOpsCassandra) (*corev1.Service, error) {
	config := ElasticsearchServiceConfig{
		Name:      cfg.GetName(),
		Namespace: cfg.GetNamespace(),
	}

	svc := &corev1.Service{}
	b := bytes.NewBuffer(nil)
	tmpl, err := template.New("elasticsearch").Funcs(sprig.FuncMap()).Parse(elasticsearchServiceTemplate)
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
