/*

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

package v1alpha1

import (
	"reflect"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	runtimev1alpha1 "github.com/crossplaneio/crossplane-runtime/apis/core/v1alpha1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ProvisionPolicy indicates whether Wordpress should be deployed to an existing
// Kubernetes cluster or to provision a new cluster.
type ProvisionPolicy string

const (
	// ProvisionNewCluster means a new KubernetesCluster claim will be created.
	ProvisionNewCluster ProvisionPolicy = "ProvisionNewCluster"

	// UseExistingTarget means an existing KubernetesTarget in the same
	// namespace will be used for scheduling.
	UseExistingTarget ProvisionPolicy = "UseExistingTarget"
)

// WordpressInstanceSpec defines the desired state of WordpressInstance
type WordpressInstanceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// NOTE(muvaf): Image is only picked up during creation and not propagated
	// after that point.

	// Image will be used as image of the container that Wordpress runs in.
	// If not specified, the default will be used.
	// +optional
	Image string `json:"image,omitempty"`

	// MySQLInstanceRef refers to the MySQLInstance object managed by this
	// WordpressInstance
	// +optional
	MySQLInstanceRef *v1.ObjectReference `json:"mySQLInstanceRef,omitempty"`

	// ProvisionPolicy indicates whether Wordpress should be deployed to an
	// existing Kubernetes cluster or to provision a new cluster.
	// +optional
	// +kubebuilder:validation:Enum=ProvisionNewCluster;UseExistingTarget
	ProvisionPolicy *ProvisionPolicy `json:"provisionPolicy,omitempty"`

	// KubernetesApplicationRef refers to the KubernetesApplication object managed by this
	// WordpressInstance
	// +optional
	KubernetesApplicationRef *v1.ObjectReference `json:"kubernetesApplicationRef,omitempty"`
}

// WordpressInstanceStatus defines the observed state of WordpressInstance
type WordpressInstanceStatus struct {
	runtimev1alpha1.ConditionedStatus `json:",inline"`
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Endpoint is the URL of Wordpress that you can use to access.
	// It will be populated as soon as a LoadBalancer is assigned to Wordpress
	// service.
	Endpoint string `json:"endpoint,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// WordpressInstance is the Schema for the wordpressinstances API
type WordpressInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WordpressInstanceSpec   `json:"spec,omitempty"`
	Status WordpressInstanceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// WordpressInstanceList contains a list of WordpressInstance
type WordpressInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WordpressInstance `json:"items"`
}

// WordpressInstance type metadata.
var (
	WordpressInstanceKind             = reflect.TypeOf(WordpressInstance{}).Name()
	WordpressInstanceKindAPIVersion   = WordpressInstanceKind + "." + GroupVersion.String()
	WordpressInstanceGroupVersionKind = GroupVersion.WithKind(WordpressInstanceKind)
)

func init() {
	SchemeBuilder.Register(&WordpressInstance{}, &WordpressInstanceList{})
}
