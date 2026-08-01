

# Apollo-ConfigMap

Español | [English](README_EN.md)

Apollo-ConfigMap se utiliza para sincronizar las configuraciones del centro de configuración [Apollo Config](https://github.com/apolloconfig/apollo) hacia los ConfigMap del clúster de K8S.

## Descripción
 [Apollo Config](https://github.com/apolloconfig/apollo) es un centro de gestión de configuración distribuido y confiable que admite la gestión y distribución de archivos de configuración como properties/yaml/xml/json; en cuanto a la integración del cliente, admite SDK para lenguajes como Java/Go/Python.

Este proyecto proporciona una nueva forma de que las aplicaciones en K8S se conecten al centro de configuración de Apollo:

Desplegar el controlador apollo-configmap y crear los recursos de K8S ApolloConfigServer y ApolloConfig;

El controlador sincronizará automáticamente las configuraciones del centro de configuración de Apollo hacia los ConfigMap del clúster de K8S, y detectará automáticamente los cambios de configuración de Apollo, logrando la actualización automática de los ConfigMap cuando cambien las configuraciones.

## Guía de inicio

### Desplegar el controlador apollo-configmap:

1. Desplegar el controlador apollo-configmap en el clúster de K8S:
```bash
kubectl apply -f dist/install.yaml
```

2. Ver los recursos creados:
```bash
kubectl get all -n apollo-configmap-system
```
Consultar los registros (logs) del Pod del controlador para asegurarse de que funciona correctamente:
```bash
 kubectl logs -n apollo-configmap-system apollo-configmap-controller-manager-7588796dc6-hgf5r
```
Debería ver registros similares a los siguientes:
   
```bash
2025-01-22T10:50:20Z	INFO	setup	starting manager
2025-01-22T10:50:20Z	INFO	controller-runtime.metrics	Starting metrics server
2025-01-22T10:50:55Z	INFO	Starting Controller	{"controller": "apolloconfigserver", "controllerGroup": "apollo.adamswanglin.com", "controllerKind": "ApolloConfigServer"}
2025-01-22T10:50:55Z	INFO	Starting Controller	{"controller": "apolloconfig", "controllerGroup": "apollo.adamswanglin.com", "controllerKind": "ApolloConfig"}
2025-01-22T10:50:55Z	INFO	Starting workers	{"controller": "apolloconfigserver", "controllerGroup": "apollo.adamswanglin.com", "controllerKind": "ApolloConfigServer", "worker count": 1}
2025-01-22T10:50:55Z	INFO	Starting workers	{"controller": "apolloconfig", "controllerGroup": "apollo.adamswanglin.com", "controllerKind": "ApolloConfig", "worker count": 1}

```


### Crear recursos ApolloConfigServer y ApolloConfig

Este ejemplo utiliza el entorno de demostración (Demo) de la [documentación oficial de Apollo](https://www.apolloconfig.com/#/zh/README):

http://81.68.181.139
Usuario/Contraseña: apollo/admin

1. Crear el recurso ApolloConfigServer:

El recurso ApolloConfigServer corresponde a una instancia de servicio Apollo Config Service. Lo que se debe configurar es la dirección del [Config Service](https://www.apolloconfig.com/#/zh/design/apollo-design?id=_131-config-service) de Apollo.
```yaml
apiVersion: apollo.adamswanglin.com/v1
kind: ApolloConfigServer
metadata:
  name: demo-server
  namespace: default
spec:
  configServerURL: http://81.68.181.139:8080

```

2. Crear el recurso ApolloConfig:
ApolloConfig corresponde a un espacio de nombres (namespace) de Apollo. En términos sencillos, se despliega un recurso por cada archivo de configuración.
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
3. Generación automática de ConfigMap
```bash
kubectl get apolloconfig demo-config-1 -o yaml
```
Cambios en el estado de ApolloConfig
```yaml
apiVersion: apollo.adamswanglin.com/v1
kind: ApolloConfig
...
status:
  lastSynced: "2025-01-23T02:10:49Z"
  notificationId: 35175
  releaseKey: 20250123101103-0d2f651648d02d85 # apollo release key
  syncStatus: Success # Estado de sincronización
  updateAt: "2025-01-23T02:10:49Z"
```
ConfigMap generado automáticamente
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

### Descripción de los campos de ApolloConfig

[Conceptos relacionados](https://www.apolloconfig.com/#/zh/design/apollo-introduction?id=_41-core-concepts) de Apollo

| Ruta del campo              | Tipo       | Valor de ejemplo              | Descripción                                                                                                                               |
|-----------------------------|------------|-------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------|
| spec.apollo                 | Objeto     | -                             | [Conceptos relacionados](https://www.apolloconfig.com/#/zh/design/apollo-introduction?id=_41-core-concepts) de Apollo                     |
| spec.apollo.accessKeySecret | Cadena     | -                             | Clave secreta para acceder a Apollo                                                                                                       |
| spec.apollo.appId           | Cadena     | "000111"                      | ID de la aplicación en Apollo                                                                                                             |
| spec.apollo.clusterName     | Cadena     | default                       | Nombre del clúster de Apollo                                                                                                              |
| spec.apollo.namespaceName   | Cadena     | application                   | Espacio de nombres de Apollo<br />El tipo properties puede no tener extensión, por ejemplo `application`, mientras que tipos como json/xml/yaml deben tener extensión, por ejemplo `application.json`, `application.yaml`, etc. |
| spec.apolloConfigServer     | Cadena     | demo-server                   | NamespacedName de ApolloConfigServer en K8S<br />Formato {{namespace}}/{{name}}                                                           |
| spec.configMap              | Cadena     | apollo-config                 | Nombre del ConfigMap generado automáticamente                                                                                             |
| spec.fileName               | Cadena     | application1.properties       | Especifica el nombre del archivo en ConfigMapData. Puede dejarse vacío; si está vacío, se usará apollo.namespaceName                      |

### Notas importantes

1. Al eliminar el recurso ApolloConfig, también se eliminará automáticamente el recurso ConfigMap asociado.
```bash
# Eliminar ApolloConfig y el ConfigMap asociado
kubectl delete apolloconfigs  demo-config-1
```
Si desea conservar el ConfigMap y solo eliminar ApolloConfig, utilice `--cascade=orphan`
```bash
# Eliminar solo ApolloConfig
kubectl delete apolloconfigs  demo-config-1 --cascade=orphan
```
## Monitoreo de métricas

Para ver los dashboards de Grafana relacionados, consulte la carpeta grafana

1. [Métricas de controller-runtime](https://github.com/kubernetes-sigs/kubebuilder/blob/master/docs/book/src/reference/metrics-reference.md) incluidas en kubebuilder.

2. Métricas personalizadas
```metrics
# ApolloConfig 数量 Gauge类型
# label: resource_namespace ApolloConfig 所在K8S集群namespace, sync_status 同步状态 包括 Success/Fail/Syncing
apollo_config_count{resource_namespace="default", sync_status="Success"}

# 访问Apollo Config Service的http请求相关Counter和Bucket类型
# label: method HTTP Method, status HTTP Status， url 请求路径(目前有两种）
http_requests_total{method="GET", status="OK", url="configs/{appId}/{clusterName}/{namespaceName}"}
http_request_duration_seconds_count{method="GET", status="OK", url="configs/{appId}/{clusterName}/{namespaceName}"}
http_request_duration_seconds_sum{method="GET", status="OK", url="configs/{appId}/{clusterName}/{namespaceName}"}
http_request_duration_seconds_bucket{method="GET", status="OK", url="configs/{appId}/{clusterName}/{namespaceName}"}
```

## Despliegue para desarrollo
1. Este paquete utiliza [marcadores de kubebuilder](https://book.kubebuilder.io/reference/markers.html) para generar la configuración de Kubernetes. Ejecute `make manifests` para crear CRDs y roles en `config/crd` y `config/rbac`.
2. Ejecute `make generate` para generar `deepcopy.go`.
3. Ejecute `make docker-build` para construir la imagen.
4. Ejecute `make deploy` para desplegar en el clúster local.

## Licencia

Este proyecto está licenciado bajo la Licencia Apache-2.0.
