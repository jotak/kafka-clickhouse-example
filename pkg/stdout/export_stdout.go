package stdout

import (
	"fmt"

	"github.com/netobserv/flowlogs-pipeline/pkg/utils"
	"github.com/sirupsen/logrus"
)

var klog = logrus.WithField("component", "Stdout")

func StartStdoutExport(out chan map[string]interface{}) {
	klog.Info("sarting stdout exporter")
	for {
		select {
		case flow := <-out:
			process(flow)
		case <-utils.ExitChannel():
			klog.Info("gracefully exiting")
			return
		}
	}
}

func process(flow map[string]interface{}) {
	fmt.Println(flow)
}
