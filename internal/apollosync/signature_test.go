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
	"context"
	"github.com/go-logr/stdr"
	"log"
	"os"
	"testing"
)

// TestUpdateRemoteConfig tests the getRemoteConfig method of ApolloClient.
func TestUpdateRemoteConfig(t *testing.T) {
	stdLogger := log.New(os.Stdout, "", log.LstdFlags) // 标准库日志实例
	logger := stdr.New(stdLogger)                      // 包装成 logr 接口
	apolloClient := ApolloClient{
		serverAddress: "http://81.68.181.139:8080/",
		accessKey:     "52465a80e7f54b9f9dd2d78eca148c60",
		appId:         "11111111",
		clusterName:   "default",
		namespaceName: "test.yaml",
		key:           "",
		retryInterval: 0,
		RemoteResult: &RemoteConfig{
			ReleaseKey:     "",
			NotificationId: -1,
			Config:         "",
		},
		//set log to fmt
		log: logger,
	}
	_, str, err := apolloClient.getRemoteConfig(context.Background())
	if err == nil {
		t.Log("config content: \n" + str)
		return
	}
	t.Error(err)
}
