# AxonOps™ Developer Operator

This Kubernetes operator can create a full Apache Cassandra® environment for development that includes
[AxonOps](https://axonops.com)

Apache Cassandra, Cassandra and Apache are either registered trademarks or trademarks of the Apache Software Foundation (http://www.apache.org/) in the United States and/or other countries and are used with permission. The Apache Software Foundation has no affiliation with and does not endorse or review AxonOps Developer Operator

## Installation

The easiest way to install it is by using the helm chart.

```sh
helm upgrade --install axonops-developer-operator --create-namespace -n axonops charts/axonops-developer-operator
```

## Usage

The simplest configuration would be the following to deploy a single Cassandra node with the AxonOps components:

```yaml
apiVersion: cassandra.axonops.com/v1beta1
kind: AxonOpsCassandra
metadata:
  name: axonopscassandra-sample
  namespace: axonops-dev
spec:
```

By default it does not use persistent storage but this is configurable.

```yaml
apiVersion: cassandra.axonops.com/v1beta1
kind: AxonOpsCassandra
metadata:
  name: axonopscassandra-sample
  namespace: axonops-dev
spec:
  cassandra:
    replicas: 3
    clusterName: "my-dev-env"
    image:
      tag: "5.0"
    persistentVolume:
      size: 2Gi
      storageClass: local-path
  axonops:
    elasticsearch:
      persistentVolume:
        size: 2Gi
        storageClass: local-path
```

We only support three Apache Cassandra major releases: 4.0, 4.1 and 5.0 (see `image.tag` above).

## Accessing the AxonOps Dashboard

### Port Forwarding

This operator is meant for developer that would like to test and use an Apache Cassandra cluster locally or in a shared
Kubernetes environment. If you're running locally most likely you cannot use a Load Balancer and therefore you're only
option to access the AxonOps Dashboard is via `kube-proxy`.

```sh
kubectl -n axonops-dev port-forward svc/ds-axonopscassandra-sample 3000:300
```

or

```sh
kubectl port-forward -n axonops-dev $(kubectl get svc -n axonops-dev -l component=dashboard -oname) 3000:3000
```


### Ingress

If you do have an ingress, you can enable it:

```yaml
apiVersion: cassandra.axonops.com/v1beta1
kind: AxonOpsCassandra
metadata:
  name: axonopscassandra-sample
  namespace: axonops-dev
spec:
  axonops:
    dashboard:
      ingress:
        enabled: true
        ingressClassName: nginx
        labels:
          hello: world
        annotations: {}
        hosts:
          - axonops-dev.mydomain.com
```

