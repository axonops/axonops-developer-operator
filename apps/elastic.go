/*
 Copyright 2024 AxonOps Limited

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

      https://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

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
  labels:
    app: es-{{ .Name }}
    component: elasticsearch
  {{- with .Labels }}
    {{- range $key, $value := . }}
    {{ $key }}: {{ $value }}
    {{- end }}
  {{- end }}
  {{- with .Annotations }}
  annotations:
    {{- range $key, $value := . }}
    {{ $key }}: {{ $value }}
    {{- end }}
  {{- end }}
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
  labels:
    app: es-{{ .Name }}
    component: elasticsearch
  {{- with .Labels }}
    {{- range $key, $value := . }}
    {{ $key }}: {{ $value }}
    {{- end }}
  {{- end }}
  {{- with .Annotations }}
  annotations:
    {{- range $key, $value := . }}
    {{ $key }}: {{ $value }}
    {{- end }}
  {{- end }}
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
        {{- range $env := .Env }}
        - name: {{ $env.Name }}
          value: "{{ $env.Value }}"
        {{- end }}
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
	Name        string
	Namespace   string
	Labels      map[string]string
	Annotations map[string]string
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
	StorageSize        string
	StorageClass       string
	Persistent         bool
	Labels             map[string]string
	Annotations        map[string]string
	Env                []cassandraaxonopscomv1beta1.EnvVars
	CpuLimit           string
	MemoryLimit        string
	CpuRequest         string
	MemoryRequest      string
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
		StorageSize:   utils.ValueOrDefault(cfg.Spec.AxonOps.Elasticsearch.PersistentVolume.Size, ""),
		StorageClass:  utils.ValueOrDefault(cfg.Spec.AxonOps.Elasticsearch.PersistentVolume.StorageClass, ""),
		Labels:        cfg.Spec.AxonOps.Server.Labels,
		Annotations:   cfg.Spec.AxonOps.Server.Annotations,
		Env:           cfg.Spec.AxonOps.Elasticsearch.Env,
		CpuRequest:    utils.ValueOrDefault(cfg.Spec.AxonOps.Elasticsearch.Resources.Requests.Cpu().String(), "500m"),
		MemoryRequest: utils.ValueOrDefault(cfg.Spec.AxonOps.Elasticsearch.Resources.Requests.Memory().String(), "1Gi"),
		CpuLimit:      utils.ValueOrDefault(cfg.Spec.AxonOps.Elasticsearch.Resources.Limits.Cpu().String(), "1000m"),
		MemoryLimit:   utils.ValueOrDefault(cfg.Spec.AxonOps.Elasticsearch.Resources.Limits.Memory().String(), "2Gi"),
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
		Name:        cfg.GetName(),
		Namespace:   cfg.GetNamespace(),
		Labels:      cfg.Spec.AxonOps.Server.Labels,
		Annotations: cfg.Spec.AxonOps.Server.Annotations,
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
