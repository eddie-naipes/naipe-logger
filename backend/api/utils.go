package api

import (
	"strconv"
	"strings"
)

func getStringValue(data map[string]interface{}, key string) string {
	if val, ok := data[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getMapValue(data map[string]interface{}, key string) map[string]interface{} {
	if val, ok := data[key]; ok {
		if m, ok := val.(map[string]interface{}); ok {
			return m
		}
	}
	return map[string]interface{}{}
}

func getFloat64Value(data map[string]interface{}, key string) float64 {
	if val, ok := data[key]; ok {
		if f, ok := val.(float64); ok {
			return f
		}
		if s, ok := val.(string); ok {
			if f, err := strconv.ParseFloat(s, 64); err == nil {
				return f
			}
		}
	}
	return 0
}

func getBoolValue(data map[string]interface{}, key string) bool {
	if val, ok := data[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
		if s, ok := val.(string); ok {
			return strings.ToLower(s) == "true" || s == "1"
		}
		if n, ok := val.(float64); ok {
			return n > 0
		}
	}
	return false
}
