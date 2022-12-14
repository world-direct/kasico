/*
Copyright 2022.

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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// RouterInstanceSpec defines the desired state of RouterInstance
type RouterInstanceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// IngressClassName is the name of the ingressClass managed by this RouterInstance.
	IngressClassName string `json:"ingressClassName,omitempty"`

	// TemplateConfigMapName is the name of the configMap for the kamailio config files.
	TemplateConfigMapName string `json:"templateConfigMapName,omitempty"`

	// RouterService defines configuration values for the generated service
	RouterService RouterServiceSpec `json:"routerService,omitempty"`
}

// RouterServiceSpec defines configuration values for the generated service
type RouterServiceSpec struct {

	// Annotations allows the user to add annoations to the router service
	Annotations map[string]string `json:"annotations,omitempty"`

	//+kubebuilder:default=5060
	UDPPort uint16 `json:"udpPort,omitempty"`

	//+kubebuilder:default=0
	TCPPort uint16 `json:"tcpPort,omitempty"`

	AdvertiseAddress string `json:"advertiseAddress,omitempty"`
}

// RouterInstanceStatus defines the observed state of RouterInstance
type RouterInstanceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Conditions represent the latest available observations of an object's state
	Conditions []metav1.Condition `json:"conditions"`

	// ConfigurationGeneration is incremented if the router pods need to be restarted
	ConfigurationGeneration int `json:"configurationVersion,omitempty"`

	// TemplatesHash is used for change-tracking
	TemplatesHash string `json:"templatesHash,omitempty"`

	// RouterDataHash is used for change-tracking
	RouterDataHash string `json:"routerDataHash,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// RouterInstance is the Schema for the routerinstances API
type RouterInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RouterInstanceSpec   `json:"spec,omitempty"`
	Status RouterInstanceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// RouterInstanceList contains a list of RouterInstance
type RouterInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RouterInstance `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RouterInstance{}, &RouterInstanceList{})
}
