# Kafka consumer example

This is a sample application show-casing how to consume NetObserv flows via Kafka, and export them to a custom storage (here, a Clickhouse database).

## Run it

For simplicity of deployment, we will deploy Clickhouse locally and use [ktunnel](https://github.com/omrikiei/ktunnel) for reverse port-forwarding. Obviously, you don't need `ktunnel` if you already have an access to a Clickhouse server from your Kubernetes cluster.

### Prerequisites

- An OpenShift or Kubernetes cluster
- [NetObserv operator](https://github.com/netobserv/network-observability-operator) installed.
- Clickhouse binary: for example, run `curl https://clickhouse.com/ | sh` (cf the [quick install guide](https://clickhouse.com/docs/en/install#quick-install))
- [ktunnel](https://github.com/omrikiei/ktunnel) binary
- Common tooling: `curl`, `kubectl`, `envsubst`...

### Prepare Kafka

```bash
# Create a namespace for all the deployments
kubectl create namespace netobserv

# Install Strimzi (Kafka)
kubectl apply -f https://strimzi.io/install/latest?namespace=netobserv -n netobserv
DEFAULT_SC=$(kubectl get storageclass -o=jsonpath='{.items[?(@.metadata.annotations.storageclass\.kubernetes\.io/is-default-class=="true")].metadata.name}') && echo "Using SC $DEFAULT_SC"
curl -s -L "https://raw.githubusercontent.com/netobserv/documents/main/examples/kafka/default.yaml" | envsubst | kubectl apply -n netobserv -f -
kubectl apply -f https://raw.githubusercontent.com/jotak/kafka-clickhouse-example/main/contrib/topic.yaml -n netobserv

# Wait to see all pods up and running (a few minutes...)
kubectl get pods -n netobserv -w

# Make sure the created Kafka topic is Ready
kubectl get kafkatopic flows-export -w
```

### Prepare NetObserv

Assuming you already installed the operator, now you must create a `FlowCollector` resource that will start sending flow logs to Kafka. We don't need to setup Loki if all we want are the flows into Kafka / Clickhouse.

Note that we configure here Kafka as an **exporter**, which is unrelated to the `spec.deploymentModel: KAFKA` / `spec.kafka` settings: those ones correspond to NetObserv's internal flows processing configuration (NetObserv being both the producer and the consumer), whereas `spec.exporters` relates to NetObserv being just the producer, leaving up to us how we want to consume it.

```bash
cat <<EOF | kubectl apply -f -
apiVersion: flows.netobserv.io/v1beta1
kind: FlowCollector
metadata:
  name: cluster
spec:
  namespace: netobserv
  deploymentModel: DIRECT
  loki:
    enable: false
  exporters:
    - type: KAFKA
      kafka:
        address: "kafka-cluster-kafka-bootstrap.netobserv"
        topic: flows-export
EOF
```

### Start Clickhouse with ktunnel

Using the Clickhouse binary that you downloaded, run:

```bash
./clickhouse server
```

This will start a Clickhouse server that listens on `:9000` on your machine.

In another terminal, setup ktunnel:

```bash
ktunnel expose clickhouse 9000:9000
```

It will create a clickhouse service in the `default` namespace, bridged to your local server.

### Run this sample app

Now that almost all pieces are up and running, we just need to bring the missing one: a Kafka consumer that will send flows to Clickhouse. This is what this repository is about. Just run:

```bash
kubectl apply -f https://raw.githubusercontent.com/jotak/kafka-clickhouse-example/main/contrib/deployment.yaml -n netobserv
```

### Check Clickhouse content

Now we can verify that flows are getting their way to the database. We can use the clickhouse client for that purpose:

```bash
./clickhouse client

cartago :) SELECT * FROM flows

SELECT *
FROM flows

Query id: ba2e551b-4219-4762-b44b-1d8f0b234387

┌─src_ip──────┬─dst_ip──────┬─src_name──────────────────────────────────┬─dst_name──────────────────────────────────┬─src_kind─┬─dst_kind─┬─src_namespace─┬─dst_namespace─┬─bytes─┬─packets─┐
│ 10.0.204.65 │ 10.0.189.25 │ ip-10-0-204-65.eu-west-3.compute.internal │ ip-10-0-189-25.eu-west-3.compute.internal │ Node     │ Node     │               │               │   132 │       2 │
└─────────────┴─────────────┴───────────────────────────────────────────┴───────────────────────────────────────────┴──────────┴──────────┴───────────────┴───────────────┴───────┴─────────┘
┌─src_ip──────┬─dst_ip──────┬─src_name──────────────────────────────────┬─dst_name──────────────────────────────────┬─src_kind─┬─dst_kind─┬─src_namespace─┬─dst_namespace─┬─bytes─┬─packets─┐
│ 10.0.189.25 │ 10.0.204.65 │ ip-10-0-189-25.eu-west-3.compute.internal │ ip-10-0-204-65.eu-west-3.compute.internal │ Node     │ Node     │               │               │    66 │       1 │
└─────────────┴─────────────┴───────────────────────────────────────────┴───────────────────────────────────────────┴──────────┴──────────┴───────────────┴───────────────┴───────┴─────────┘
┌─src_ip──────┬─dst_ip──────┬─src_name─────────┬─dst_name───────────────────────┬─src_kind─┬─dst_kind─┬─src_namespace────────┬─dst_namespace───────────┬─bytes─┬─packets─┐
│ 10.128.2.17 │ 10.129.0.30 │ prometheus-k8s-0 │ etcd-operator-5d9f6db48c-gqdg7 │ Pod      │ Pod      │ openshift-monitoring │ openshift-etcd-operator │   114 │       1 │
└─────────────┴─────────────┴──────────────────┴────────────────────────────────┴──────────┴──────────┴──────────────────────┴─────────────────────────┴───────┴─────────┘
┌─src_ip───────┬─dst_ip──────┬─src_name───────────────────────────────────┬─dst_name──────────────────────────────────┬─src_kind─┬─dst_kind─┬─src_namespace─┬─dst_namespace─┬─bytes─┬─packets─┐
│ 10.0.206.142 │ 10.0.155.14 │ ip-10-0-206-142.eu-west-3.compute.internal │ ip-10-0-155-14.eu-west-3.compute.internal │ Node     │ Node     │               │               │   125 │       1 │
└──────────────┴─────────────┴────────────────────────────────────────────┴───────────────────────────────────────────┴──────────┴──────────┴───────────────┴───────────────┴───────┴─────────┘
┌─src_ip──────┬─dst_ip──────┬─src_name─────────┬─dst_name───────────────────────┬─src_kind─┬─dst_kind─┬─src_namespace────────┬─dst_namespace───────────┬─bytes─┬─packets─┐
│ 10.128.2.17 │ 10.129.0.30 │ prometheus-k8s-0 │ etcd-operator-5d9f6db48c-gqdg7 │ Pod      │ Pod      │ openshift-monitoring │ openshift-etcd-operator │   228 │       2 │
└─────────────┴─────────────┴──────────────────┴────────────────────────────────┴──────────┴──────────┴──────────────────────┴─────────────────────────┴───────┴─────────┘
┌─src_ip──────┬─dst_ip───────┬─src_name──────────────────────────────────┬─dst_name─┬─src_kind─┬─dst_kind─┬─src_namespace─┬─dst_namespace─┬─bytes─┬─packets─┐
│ 10.0.189.25 │ 10.0.173.213 │ ip-10-0-189-25.eu-west-3.compute.internal │          │ Node     │          │               │               │   587 │       1 │
└─────────────┴──────────────┴───────────────────────────────────────────┴──────────┴──────────┴──────────┴───────────────┴───────────────┴───────┴─────────┘
┌─src_ip──────┬─dst_ip───────┬─src_name──────────────────────────────────┬─dst_name─┬─src_kind─┬─dst_kind─┬─src_namespace─┬─dst_namespace─┬─bytes─┬─packets─┐
│ 10.0.189.25 │ 10.0.173.213 │ ip-10-0-189-25.eu-west-3.compute.internal │          │ Node     │          │               │               │   327 │       2 │
└─────────────┴──────────────┴───────────────────────────────────────────┴──────────┴──────────┴──────────┴───────────────┴───────────────┴───────┴─────────┘
┌─src_ip───────┬─dst_ip──────┬─src_name───────────────────────────────────┬─dst_name──────────────────────────────────┬─src_kind─┬─dst_kind─┬─src_namespace─┬─dst_namespace─┬─bytes─┬─packets─┐
│ 10.0.206.142 │ 10.0.189.25 │ ip-10-0-206-142.eu-west-3.compute.internal │ ip-10-0-189-25.eu-west-3.compute.internal │ Node     │ Node     │               │               │  1531 │       1 │
└──────────────┴─────────────┴────────────────────────────────────────────┴───────────────────────────────────────────┴──────────┴──────────┴───────────────┴───────────────┴───────┴─────────┘

```

We're done!
\o/
