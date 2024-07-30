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

const defaultServerImage = "registry.axonops.com/axonops-public/axonops-docker/axon-server"
const defaultServerTag = "latest"

const ServerServiceTemplate = `
apiVersion: v1
kind: Service
metadata:
  name: as-{{ .Name }}
  namespace: {{ .Namespace }}
  labels:
    app: as-{{ .Name }}
    component: axon-server
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
  labels:
    app: as-{{ .Name }}
    component: axon-server
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
`

type ServerServiceConfig struct {
	Name        string
	Namespace   string
	Labels      map[string]string
	Annotations map[string]string
}

type ServerConfig struct {
	Name          string
	Namespace     string
	Replicas      int
	Image         string
	Labels        map[string]string
	Annotations   map[string]string
	Env           []cassandraaxonopscomv1beta1.EnvVars
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
		Labels:        cfg.Spec.AxonOps.Server.Labels,
		Annotations:   cfg.Spec.AxonOps.Server.Annotations,
		Env:           cfg.Spec.AxonOps.Server.Env,
		CpuRequest:    utils.ValueOrDefault(cfg.Spec.AxonOps.Server.Resources.Requests.Cpu().String(), "250m"),
		MemoryRequest: utils.ValueOrDefault(cfg.Spec.AxonOps.Server.Resources.Requests.Memory().String(), "256Mi"),
		CpuLimit:      utils.ValueOrDefault(cfg.Spec.AxonOps.Server.Resources.Limits.Cpu().String(), "1000m"),
		MemoryLimit:   utils.ValueOrDefault(cfg.Spec.AxonOps.Server.Resources.Limits.Memory().String(), "512Mi"),
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
		Name:        cfg.GetName(),
		Namespace:   cfg.GetNamespace(),
		Labels:      cfg.Spec.AxonOps.Server.Labels,
		Annotations: cfg.Spec.AxonOps.Server.Annotations,
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
