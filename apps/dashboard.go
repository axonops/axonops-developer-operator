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
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
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
  labels:
    app: ds-{{ .Name }}
    component: dashboard
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
  labels:
    app: ds-{{ .Name }}
    component: dashboard
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

const DashboardIngressTemplate = `{{- if .IngressEnabled -}}
apiVersion: {{ default "networking.k8s.io/v1" .APIVersion }}
kind: Ingress
metadata:
  name: ds-{{ .Name }}
  namespace: {{ .Namespace }}
  labels:
    app: ds-{{ .Name }}
    component: dashboard
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
  {{- if .ClassName }}
  ingressClassName: {{ default "" .ClassName }}
  {{- end }}
  {{- if .Tls }}
  tls:
    - hosts:
        {{- range $host := .Hosts }}
        - {{ $host }}
        {{- end }}
      secretName: {{ .Name }}-tls
  {{- end }}
  rules:
    {{- range $host := .Hosts }}
    - host: {{ $host | quote }}
      http:
        paths:
          - pathType: {{ default $.PathType "Prefix" }}
            path: {{ default $.Path "/" }}
            backend:
              service:
                name: "ds-{{ $.Name }}"
                port:
                  number: 3000
    {{- end }}
{{- end }}`

type DashboardServiceConfig struct {
	Name        string
	Namespace   string
	Labels      map[string]string
	Annotations map[string]string
}

type DashboardConfig struct {
	Name        string
	Namespace   string
	Replicas    int
	Image       string
	Labels      map[string]string
	Annotations map[string]string
	Env         []cassandraaxonopscomv1beta1.EnvVars
	Resources   corev1.ResourceRequirements `json:"resources,omitempty"`
}

type DashboardIngressConfig struct {
	Name           string
	Namespace      string
	IngressEnabled bool
	APIVersion     string
	Labels         map[string]string
	Annotations    map[string]string
	ClassName      string
	Tls            bool
	Hosts          []string
	Path           string
	PathType       string
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
		Labels:      cfg.Spec.AxonOps.Dashboard.Labels,
		Annotations: cfg.Spec.AxonOps.Dashboard.Annotations,
		Env:         cfg.Spec.AxonOps.Dashboard.Env,
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(utils.ValueOrDefault(cfg.Spec.AxonOps.Dashboard.Resources.Requests.Cpu().String(), "500m")),
				corev1.ResourceMemory: resource.MustParse(utils.ValueOrDefault(cfg.Spec.AxonOps.Dashboard.Resources.Requests.Memory().String(), "256Mi")),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(utils.ValueOrDefault(cfg.Spec.AxonOps.Dashboard.Resources.Limits.Cpu().String(), "1000m")),
				corev1.ResourceMemory: resource.MustParse(utils.ValueOrDefault(cfg.Spec.AxonOps.Dashboard.Resources.Limits.Memory().String(), "512Mi")),
			},
		},
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
		Name:        cfg.GetName(),
		Namespace:   cfg.GetNamespace(),
		Labels:      cfg.Spec.AxonOps.Dashboard.Labels,
		Annotations: cfg.Spec.AxonOps.Dashboard.Annotations,
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

func GenerateDashboardIngressConfig(cfg cassandraaxonopscomv1beta1.AxonOpsCassandra) (*networkingv1.Ingress, error) {
	config := DashboardIngressConfig{
		Name:           cfg.GetName(),
		Namespace:      cfg.GetNamespace(),
		IngressEnabled: utils.ValueOrDefaultBool(cfg.Spec.AxonOps.Dashboard.Ingress.Enabled, false),
		APIVersion:     utils.ValueOrDefault(cfg.Spec.AxonOps.Dashboard.Ingress.ApiVersion, "networking.k8s.io/v1"),
		Labels:         cfg.Spec.AxonOps.Dashboard.Ingress.Labels,
		Annotations:    cfg.Spec.AxonOps.Dashboard.Ingress.Annotations,
		ClassName:      utils.ValueOrDefault(cfg.Spec.AxonOps.Dashboard.Ingress.IngressClassName, ""),
		Tls:            true,
		Hosts:          cfg.Spec.AxonOps.Dashboard.Ingress.Hosts,
		Path:           utils.ValueOrDefault(cfg.Spec.AxonOps.Dashboard.Ingress.Path, "/"),
		PathType:       "Exact",
	}

	ingress := &networkingv1.Ingress{}
	b := bytes.NewBuffer(nil)
	tmpl, err := template.New("DashboardIngress").Funcs(sprig.FuncMap()).Parse(DashboardIngressTemplate)
	if err != nil {
		return ingress, err
	}

	err = tmpl.Execute(b, config)
	if err != nil {
		return ingress, err
	}

	obj := &unstructured.Unstructured{}
	es := yaml.NewYAMLOrJSONDecoder(b, 500)
	if err := es.Decode(obj); err != nil {
		return ingress, err
	}

	err = runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, ingress)
	if err != nil {
		return ingress, err
	}
	return ingress, nil
}
