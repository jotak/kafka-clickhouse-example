package main

import (
	"flag"

	"github.com/netobserv/flowlogs-pipeline/pkg/clickhouse"
	"github.com/netobserv/flowlogs-pipeline/pkg/kafka"
	"github.com/netobserv/flowlogs-pipeline/pkg/stdout"
	"github.com/netobserv/flowlogs-pipeline/pkg/utils"
	"github.com/sirupsen/logrus"
)

var (
	app           = "kafka-clickhouse-example"
	log           = logrus.WithField("module", "main")
	logLevel      = flag.String("loglevel", "info", "log level (default: info)")
	kafkaURL      = flag.String("kafkaurl", "", "Kafka bootstrap URL")
	clickhouseURL = flag.String("clickhouseurl", "", "Clickhouse URL")
)

func main() {
	flag.Parse()

	lvl, err := logrus.ParseLevel(*logLevel)
	if err != nil {
		log.Errorf("Log level %s not recognized, using info", *logLevel)
		*logLevel = "info"
		lvl = logrus.InfoLevel
	}
	logrus.SetLevel(lvl)
	log.Infof("Starting %s at log level %s", app, *logLevel)

	utils.SetupElegantExit()
	kafkaOut := make(chan map[string]interface{})
	go kafka.StartKafkaReader(*kafkaURL, kafkaOut)

	if clickhouseURL == nil || *clickhouseURL == "" {
		stdout.StartStdoutExport(kafkaOut)
	} else {
		clickhouse.StartClickhouseExport(*clickhouseURL, kafkaOut)
	}
}
