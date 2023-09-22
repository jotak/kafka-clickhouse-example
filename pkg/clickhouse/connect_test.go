package clickhouse

import "testing"

func TestConnect(t *testing.T) {
	_, err := connectAndCheck("localhost:9000")
	if err != nil {
		t.Error(err)
	}
}
