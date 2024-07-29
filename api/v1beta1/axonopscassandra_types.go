/*
Copyright AxonOps Limited 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta1

import (
	networking "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Ingress defines an ingress configuration for the AxonOps Workbench
type Ingress struct {
	Enabled          bool                    `json:"enabled,omitempty"`
	ApiVersion       string                  `json:"apiVersion,omitempty"`
	Annotations      map[string]string       `json:"annotations,omitempty"`
	Labels           map[string]string       `json:"labels,omitempty"`
	IngressClassName string                  `json:"ingressClassName,omitempty"`
	Hosts            []string                `json:"hosts,omitempty"`
	TLS              []networking.IngressTLS `json:"tls,omitempty"`
	Path             string                  `json:"path,omitempty"`
	PathType         networking.PathType     `json:"pathType,omitempty"`
	ServiceName      string                  `json:"serviceName,omitempty"`
}

// PersistentVolumeSpec defines the persistent volume specification
type PersistentVolumeSpec struct {
	StorageClass string `json:"storageClass,omitempty"`
	Size         string `json:"size,omitempty"`
}

type ContainerImage struct {
	Repository string `json:"repository,omitempty"`
	Tag        string `json:"tag,omitempty"`
}

// AxonOpsCassandraCluster defines the Apache Cassandra cluster to install
type AxonOpsCassandraCluster struct {
	Image            ContainerImage       `json:"image,omitempty"`
	Replicas         int                  `json:"replicas,omitempty"`
	ClusterName      string               `json:"clusterName,omitempty"`
	PersistentVolume PersistentVolumeSpec `json:"persistentVolume,omitempty"`
	JavaOpts         string               `json:"javaOpts,omitempty"`
	HeapSize         string               `json:"heapSize,omitempty"`
}

// AxonOpsDashboard defines the dashboard
type AxonOpsDashboard struct {
	// Change the default repository and tag
	Image ContainerImage `json:"image,omitempty"`
	// Increase the number of replicas if desired from the default, 1
	Replicas int     `json:"replicas,omitempty"`
	Ingress  Ingress `json:"ingress,omitempty"`
}

// AxonOpsServer defines the dashboard
type AxonOpsServer struct {
	// Container image definition with repository and tag
	Image ContainerImage `json:"image,omitempty"`
}

// AxonOpsServer defines the dashboard
type Elasticsearch struct {
	// Container image definition with repository and tag
	Image            ContainerImage       `json:"image,omitempty"`
	PersistentVolume PersistentVolumeSpec `json:"persistentVolume,omitempty"`
	JavaOpts         string               `json:"javaOpts,omitempty"`
	ClusterName      string               `json:"clusterName,omitempty"`
}

// AxonOpsCassandraCluster defines the Apache Cassandra cluster to install
type AxonOpsCluster struct {
	Dashboard     AxonOpsDashboard `json:"dashboard,omitempty"`
	Server        AxonOpsServer    `json:"server,omitempty"`
	Elasticsearch Elasticsearch    `json:"elasticsearch,omitempty"`
}

// AxonOpsCassandraSpec defines the desired state of AxonOpsCassandra
type AxonOpsCassandraSpec struct {
	// Defines the Development cluster composition. The default is to build
	// an Apache Cassandra cluster with not persistent storage and
	// connected to a locally running AxonOps which requires
	// the AxonOps server, the AxonOps dashboard and Elasticsearch as metrics storage
	Cassandra AxonOpsCassandraCluster `json:"cassandra,omitempty"`
	AxonOps   AxonOpsCluster          `json:"axonops,omitempty"`
}

// AxonOpsCassandraStatus defines the observed state of AxonOpsCassandra
type AxonOpsCassandraStatus struct {
	// Status not defined yet
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// AxonOpsCassandra is the Schema for the axonopscassandras API
type AxonOpsCassandra struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AxonOpsCassandraSpec   `json:"spec,omitempty"`
	Status AxonOpsCassandraStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// AxonOpsCassandraList contains a list of AxonOpsCassandra
type AxonOpsCassandraList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AxonOpsCassandra `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AxonOpsCassandra{}, &AxonOpsCassandraList{})
}
