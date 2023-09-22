package kafka

import (
	"encoding/json"
)

func decode(b []byte) (map[string]interface{}, error) {
	var rawLine map[string]interface{}
	err := json.Unmarshal(b, &rawLine)
	return rawLine, err
}
