package clickhouse

import (
	"context"
	"fmt"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

func connect(url string) (driver.Conn, error) {
	var (
		ctx       = context.Background()
		conn, err = ch.Open(&ch.Options{
			Addr: []string{url},
			ClientInfo: ch.ClientInfo{
				Products: []struct {
					Name    string
					Version string
				}{
					{Name: "an-example-go-client", Version: "0.1"},
				},
			},

			Debugf: func(format string, v ...interface{}) {
				fmt.Printf(format, v)
			},
		})
	)

	if err != nil {
		return nil, err
	}

	if err := conn.Ping(ctx); err != nil {
		if exception, ok := err.(*ch.Exception); ok {
			fmt.Printf("Exception [%d] %s \n%s\n", exception.Code, exception.Message, exception.StackTrace)
		}
		return nil, err
	}
	return conn, nil
}

func connectAndCheck(url string) (driver.Conn, error) {
	klog.Info("checking clickhouse connection...")

	conn, err := connect(url)
	if err != nil {
		return nil, fmt.Errorf("connection failure: %w", err)
	}
	ctx := context.Background()
	rows, err := conn.Query(ctx, "SELECT name,toString(uuid) as uuid_str FROM system.tables LIMIT 5")
	if err != nil {
		return nil, fmt.Errorf("probing query failed: %w", err)
	}

	for rows.Next() {
		var (
			name, uuid string
		)
		if err := rows.Scan(&name, &uuid); err != nil {
			return nil, fmt.Errorf("probing scan failed: %w", err)
		}
		klog.Infof("name: %s, uuid: %s", name, uuid)
	}
	return conn, nil
}

func setupTable(conn driver.Conn) error {
	if err := conn.Exec(context.Background(), `DROP TABLE IF EXISTS flows`); err != nil {
		return err
	}
	// TODO: use DateTime
	if err := conn.Exec(context.Background(), `
    CREATE TABLE IF NOT EXISTS flows (
				start String,
				end String,
        src_ip String,
        dst_ip String,
        src_name String,
        dst_name String,
        src_kind String,
        dst_kind String,
        src_namespace String,
        dst_namespace String,
        bytes UInt32,
        packets UInt32,
    ) engine=Memory
`); err != nil {
		return err
	}
	return nil
}
