# Apollo-ConfigMap

[中文](README.md) | [English](README_EN.md)

Apollo-ConfigMap 用来同步 [Apollo Config](https://github.com/apolloconfig/apollo) 配置中心的配置到K8S集群的ConfigMap中。

## 说明
 [Apollo Config](https://github.com/apolloconfig/apollo) 是一款可靠的分布式配置管理中心，支持properties/yaml/xml/json等配置文件的管理和分发；客户端接入上支持Java/Go/Python等语言的SDK接入。

本项目提供一个新的 K8S 中应用的接入Apollo配置中心的方式：

部署apollo-configmap controller, 创建ApolloConfigServer和ApolloConfig K8S资源；

controller 会自动同步Apollo配置中心的配置到 K8S 集群的 ConfigMap中，并且自动感知Apollo的配置变更，实现配置变更自动更新ConfigMap。

## 入门指南

### 部署apollo-configmap controller:

1. K8S集群中部署apollo-configmap controller：
```bash
kubectl apply -f scripts/deploy.yaml
```

2. 查看已创建的资源：
```bash
kubectl get all -n apollo-configmap-system
```
查看controller Pod 的日志，确保controller正常运行：
```bash
 kubectl logs -n apollo-configmap-system apollo-configmap-controller-manager-7588796dc6-hgf5r
```
您应该看到类似以下的日志：
   
```bash
2025-01-22T10:50:20Z	INFO	setup	starting manager
2025-01-22T10:50:20Z	INFO	controller-runtime.metrics	Starting metrics server
2025-01-22T10:50:55Z	INFO	Starting Controller	{"controller": "apolloconfigserver", "controllerGroup": "apollo.adamswanglin.com", "controllerKind": "ApolloConfigServer"}
2025-01-22T10:50:55Z	INFO	Starting Controller	{"controller": "apolloconfig", "controllerGroup": "apollo.adamswanglin.com", "controllerKind": "ApolloConfig"}
2025-01-22T10:50:55Z	INFO	Starting workers	{"controller": "apolloconfigserver", "controllerGroup": "apollo.adamswanglin.com", "controllerKind": "ApolloConfigServer", "worker count": 1}
2025-01-22T10:50:55Z	INFO	Starting workers	{"controller": "apolloconfig", "controllerGroup": "apollo.adamswanglin.com", "controllerKind": "ApolloConfig", "worker count": 1}

```


### 创建 ApolloConfigServer 和 ApolloConfig 资源

本示例使用[Apollo官方文档](https://www.apolloconfig.com/#/zh/README)中的演示环境（Demo）:

http://81.68.181.139
账号/密码:apollo/admin

1. 创建 ApolloConfigServer 资源:

ApolloConfigServer 资源对应 Apollo Config Service 服务实例, 需要配置的是Apollo [Config Service](https://www.apolloconfig.com/#/zh/design/apollo-design?id=_131-config-service) 的地址。
```yaml
apiVersion: apollo.adamswanglin.com/v1
kind: ApolloConfigServer
metadata:
  name: demo-server
  namespace: default
spec:
  configServerURL: http://81.68.181.139:8080

```

2. 创建 ApolloConfig 资源:
ApolloConfig对应的是 Apollo的namespace, 简单理解一个配置文件部署一个资源。
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
3. 自动生成ConfigMap
```bash
kubectl get apolloconfig demo-config-1 -o yaml
```
ApolloConfig中状态变化
```yaml
apiVersion: apollo.adamswanglin.com/v1
kind: ApolloConfig
...
status:
  lastSynced: "2025-01-23T02:10:49Z"
  notificationId: 35175
  releaseKey: 20250123101103-0d2f651648d02d85 # apollo release key
  syncStatus: Success # 同步状态
  updateAt: "2025-01-23T02:10:49Z"
```
自动生成的ConfigMap
```bash
kubectl get cm apollo-config -o yaml
```
```yaml
apiVersion: v1
data:
  application1.properties: |
    aps.redis.auth = false
    Test = {\n"key": "value"\n}
    1 = false
    c = c
kind: ConfigMap
metadata:
  creationTimestamp: "2025-01-17T06:44:08Z"
  name: apollo-config
  namespace: default
  ownerReferences:
  - apiVersion: apollo.adamswanglin.com/v1
    blockOwnerDeletion: true
    controller: true
    kind: ApolloConfig
    name: demo-config-1
    uid: 6134b53a-82c0-4c6b-93bf-147ab4b4eca0
  resourceVersion: "7393121"
  uid: 1f62db9f-d098-4871-8ef1-9664aafb4239
```

### ApolloConfig 字段说明

Apollo[相关概念](https://www.apolloconfig.com/#/zh/design/apollo-introduction?id=_41-core-concepts)

| 字段路径                  | 类型       | 示例值                          | 说明                                   |
|--------------------------|------------|---------------------------------|---------------------------------------|
| spec.apollo              | 对象       | -                               | Apollo [相关概念](https://www.apolloconfig.com/#/zh/design/apollo-introduction?id=_41-core-concepts)                       |
| spec.apollo.accessKeySecret | 字符串   | - | 用于访问 Apollo 的密钥               |
| spec.apollo.appId        | 字符串     | "000111"                        | Apollo 中的应用 ID                    |
| spec.apollo.clusterName  | 字符串     | default                         | Apollo 集群名称                       |
| spec.apollo.namespaceName | 字符串    | application                     | Apollo 命名空间<br />properties类型可以不带后缀例如application, json/xml/yaml等类型需要带后缀例如application.json, application.yaml等 |
| spec.apolloConfigServer  | 字符串     | demo-server                     | K8S中ApolloConfigServer的NamespacedName<br />格式{{namespace}}/{{name}}       |
| spec.configMap           | 字符串     | apollo-config                   | 自动生成的 ConfigMap 名称         |
| spec.fileName            | 字符串     | application1.properties         | 指定ConfigMapData中的文件名，可以为空，为空取apollo.namespaceName                    |

### 注意事项

1. 删除 ApolloConfig 资源，会自动删除关联的 ConfigMap 资源。
```bash
# 删除ApolloConfig和关联的ConfigMap
kubectl delete apolloconfigs  demo-config-1
```
如果希望保留ConfigMap，仅删除ApolloConfig，请使用`--cascade=orphan`
```bash
# 仅删除ApolloConfig
kubectl delete apolloconfigs  demo-config-1 --cascade=orphan
```

## 开发部署
1. 本包使用 [kubebuilder markers](https://book.kubebuilder.io/reference/markers.html) 生成 Kubernetes 配置。运行 `make manifests` 以在 `config/crd` 和 `config/rbac` 中创建 CRD 和角色。
2. 运行 `make generate` 生成 `deepcopy.go`。
3. 运行 `make docker-build` 构建镜像。
4. 运行 `make deploy`部署到本机集群。

## 许可证

本项目基于 Apache-2.0 许可证授权。