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

# Install Strimzi (Kafka) and create topic
kubectl apply -f https://strimzi.io/install/latest?namespace=netobserv -n netobserv
export DEFAULT_SC=$(kubectl get storageclass -o=jsonpath='{.items[?(@.metadata.annotations.storageclass\.kubernetes\.io/is-default-class=="true")].metadata.name}') && echo "Using SC $DEFAULT_SC"
curl -s -L "https://raw.githubusercontent.com/jotak/kafka-clickhouse-example/main/contrib/kafka.yaml" | envsubst | kubectl apply -n netobserv -f -

# Wait that all pods are up and running, with the KafkaTopic being ready (a few minutes...)
kubectl wait --timeout=180s --for=condition=ready kafkatopic flows-export -n netobserv
kubectl get pods -n netobserv
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

cartago :) SELECT fromUnixTimestamp(intDiv(start,1000)) AS start,fromUnixTimestamp(intDiv(end,1000)) as end,src_ip,dst_ip,src_name,dst_name,src_kind,dst_kind,src_namespace,dst_namespace,bytes,packets FROM flows LIMIT 100

SELECT
    fromUnixTimestamp(intDiv(start, 1000)) AS start,
    fromUnixTimestamp(intDiv(end, 1000)) AS end,
    src_ip,
    dst_ip,
    src_name,
    dst_name,
    src_kind,
    dst_kind,
    src_namespace,
    dst_namespace,
    bytes,
    packets
FROM flows
LIMIT 100

Query id: 21f7ccfc-59ec-4e80-b601-9f5220bf4ffb


┌───────────────start─┬─────────────────end─┬─src_ip──────┬─dst_ip──────┬─src_name─────────┬─dst_name────────────────────────┬─src_kind─┬─dst_kind─┬─src_namespace────────┬─dst_namespace─────┬─bytes─┬─packets─┐
│ 2023-09-26 10:10:32 │ 2023-09-26 10:10:32 │ 10.128.2.13 │ 10.128.2.10 │ prometheus-k8s-0 │ router-default-559c74465f-fh8n6 │ Pod      │ Pod      │ openshift-monitoring │ openshift-ingress │  2649 │       1 │
└─────────────────────┴─────────────────────┴─────────────┴─────────────┴──────────────────┴─────────────────────────────────┴──────────┴──────────┴──────────────────────┴───────────────────┴───────┴─────────┘
┌───────────────start─┬─────────────────end─┬─src_ip──────┬─dst_ip──────┬─src_name──────────────────────────────────┬─dst_name─┬─src_kind─┬─dst_kind─┬─src_namespace─┬─dst_namespace─┬─bytes─┬─packets─┐
│ 2023-09-26 10:10:31 │ 2023-09-26 10:10:31 │ 10.0.144.30 │ 10.0.40.195 │ ip-10-0-144-30.eu-west-3.compute.internal │          │ Node     │          │               │               │    66 │       1 │
└─────────────────────┴─────────────────────┴─────────────┴─────────────┴───────────────────────────────────────────┴──────────┴──────────┴──────────┴───────────────┴───────────────┴───────┴─────────┘
┌───────────────start─┬─────────────────end─┬─src_ip──────┬─dst_ip──────┬─src_name────────────────┬─dst_name──────────────┬─src_kind─┬─dst_kind─┬─src_namespace─┬─dst_namespace─┬─bytes─┬─packets─┐
│ 2023-09-26 10:10:30 │ 2023-09-26 10:10:30 │ 10.129.0.55 │ 10.129.2.21 │ flowlogs-pipeline-hz8mz │ kafka-cluster-kafka-1 │ Pod      │ Pod      │ netobserv     │ netobserv     │  4309 │       4 │
└─────────────────────┴─────────────────────┴─────────────┴─────────────┴─────────────────────────┴───────────────────────┴──────────┴──────────┴───────────────┴───────────────┴───────┴─────────┘
┌───────────────start─┬─────────────────end─┬─src_ip───────┬─dst_ip──────┬─src_name───────────────────────────────────┬─dst_name──────────────────────────────────┬─src_kind─┬─dst_kind─┬─src_namespace─┬─dst_namespace─┬─bytes─┬─packets─┐
│ 2023-09-26 10:10:31 │ 2023-09-26 10:10:31 │ 10.0.200.252 │ 10.0.211.51 │ ip-10-0-200-252.eu-west-3.compute.internal │ ip-10-0-211-51.eu-west-3.compute.internal │ Node     │ Node     │               │               │   124 │       1 │
└─────────────────────┴─────────────────────┴──────────────┴─────────────┴────────────────────────────────────────────┴───────────────────────────────────────────┴──────────┴──────────┴───────────────┴───────────────┴───────┴─────────┘
┌───────────────start─┬─────────────────end─┬─src_ip──────┬─dst_ip──────────┬─src_name──────────────────────────────────┬─dst_name─┬─src_kind─┬─dst_kind─┬─src_namespace─┬─dst_namespace─┬─bytes─┬─packets─┐
│ 2023-09-26 10:10:33 │ 2023-09-26 10:10:33 │ 10.0.144.30 │ 169.254.169.254 │ ip-10-0-144-30.eu-west-3.compute.internal │          │ Node     │          │               │               │   304 │       1 │
└─────────────────────┴─────────────────────┴─────────────┴─────────────────┴───────────────────────────────────────────┴──────────┴──────────┴──────────┴───────────────┴───────────────┴───────┴─────────┘
┌───────────────start─┬─────────────────end─┬─src_ip──────┬─dst_ip──────┬─src_name─────────┬─dst_name──────────────────────────────────┬─src_kind─┬─dst_kind─┬─src_namespace────────┬─dst_namespace─┬─bytes─┬─packets─┐
│ 2023-09-26 10:10:34 │ 2023-09-26 10:10:34 │ 10.129.2.14 │ 10.0.211.51 │ prometheus-k8s-1 │ ip-10-0-211-51.eu-west-3.compute.internal │ Pod      │ Node     │ openshift-monitoring │               │    66 │       1 │
└─────────────────────┴─────────────────────┴─────────────┴─────────────┴──────────────────┴───────────────────────────────────────────┴──────────┴──────────┴──────────────────────┴───────────────┴───────┴─────────┘
┌───────────────start─┬─────────────────end─┬─src_ip──────┬─dst_ip─────┬─src_name────────────────────────────┬─dst_name──────────────────────────────────┬─src_kind─┬─dst_kind─┬─src_namespace────────────────┬─dst_namespace─┬─bytes─┬─packets─┐
│ 2023-09-26 10:10:33 │ 2023-09-26 10:10:33 │ 10.129.0.13 │ 10.129.0.2 │ controller-manager-565b9fb799-vz9w9 │ ip-10-0-211-51.eu-west-3.compute.internal │ Pod      │ Node     │ openshift-controller-manager │               │    66 │       1 │
└─────────────────────┴─────────────────────┴─────────────┴────────────┴─────────────────────────────────────┴───────────────────────────────────────────┴──────────┴──────────┴──────────────────────────────┴───────────────┴───────┴─────────┘
```

We made it!
\o/
