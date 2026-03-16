package etl

import (
	"encoding/json"
	"fmt"
)

func init() {
	RegisterFunction("log_rows", func(data []map[string]string) ([]map[string]string, error) {
		for _, row := range data {
			b, _ := json.Marshal(row)
			fmt.Printf("FUNCTION CALL: %s\n", string(b))
		}
		return data, nil
	})
}
