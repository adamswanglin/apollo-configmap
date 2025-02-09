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
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	v1 "adamswanglin.github.com/apollo-configmap/api/v1"
	"github.com/go-logr/logr"
	"github.com/iancoleman/orderedmap"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/apimachinery/pkg/util/json"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	initialRetryInterval  = 1 * time.Second
	maxRetryInterval      = 60 * time.Second // Maximum retry interval
	initialNotificationId = -1
)

var (
	httpClient = &http.Client{
		Timeout: 70 * time.Second, // Set timeout for the request
		Transport: &MetricsRoundTripper{
			transport: http.DefaultTransport,
		},
	}
	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "url", "status"},
	)
	httpRequestCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "url", "status"},
	)
)

func init() {
	metrics.Registry.MustRegister(httpRequestDuration, httpRequestCount)
}

// MetricsRoundTripper is a custom RoundTripper that captures HTTP metrics.
type MetricsRoundTripper struct {
	transport http.RoundTripper
}

// RoundTrip captures metrics for each HTTP request.
func (m *MetricsRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()
	resp, err := m.transport.RoundTrip(req)
	duration := time.Since(start).Seconds()

	status := "error"
	if resp != nil {
		status = http.StatusText(resp.StatusCode)
	}

	// replace path configs/{appId}/{clusterName}/{namespaceName} notifications/v2
	path := req.URL.Path
	if strings.Contains(path, "notifications/v2") {
		path = "notifications/v2"
	} else {
		path = "configs/{appId}/{clusterName}/{namespaceName}"
	}

	// Update metrics
	httpRequestDuration.WithLabelValues(req.Method, path, status).Observe(duration)
	httpRequestCount.WithLabelValues(req.Method, path, status).Inc()

	return resp, err
}

type RemoteConfig struct {
	ReleaseKey     string
	NotificationId int
	Config         string
	SyncStatus     string
}

type ApolloClient struct {
	configStore   *ConfigStore
	RemoteResult  *RemoteConfig
	log           logr.Logger
	serverAddress string
	accessKey     string
	appId         string
	clusterName   string
	namespaceName string
	key           string
	retryInterval time.Duration
	rwMutex       *sync.RWMutex
	cancelFunc    context.CancelFunc
}

func NewApolloClient(apolloConfig *v1.ApolloConfig, apolloConfigServer *v1.ApolloConfigServer, configStore *ConfigStore) *ApolloClient {

	notificationId := initialNotificationId
	if apolloConfig.Status.NotificationId > 0 {
		notificationId = apolloConfig.Status.NotificationId
	}
	serverAddress := ""
	if apolloConfigServer != nil {
		serverAddress = apolloConfigServer.Spec.ConfigServerURL
	}
	apolloClient := ApolloClient{
		serverAddress: serverAddress,
		accessKey:     apolloConfig.Spec.Apollo.AccessKeySecret,
		appId:         apolloConfig.Spec.Apollo.AppId,
		clusterName:   apolloConfig.Spec.Apollo.ClusterName,
		namespaceName: apolloConfig.Spec.Apollo.NamespaceName,
		configStore:   configStore,
		key:           convertToKey(apolloConfig),
		retryInterval: initialRetryInterval,
		log:           log.FromContext(context.Background()).WithValues("appId", apolloConfig.Spec.Apollo.AppId, "clusterName", apolloConfig.Spec.Apollo.ClusterName, "namespaceName", apolloConfig.Spec.Apollo.NamespaceName, "serverAddress", serverAddress),
		RemoteResult: &RemoteConfig{
			NotificationId: notificationId,
		},
		rwMutex: &sync.RWMutex{},
	}
	return &apolloClient
}

// notifyRequest see: https://www.apolloconfig.com/#/zh/client/other-language-client-user-guide?id=_142-http%e6%8e%a5%e5%8f%a3%e8%af%b4%e6%98%8e
func (client *ApolloClient) notifyRequest(ctx context.Context) (*http.Request, error) {
	// avoid partial param change
	client.rwMutex.RLock()
	defer client.rwMutex.RUnlock()

	notifyStr := "[{\"namespaceName\":\"" + client.namespaceName + "\",\"notificationId\":" + strconv.Itoa(client.RemoteResult.NotificationId) + "}]"
	notifyStr = url.QueryEscape(notifyStr)

	split := ""
	if len(client.serverAddress) > 1 && client.serverAddress[len(client.serverAddress)-1] != '/' {
		split = "/"
	}

	url := client.serverAddress + split + "notifications/v2?"

	params := make([]string, 0)

	if len(client.appId) > 0 {
		params = append(params, "appId="+client.appId)
	}

	if len(client.clusterName) > 0 {
		params = append(params, "cluster="+client.clusterName)
	}

	if len(notifyStr) > 0 {
		params = append(params, "notifications="+notifyStr)
	}

	if len(params) > 0 {
		url += strings.Join(params, "&")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create HTTP request")
	}

	if len(client.accessKey) != 0 {
		headers, err := buildAuthorizationHeaders(url, client.appId, client.accessKey)
		if err != nil {
			return nil, errors.Wrap(err, "unable to sign apollo auth token")
		}
		for key, value := range headers {
			req.Header.Set(key, value)
		}
	}
	req.Header.Set("Accept", "application/json")
	return req, nil
}

// getConfigRequest see: https://www.apolloconfig.com/#/zh/client/other-language-client-user-guide?id=_13-%e9%80%9a%e8%bf%87%e4%b8%8d%e5%b8%a6%e7%bc%93%e5%ad%98%e7%9a%84http%e6%8e%a5%e5%8f%a3%e4%bb%8eapollo%e8%af%bb%e5%8f%96%e9%85%8d%e7%bd%ae
func (client *ApolloClient) getConfigRequest(ctx context.Context) (*http.Request, error) {
	// avoid partial param change
	client.rwMutex.RLock()
	defer client.rwMutex.RUnlock()

	split := ""
	if len(client.serverAddress) > 1 && client.serverAddress[len(client.serverAddress)-1] != '/' {
		split = "/"
	}
	url := client.serverAddress + split + "configs/" + client.appId + "/" + client.clusterName + "/" + client.namespaceName

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create HTTP request")
	}

	if len(client.accessKey) != 0 {
		headers, err := buildAuthorizationHeaders(url, client.appId, client.accessKey)
		if err != nil {
			return nil, errors.Wrap(err, "unable to sign apollo auth token")
		}
		for key, value := range headers {
			req.Header.Set(key, value)
		}
	}
	req.Header.Set("Accept", "application/json")
	return req, nil
}

func (client *ApolloClient) increaseRetryInterval(err error) {
	client.retryInterval = time.Duration(math.Min(float64(client.retryInterval*2), float64(maxRetryInterval)))
	if err != nil {
		client.log.Info("Failed pulling apollo config change, will retry in "+client.retryInterval.String(), "error", err.Error())
	}
}

func (client *ApolloClient) resetRetryInterval() {
	client.retryInterval = 1 * time.Second
}

// startPolling start polling
func (client *ApolloClient) startPolling() {
	ctx, cancel := context.WithCancel(context.Background())
	client.cancelFunc = cancel
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				err := client.doRequestForNotification(ctx)
				if err != nil {
					client.increaseRetryInterval(err)
				} else {
					client.resetRetryInterval()
				}
				time.Sleep(client.retryInterval)
			}
		}
	}()

}

type Notification struct {
	NamespaceName  string `json:"namespaceName"`
	NotificationID int    `json:"notificationId"`
}

// doRequestForNotification do request for config change notification
func (client *ApolloClient) doRequestForNotification(ctx context.Context) error {
	if len(client.serverAddress) == 0 {
		return errors.New("serverAddress is empty")
	}

	// Create the HTTP request
	req, err := client.notifyRequest(ctx)
	if err != nil {
		return err
	}

	var resp *http.Response

	// Send the request
	resp, err = httpClient.Do(req)
	defer func() {
		if resp != nil && resp.Body != nil {
			err := resp.Body.Close()
			if err != nil {
				return
			}
		}
	}()

	if err == nil && resp.StatusCode == http.StatusOK {
		// body json: [{"namespaceName":"application","notificationId":101}]，获取notificationId
		notifications := new([]Notification)
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return errors.Wrap(err, "Error reading notification response body ")
		}
		err = json.Unmarshal(body, notifications)
		if err != nil {
			return errors.Wrap(err, "Error unmarshaling notification response body ")
		}

		// 获取 notificationId
		if len(*notifications) == 0 {
			return errors.New("Error notification response body, " + ", body :" + string(body))
		}
		// Update ApolloConfig Status
		notificationId := (*notifications)[0].NotificationID
		if err := client.configStore.NotifyApolloConfigChange(client.key, notificationId); err == nil {
			client.RemoteResult.NotificationId = notificationId
		} else {
			// update status error, will auto retry apollo notify request
			return err
		}

		return nil
	} else if resp != nil && resp.StatusCode == http.StatusNotModified {
		// unchanged
		return nil
	} else {
		if err == nil {
			err = errors.New("failed to execute HTTP request: " + strconv.Itoa(resp.StatusCode))
		}
		return err
	}
}

// ApolloConfig represents the structure of the JSON response from the Apollo server.
type configRetriveResponse struct {
	AppID          string                 `json:"appId"`
	Cluster        string                 `json:"cluster"`
	Configurations *orderedmap.OrderedMap `json:"configurations"`
	NamespaceName  string                 `json:"namespaceName"`
	ReleaseKey     string                 `json:"releaseKey"`
}

// getRemoteConfig update remote config from apollo
func (client *ApolloClient) getRemoteConfig() (releaseKey string, fileContent string, err error) {
	ctx1, cancel1 := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel1()
	req, err := client.getConfigRequest(ctx1)
	if err != nil {
		return releaseKey, fileContent, err
	}
	resp, err := httpClient.Do(req)
	defer func() {
		if resp != nil && resp.Body != nil {
			err := resp.Body.Close()
			if err != nil {
				return
			}
		}
	}()
	if err != nil {
		err = errors.Wrap(err, "failed to execute HTTP request")
		return releaseKey, fileContent, err
	}

	if resp.StatusCode != http.StatusOK {
		body, err2 := io.ReadAll(resp.Body)
		if err2 != nil {
			err = errors.Wrap(err2, "error reading config response body")
			return releaseKey, fileContent, err
		}
		err = errors.New("error getting config from apollo response: " + string(body))
		return releaseKey, fileContent, err
	}

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		err = errors.Wrap(err, "error reading config response body")
		return releaseKey, fileContent, err
	}

	configResponse := new(configRetriveResponse)
	if err = json.Unmarshal(body, &configResponse); err != nil {
		err = errors.Wrap(err, "error unmarshaling config response: "+string(body))
		return releaseKey, fileContent, err
	}

	// properties file content is key value paris in Configurations
	// for xml、json、yml、yaml、txt file content in key content
	releaseKey = configResponse.ReleaseKey
	if len(configResponse.Configurations.Keys()) == 0 {
		err = errors.New("empty config from apollo: " + string(body))
		return releaseKey, fileContent, err
	} else if content, exist := configResponse.Configurations.Get("content"); len(configResponse.Configurations.Keys()) == 1 && exist {
		fileContent = content.(string)
		return releaseKey, fileContent, err
	} else {
		sb := strings.Builder{}
		for _, key := range configResponse.Configurations.Keys() {
			value, _ := configResponse.Configurations.Get(key)
			sb.WriteString(key)
			sb.WriteString(" = ")
			// value中的特殊符号需要转义
			sb.WriteString(strings.ReplaceAll(value.(string), "\n", "\\n"))
			sb.WriteString("\n")

		}
		fileContent = sb.String()
		return releaseKey, fileContent, err
	}
}

func (client *ApolloClient) stopPolling() {
	client.cancelFunc()
}
