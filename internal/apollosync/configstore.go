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

package apollosync

import (
	v1 "adamswanglin.github.com/apollo-configmap/api/v1"
	"adamswanglin.github.com/apollo-configmap/internal"
	"context"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sync"
)

const SYNC_SYNCING = "Syncing"

// ConfigStore a bridge between remote config and k8s ApolloConfig CR
type ConfigStore struct {
	// key is apollo, value is local config release key
	localConfigReleaseMap map[string]string
	// key is apollo, value is remote config release key
	remoteConfigReleaseMap map[string]*ApolloClient
	rwMutex                sync.RWMutex
	k8sClient              client.Client
}

func NewConfigStore(k8sClient client.Client) *ConfigStore {
	return &ConfigStore{
		localConfigReleaseMap:  make(map[string]string),
		remoteConfigReleaseMap: make(map[string]*ApolloClient),
		rwMutex:                sync.RWMutex{},
		k8sClient:              k8sClient,
	}
}

// DeleteApolloConfig delete apollo config
func (configStore *ConfigStore) DeleteApolloConfig(apolloConfig v1.ApolloConfig) {
	configStore.rwMutex.Lock()
	defer configStore.rwMutex.Unlock()
	key := convertToKey(&apolloConfig)
	if apolloClient, ok := configStore.remoteConfigReleaseMap[key]; ok {
		apolloClient.stopPolling()
		delete(configStore.remoteConfigReleaseMap, key)
	}
	if _, ok := configStore.localConfigReleaseMap[key]; ok {
		delete(configStore.localConfigReleaseMap, key)
	}
}

// CreateOrUpdateApolloConfig create or update apollo config
func (configStore *ConfigStore) CreateOrUpdateApolloConfig(apolloConfig v1.ApolloConfig, apolloConfigServer *v1.ApolloConfigServer) {
	configStore.rwMutex.Lock()
	defer configStore.rwMutex.Unlock()
	key := convertToKey(&apolloConfig)
	newApolloConfig := NewApolloClient(&apolloConfig, apolloConfigServer, configStore)
	if apolloClient, ok := configStore.remoteConfigReleaseMap[key]; ok {
		if apolloClient.serverAddress != newApolloConfig.serverAddress ||
			apolloClient.appId != newApolloConfig.appId ||
			apolloClient.clusterName != newApolloConfig.clusterName ||
			apolloClient.namespaceName != newApolloConfig.namespaceName ||
			apolloClient.accessKey != newApolloConfig.accessKey {
			//changed
			apolloClient.rwMutex.Lock()
			defer apolloClient.rwMutex.Unlock()

			apolloClient.RemoteResult.NotificationId = initialNotificationId
			apolloClient.serverAddress = newApolloConfig.serverAddress
			apolloClient.appId = newApolloConfig.appId
			apolloClient.clusterName = newApolloConfig.clusterName
			apolloClient.namespaceName = newApolloConfig.namespaceName
			apolloClient.accessKey = newApolloConfig.accessKey

		}
	} else {
		configStore.remoteConfigReleaseMap[key] = newApolloConfig
		newApolloConfig.startPolling()
	}
}

// NotifyApolloConfigChange notify apollo config change by update ApolloConfig custom resource status
func (configStore *ConfigStore) NotifyApolloConfigChange(namespacedName string, notificationId int) error {
	apolloConfig := new(v1.ApolloConfig)
	obj, err := internal.KeyToNamespacedName(namespacedName)
	if err != nil {
		return err
	}
	err = configStore.k8sClient.Get(context.Background(), *obj, apolloConfig)
	if err != nil {
		return errors.Wrap(err, "Unable to get apollo config:"+namespacedName)
	} else if apolloConfig != nil && len(apolloConfig.Name) > 0 {
		if apolloConfig.Status.NotificationId == notificationId {
			// ignore same notification
			return nil
		} else {
			apolloConfig.Status.NotificationId = notificationId
			apolloConfig.Status.UpdateAt = metav1.Now()
			apolloConfig.Status.SyncStatus = SYNC_SYNCING
			err := configStore.k8sClient.Status().Update(context.Background(), apolloConfig)
			if err != nil {
				return err
			}
		}
	} else {
		return errors.New("Unable to get apollo config:" + namespacedName)
	}
	return nil
}

// GetRemoteConfig get remote config
func (configStore *ConfigStore) GetRemoteConfig(apolloConfig *v1.ApolloConfig) (releaseKey string, fileContent string, err error) {
	configStore.rwMutex.RLock()
	defer configStore.rwMutex.RUnlock()
	key := convertToKey(apolloConfig)
	if apolloClient, ok := configStore.remoteConfigReleaseMap[key]; ok {
		return apolloClient.getRemoteConfig(context.Background())
	} else {
		return "", "", errors.New("Apollo client not initialized: " + key)
	}
}

// convertToKey convert apollo config to key
func convertToKey(apolloConfig *v1.ApolloConfig) string {
	return types.NamespacedName{
		Namespace: apolloConfig.Namespace,
		Name:      apolloConfig.Name,
	}.String()
}
