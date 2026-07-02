package demo

import (
	"encoding/json"
	"time"
)

const Name = "demo-task"

func Execute(payload json.RawMessage) (any, error) {
	time.Sleep(5 * time.Second)

	return map[string]any{
		"message":      "task completed",
		"payload_size": len(payload),
	}, nil
}
