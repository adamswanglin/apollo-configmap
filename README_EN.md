# Apollo-ConfigMap

[中文](README.md) | [English](README_EN.md)

Apollo-ConfigMap is a tool for synchronizing configurations from the [Apollo Config](https://github.com/apolloconfig/apollo) management system to ConfigMaps in a Kubernetes (K8S) cluster.

## Overview

[Apollo Config](https://github.com/apolloconfig/apollo) is a reliable distributed configuration management center that supports managing and distributing configuration files in formats like properties, YAML, XML, and JSON. It offers SDKs for integration with languages such as Java, Go, and Python.

This project introduces a new method for applications in Kubernetes to access configurations. By deploying the apollo-configmap controller and creating custom Kubernetes resources (`ApolloConfigServer` and `ApolloConfig`), the controller automatically syncs configurations from the Apollo configuration center to Kubernetes ConfigMaps. It also detects configuration changes in Apollo, ensuring the ConfigMaps are updated automatically.

## Getting Started

### Deploy the Apollo-ConfigMap Controller

1. Deploy the `apollo-configmap` controller in your Kubernetes cluster:
   ```bash
   kubectl apply -f scripts/deploy.yaml
   ```

2. Check the created resources:
   ```bash
   kubectl get all -n apollo-configmap-system
   ```

   Verify the controller is running by viewing the logs of the controller Pod:
   ```bash
   kubectl logs -n apollo-configmap-system apollo-configmap-controller-manager-<pod-name>
   ```

   Example log output:
   ```bash
   2025-01-22T10:50:20Z INFO setup starting manager
   2025-01-22T10:50:20Z INFO controller-runtime.metrics Starting metrics server
   2025-01-22T10:50:55Z INFO Starting Controller {"controller": "apolloconfigserver", "controllerGroup": "apollo.adamswanglin.com", "controllerKind": "ApolloConfigServer"}
   ```

### Create `ApolloConfigServer` and `ApolloConfig` Resources

This example uses the demo environment from the [Apollo official documentation](https://www.apolloconfig.com/#/zh/README):

- URL: `http://81.68.181.139`
- Username/Password: `apollo/admin`

1. **Create the `ApolloConfigServer` resource:**

   The `ApolloConfigServer` resource maps to the Apollo Config Service instance. Configure the URL of the Apollo [Config Service](https://www.apolloconfig.com/#/zh/design/apollo-design?id=_131-config-service):

   ```yaml
   apiVersion: apollo.adamswanglin.com/v1
   kind: ApolloConfigServer
   metadata:
     name: demo-server
     namespace: default
   spec:
     configServerURL: http://81.68.181.139:8080
   ```

2. **Create the `ApolloConfig` resource:**

   The `ApolloConfig` resource corresponds to a namespace in Apollo. Each configuration file maps to a resource:

   ```yaml
   apiVersion: apollo.adamswanglin.com/v1
   kind: ApolloConfig
   metadata:
     name: demo-config-1
     namespace: default
   spec:
     apollo:
       accessKeySecret: 5e4f59f2035046c2a18e53e31b138f93
       appId: "000111"
       clusterName: default
       namespaceName: application
     apolloConfigServer: default/demo-server
     configMap: apollo-config
     fileName: application1.properties
   ```

3. **Generate the ConfigMap automatically:**

   Check the status of the `ApolloConfig`:
   ```bash
   kubectl get apolloconfig demo-config-1 -o yaml
   ```

   Example status:
   ```yaml
   status:
     lastSynced: "2025-01-23T02:10:49Z"
     notificationId: 35175
     releaseKey: 20250123101103-0d2f651648d02d85
     syncStatus: Success
     updateAt: "2025-01-23T02:10:49Z"
   ```

   View the generated ConfigMap:
   ```bash
   kubectl get cm apollo-config -o yaml
   ```

   Example ConfigMap:
   ```yaml
   apiVersion: v1
   data:
     application1.properties: |
       aps.redis.auth = false
       Test = {"key": "value"}
       1 = false
       c = c
   kind: ConfigMap
   metadata:
     name: apollo-config
     namespace: default
   ```

### Field Descriptions for `ApolloConfig`

| Field Path              | Type   | Example Value                | Description                                                                 |
|-------------------------|--------|------------------------------|-----------------------------------------------------------------------------|
| `spec.apollo`           | Object | -                            | Configuration details for Apollo                                           |
| `spec.apollo.accessKeySecret` | String | -                        | Access key for Apollo                                                      |
| `spec.apollo.appId`     | String | "000111"                    | Application ID in Apollo                                                   |
| `spec.apollo.clusterName` | String | default                     | Cluster name in Apollo                                                     |
| `spec.apollo.namespaceName` | String | application               | Namespace name in Apollo                                                   |
| `spec.apolloConfigServer` | String | demo-server                | Namespaced name of the `ApolloConfigServer` resource                       |
| `spec.configMap`        | String | apollo-config                | Name of the generated ConfigMap                                            |
| `spec.fileName`         | String | application1.properties      | Name of the file in the ConfigMap data; defaults to `namespaceName` if empty|

### Notes

1. Deleting an `ApolloConfig` resource will automatically delete the associated ConfigMap:
   ```bash
   kubectl delete apolloconfig demo-config-1
   ```

2. To retain the ConfigMap while deleting the `ApolloConfig`, use the `--cascade=orphan` flag:
   ```bash
   kubectl delete apolloconfig demo-config-1 --cascade=orphan
   ```

### Metrics & Monitoring

The related Grafana dashboards can be found in the `grafana` folder.

1. **Controller-runtime metrics provided by kubebuilder**  
   See the [controller-runtime metrics reference](https://github.com/kubernetes-sigs/kubebuilder/blob/master/docs/book/src/reference/metrics-reference.md).

2. **Custom Metrics**

```metrics
# Gauge type metric for the number of ApolloConfig resources
# Labels: 
# - resource_namespace: The namespace of the ApolloConfig resource in the Kubernetes cluster.
# - sync_status: Synchronization status, including Success, Fail, or Syncing.
apollo_config_count{resource_namespace="default", sync_status="Success"}

# HTTP request metrics related to accessing Apollo Config Service (Counter and Bucket types)
# Labels: 
# - method: HTTP method.
# - status: HTTP status.
# - url: Request path (currently two types are monitored).
http_requests_total{method="GET", status="OK", url="configs/{appId}/{clusterName}/{namespaceName}"}
http_request_duration_seconds_count{method="GET", status="OK", url="configs/{appId}/{clusterName}/{namespaceName}"}
http_request_duration_seconds_sum{method="GET", status="OK", url="configs/{appId}/{clusterName}/{namespaceName}"}
http_request_duration_seconds_bucket{method="GET", status="OK", url="configs/{appId}/{clusterName}/{namespaceName}"}
``` 

## Development and Deployment

1. Use [kubebuilder markers](https://book.kubebuilder.io/reference/markers.html) to generate Kubernetes configurations:
   ```bash
   make manifests
   ```

2. Generate `deepcopy.go`:
   ```bash
   make generate
   ```

3. Build the Docker image:
   ```bash
   make docker-build
   ```

4. Deploy to the local cluster:
   ```bash
   make deploy
   ```

## License

This project is licensed under the Apache-2.0 License.