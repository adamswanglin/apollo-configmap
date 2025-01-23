/*
 Copyright 2025 adamswanglin

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

// ApolloConfigServerSpec defines the desired state of ApolloConfigServer.
type ApolloConfigServerSpec struct {

	// ConfigServerURL is the URL of the Apollo Config Server.e.g. http://localhost:8080
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern="^https?://"
	ConfigServerURL string `json:"configServerURL,omitempty"`
}

// ApolloConfigServerStatus defines the observed state of ApolloConfigServer.
type ApolloConfigServerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ApolloConfigServer is the Schema for the apolloconfigservers API.
type ApolloConfigServer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ApolloConfigServerSpec   `json:"spec,omitempty"`
	Status ApolloConfigServerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ApolloConfigServerList contains a list of ApolloConfigServer.
type ApolloConfigServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ApolloConfigServer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ApolloConfigServer{}, &ApolloConfigServerList{})
}
