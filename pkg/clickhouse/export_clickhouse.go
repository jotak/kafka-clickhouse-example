package clickhouse

import (
	"context"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/netobserv/flowlogs-pipeline/pkg/utils"
	"github.com/netobserv/network-observability-console-plugin/pkg/model/fields"
	"github.com/sirupsen/logrus"
)

var klog = logrus.WithField("component", "ClickhouseExport")

func StartClickhouseExport(url string, out chan map[string]interface{}) {
	klog.Info("starting clickhouse exporter")

	conn, err := connectAndCheck(url)
	if err != nil {
		klog.Fatal(err)
	}
	err = setupTable(conn)
	if err != nil {
		klog.Fatal(err)
	}

	for {
		select {
		case flow := <-out:
			process(conn, flow)
		case <-utils.ExitChannel():
			klog.Info("gracefully exiting")
			conn.Close()
			return
		}
	}
}

func process(conn driver.Conn, rawFlow map[string]interface{}) {
	klog.Tracef("Exporting to clickhouse: %v", rawFlow)
	var (
		srcAddr, dstAddr, srcName, dstName, srcKind, dstKind, srcNamespace, dstNamespace string
		bytes, packets                                                                   int
	)
	if v, ok := rawFlow[fields.SrcAddr]; ok {
		srcAddr = v.(string)
	}
	if v, ok := rawFlow[fields.DstAddr]; ok {
		dstAddr = v.(string)
	}
	if v, ok := rawFlow[fields.SrcName]; ok {
		srcName = v.(string)
	}
	if v, ok := rawFlow[fields.DstName]; ok {
		dstName = v.(string)
	}
	if v, ok := rawFlow[fields.SrcType]; ok {
		srcKind = v.(string)
	}
	if v, ok := rawFlow[fields.DstType]; ok {
		dstKind = v.(string)
	}
	if v, ok := rawFlow[fields.SrcNamespace]; ok {
		srcNamespace = v.(string)
	}
	if v, ok := rawFlow[fields.DstNamespace]; ok {
		dstNamespace = v.(string)
	}
	if v, ok := rawFlow[fields.Bytes]; ok {
		bytes = int(v.(float64))
	}
	if v, ok := rawFlow[fields.Packets]; ok {
		packets = int(v.(float64))
	}
	q := fmt.Sprintf("INSERT INTO flows VALUES ('%s','%s','%s','%s','%s','%s','%s','%s','%d','%d')",
		srcAddr, dstAddr, srcName, dstName, srcKind, dstKind, srcNamespace, dstNamespace, bytes, packets,
	)
	if err := conn.Exec(context.Background(), q); err != nil {
		klog.Warnf("Insertion error: %v", err)
	}
}
