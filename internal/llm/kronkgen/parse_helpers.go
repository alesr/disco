package kronkgen

import (
	"fmt"
	"strconv"
	"strings"
)

func containsIgnoreCase(values []string, target string) bool {
	trimmed := strings.TrimSpace(target)
	for _, value := range values {
		if strings.EqualFold(value, trimmed) {
			return true
		}
	}
	return false
}

func normalizeJSONObject(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	if strings.HasPrefix(trimmed, "```") {
		// providers occasionally wrap json in markdown fences and we normalize before strict parsing
		trimmed = strings.TrimPrefix(trimmed, "```json")
		trimmed = strings.TrimPrefix(trimmed, "```")
		trimmed = strings.TrimSuffix(trimmed, "```")
		trimmed = strings.TrimSpace(trimmed)
	}

	start := strings.Index(trimmed, "{")
	end := strings.LastIndex(trimmed, "}")
	if start >= 0 && end > start {
		return strings.TrimSpace(trimmed[start : end+1])
	}
	return trimmed
}

func normalizeSeverityValue(severity string) string {
	normalized := strings.ToLower(strings.TrimSpace(severity))
	if normalized == "critical" {
		// preserves existing severity gates and output contracts
		return "high"
	}
	return normalized
}

func coerceString(value any) string {
	if value == nil {
		return ""
	}

	if text, ok := value.(string); ok {
		return strings.TrimSpace(text)
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func coerceBool(value any) (bool, error) {
	switch v := value.(type) {
	case bool:
		return v, nil
	case string:
		parsed, err := strconv.ParseBool(strings.TrimSpace(v))
		if err != nil {
			return false, err
		}
		return parsed, nil
	default:
		return false, fmt.Errorf("unsupported bool type %T", value)
	}
}

func coerceFloat32(value any) (float32, error) {
	switch v := value.(type) {
	case float64:
		return float32(v), nil
	case float32:
		return v, nil
	case int:
		return float32(v), nil
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(v), 32)
		if err != nil {
			return 0, err
		}
		return float32(parsed), nil
	default:
		return 0, fmt.Errorf("unsupported confidence type %T", value)
	}
}
