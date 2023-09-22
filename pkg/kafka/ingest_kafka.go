package kafka

import (
	"time"

	"github.com/netobserv/flowlogs-pipeline/pkg/utils"
	kafkago "github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

var klog = logrus.WithField("component", "KafkaReader")

func StartKafkaReader(url string, out chan map[string]interface{}) {
	cfg := kafkago.ReaderConfig{
		Brokers:        []string{url},
		Topic:          "flows-export",
		GroupID:        "to-clickhouse",
		GroupBalancers: []kafkago.GroupBalancer{&kafkago.RoundRobinGroupBalancer{}},
		StartOffset:    kafkago.FirstOffset,
		CommitInterval: time.Duration(500 * time.Millisecond),
		Dialer: &kafkago.Dialer{
			Timeout:   kafkago.DefaultDialer.Timeout,
			DualStack: kafkago.DefaultDialer.DualStack,
		},
	}

	r := kafkago.NewReader(cfg)

	for {
		if utils.IsStopped() {
			klog.Info("gracefully exiting")
			break
		}
		m, err := r.ReadMessage(context.Background())
		if err != nil {
			break
		}
		klog.Tracef("message at offset %d: %s = %s\n", m.Offset, string(m.Key), string(m.Value))
		if flow, err := decode(m.Value); err != nil {
			klog.Errorf("failed to decode: %v", err)
		} else {
			out <- flow
		}
	}

	if err := r.Close(); err != nil {
		klog.Errorf("failed to close reader: %v", err)
	}
}
