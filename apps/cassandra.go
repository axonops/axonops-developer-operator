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

const defaultCassandraImage = "registry.axonops.com/axonops-public/axonops-docker/cassandra"
const defaultCassandraTag = "4.1"

const cassandraHeadlessServiceTemplate = `
apiVersion: v1
kind: Service
metadata:
  name: ca-{{ .Name }}-headless
  namespace: {{ .Namespace }}
  labels:
    app: ds-{{ .Name }}
    component: cassandra
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
  publishNotReadyAddresses: true
  clusterIP: None
  selector:
    app: ca-{{ .Name }}
  ports:
    - name: intra
      port: 7000
      targetPort: intra
    - name: tls
      port: 7001
      targetPort: tls
    - name: jmx
      port: 7199
      targetPort: jmx
    - name: cql
      port: 9042
      targetPort: cql
`
const cassandraServiceTemplate = `
apiVersion: v1
kind: Service
metadata:
  name: ca-{{ .Name }}
  namespace: {{ .Namespace }}
  labels:
    app: ds-{{ .Name }}
    component: cassandra
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
    app: ca-{{ .Name }}
  ports:
    - name: intra
      port: 7000
      targetPort: intra
    - name: tls
      port: 7001
      targetPort: tls
    - name: jmx
      port: 7199
      targetPort: jmx
    - name: cql
      port: 9042
      targetPort: cql
`

const cassandraTemplate = `
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: ca-{{ .Name }}
  namespace: {{ .Namespace }}
  labels:
    app: ds-{{ .Name }}
    component: cassandra
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
  serviceName: ca-{{ .Name }}
  replicas: {{ .Replicas }}
  selector:
    matchLabels:
      app: ca-{{ .Name }}
  template:
    metadata:
      labels:
        app: ca-{{ .Name }}
    spec:
      containers:
      - name: cassandra
        image: {{ .Image }}
        ports:
        - containerPort: 9042
          name: cql
        - containerPort: 7199
          name: jmx
        - containerPort: 7000
          name: intra
        - containerPort: 7001
          name: tls
        env:
        - name: CASSANDRA_CLUSTER_NAME
          value: {{ .ClusterName }}
        - name: CASSANDRA_SEEDS
          value: ca-{{ .Name }}-0.ca-{{ .Name }}.{{ .Namespace }}.svc.cluster.local
        - name: CASSANDRA_ENDPOINT_SNITCH
          value: GossipingPropertyFileSnitch
        - name: CASSANDRA_DC
          value: dc1
        - name: CASSANDRA_RACK
          value: rack1
        - name: CASSANDRA_BROADCAST_RPC_ADDRESS
          value: 127.0.0.1
        - name: CASSANDRA_NATIVE_TRANSPORT_PORT
          value: "9042"
        - name: MAX_HEAP_SIZE
          value: {{ .HeapSize }}
        - name: HEAP_NEWSIZE
          value: 50m
        - name: AXON_AGENT_SERVER_HOST
          value: as-{{ .Name }}
        - name: AXON_AGENT_SERVER_PORT
          value: "1888"
        - name: AXON_AGENT_ORG
          value: developer
        - name: AXON_AGENT_TLS_MODE
          value: none
        - name: AXON_AGENT_LOG_OUTPUT
          value: file
        - name: node.name
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: ES_JAVA_OPTS
          value: "{{ .JavaOpts }}"
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
        livenessProbe:
          exec:
            command:
              - /bin/bash
              - -ec
              - |
                nodetool info | grep "Native Transport active: true"
          initialDelaySeconds: 60
          periodSeconds: 30
          timeoutSeconds: 30
          successThreshold: 1
          failureThreshold: 5
        readinessProbe:
          exec:
            command:
              - /bin/bash
              - -ec
              - |
                nodetool status | grep -E "^UN\\s+${POD_IP}"
          initialDelaySeconds: 60
          periodSeconds: 30
          timeoutSeconds: 30
          successThreshold: 1
          failureThreshold: 5
        startupProbe:
          exec:
            command:
              - /bin/bash
              - -ec
              - |
                nodetool status | grep -E "^UN\\s+${POD_IP}"
          initialDelaySeconds: 60
          periodSeconds: 30
          timeoutSeconds: 30
          successThreshold: 1
          failureThreshold: 5
        lifecycle:
          preStop:
            exec:
              command:
                - bash
                - -ec
                {{- if ne .StorageSize "" }}
                - nodetool drain
                {{- else }}
                - nodetool decommission
                {{- end }}
{{- if ne .StorageSize "" }}
        volumeMounts:
        - name: data
          mountPath: /var/lib/cassandra
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

type CassandraServiceConfig struct {
	Name        string
	Namespace   string
	Labels      map[string]string
	Annotations map[string]string
}

type CassandraConfig struct {
	Name          string
	Namespace     string
	Replicas      int
	Image         string
	ClusterName   string
	JavaOpts      string
	CpuLimit      string
	MemoryLimit   string
	CpuRequest    string
	MemoryRequest string
	HeapSize      string
	StorageSize   string
	StorageClass  string
	Labels        map[string]string
	Annotations   map[string]string
	Env           []cassandraaxonopscomv1beta1.EnvVars
}

func GenerateCassandraConfig(cfg cassandraaxonopscomv1beta1.AxonOpsCassandra) (*appsv1.StatefulSet, error) {
	config := CassandraConfig{
		Name:      cfg.GetName(),
		Namespace: cfg.GetNamespace(),
		Replicas:  utils.ValueOrDefaultInt(cfg.Spec.Cassandra.Replicas, 1),
		Image: fmt.Sprintf("%s:%s",
			utils.ValueOrDefault(cfg.Spec.Cassandra.Image.Repository, defaultCassandraImage),
			utils.ValueOrDefault(cfg.Spec.Cassandra.Image.Tag, defaultCassandraTag),
		),
		ClusterName:   utils.ValueOrDefault(cfg.Spec.Cassandra.ClusterName, cfg.GetName()),
		JavaOpts:      utils.ValueOrDefault(cfg.Spec.Cassandra.JavaOpts, "-Xms512m -Xmx512m"),
		CpuLimit:      "1000m",
		MemoryLimit:   "2Gi",
		CpuRequest:    "100m",
		MemoryRequest: "1Gi",
		StorageSize:   utils.ValueOrDefault(cfg.Spec.AxonOps.Elasticsearch.PersistentVolume.Size, ""),
		StorageClass:  utils.ValueOrDefault(cfg.Spec.AxonOps.Elasticsearch.PersistentVolume.StorageClass, ""),
		HeapSize:      utils.ValueOrDefault(cfg.Spec.Cassandra.HeapSize, "512M"),
		Labels:        cfg.Spec.Cassandra.Labels,
		Annotations:   cfg.Spec.Cassandra.Annotations,
		Env:           cfg.Spec.Cassandra.Env,
	}

	statefulSet := &appsv1.StatefulSet{}
	b := bytes.NewBuffer(nil)
	tmpl, err := template.New("cassandra").Funcs(sprig.FuncMap()).Parse(cassandraTemplate)
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

func GenerateCassandraServiceConfig(cfg cassandraaxonopscomv1beta1.AxonOpsCassandra) (*corev1.Service, error) {
	config := CassandraServiceConfig{
		Name:        cfg.GetName(),
		Namespace:   cfg.GetNamespace(),
		Labels:      cfg.Spec.Cassandra.Labels,
		Annotations: cfg.Spec.Cassandra.Annotations,
	}

	svc := &corev1.Service{}
	b := bytes.NewBuffer(nil)
	tmpl, err := template.New("cassandra").Funcs(sprig.FuncMap()).Parse(cassandraServiceTemplate)
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

/* not used right now */
func GenerateCassandraHeadlessServiceConfig(cfg cassandraaxonopscomv1beta1.AxonOpsCassandra) (*corev1.Service, error) {
	config := CassandraServiceConfig{
		Name:        cfg.GetName(),
		Namespace:   cfg.GetNamespace(),
		Labels:      cfg.Spec.Cassandra.Labels,
		Annotations: cfg.Spec.Cassandra.Annotations,
	}

	svc := &corev1.Service{}
	b := bytes.NewBuffer(nil)
	tmpl, err := template.New("cassandra").Funcs(sprig.FuncMap()).Parse(cassandraHeadlessServiceTemplate)
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
