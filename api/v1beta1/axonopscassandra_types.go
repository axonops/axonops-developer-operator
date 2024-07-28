/*
Copyright 2024.

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
}

// AxonOpsDashboard defines the dashboard
type AxonOpsDashboard struct {
	Image    ContainerImage         `json:"image,omitempty"`
	Replicas int                    `json:"replicas,omitempty"`
	Ingress  networking.IngressSpec `json:"ingress,omitempty"`
}

// AxonOpsServer defines the dashboard
type AxonOpsServer struct {
	Image ContainerImage `json:"image,omitempty"`
}

// AxonOpsServer defines the dashboard
type Elasticsearch struct {
	Image            ContainerImage       `json:"image,omitempty"`
	PersistentVolume PersistentVolumeSpec `json:"persistentVolume,omitempty"`
}

// AxonOpsCassandraCluster defines the Apache Cassandra cluster to install
type AxonOpsCluster struct {
	Dashboard     AxonOpsDashboard `json:"dashboard,omitempty"`
	Server        AxonOpsServer    `json:"server,omitempty"`
	Elasticsearch Elasticsearch    `json:"elasticsearch,omitempty"`
}

// AxonOpsCassandraSpec defines the desired state of AxonOpsCassandra
type AxonOpsCassandraSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Cassandra AxonOpsCassandraCluster `json:"cassandra,omitempty"`
	AxonOps   AxonOpsCluster          `json:"axonops,omitempty"`
	//Image ContainerImage `json:"name,omitempty"`
}

// AxonOpsCassandraStatus defines the observed state of AxonOpsCassandra
type AxonOpsCassandraStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
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
