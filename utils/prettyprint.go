package utils

import (
	"encoding/json"
)

func PrettyPrintJSON(input []byte) string {
	var out map[string]interface{}
	err := json.Unmarshal(input, &out)
	if err != nil {
		return "Invalid JSON: " + err.Error()
	}
	pretty, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return "Error formatting JSON: " + err.Error()
	}
	return string(pretty)
}
