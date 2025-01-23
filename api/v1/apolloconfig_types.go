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

// Config defines the apollo config detail.
type Config struct {
	// +kubebuilder:validation:MinLength:=1
	// AppId is apollo appId
	AppId string `json:"appId,omitempty"`

	// ClusterName is the name of cluster
	ClusterName string `json:"clusterName,omitempty"`

	// NamespaceName is the name of namespace
	NamespaceName string `json:"namespaceName,omitempty"`

	// AccessKeySecret is the secret key of the Apollo Config Server.
	AccessKeySecret string `json:"accessKeySecret,omitempty"`
}

// ApolloConfigSpec defines the desired state of ApolloConfig.
type ApolloConfigSpec struct {
	// +kubebuilder:validation:MinLength:=1
	// ConfigServer is the name of custom resource ApolloConfigServer
	ApolloConfigServer string `json:"apolloConfigServer,omitempty"`

	// +kubebuilder:validation:MinLength:=1
	// ConfigMap is the name of generated ConfigMap
	ConfigMap string `json:"configMap,omitempty"`

	// FileName is the name of config file in ConfigMap. If not specified, apollo namespace name will be used
	FileName string `json:"fileName,omitempty"`

	// +kubebuilder:validation:
	// Configs is the list of apollo configs
	Apollo Config `json:"apollo"`
}

// ApolloConfigStatus defines the observed state of ApolloConfig.
type ApolloConfigStatus struct {
	LastSyncedSuccess metav1.Time `json:"lastSynced,omitempty"`
	UpdateAt          metav1.Time `json:"updateAt,omitempty"`
	SyncStatus        string      `json:"syncStatus,omitempty"`
	Message           string      `json:"message,omitempty"`
	ReleaseKey        string      `json:"releaseKey,omitempty"`
	NotificationId    int         `json:"notificationId,omitempty"`
}

// +kubebuilder:validation:Enum:=Success;Failed;

// SyncStatus defines the status of sync.
type SyncStatus string

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ApolloConfig is the Schema for the apolloconfigs API.
type ApolloConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ApolloConfigSpec   `json:"spec,omitempty"`
	Status ApolloConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ApolloConfigList contains a list of ApolloConfig.
type ApolloConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ApolloConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ApolloConfig{}, &ApolloConfigList{})
}
