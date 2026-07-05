package model

import (
	"encoding/json"
	"sort"
	"strings"
)

func BuildArgs(raw string) map[string]string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]string{}
	}
	var values map[string]string
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return map[string]string{}
	}
	return NormalizeBuildArgs(values)
}

func NormalizeBuildArgs(values map[string]string) map[string]string {
	normalized := make(map[string]string)
	for key, value := range values {
		key = strings.TrimSpace(key)
		if !IsBuildArgKey(key) {
			continue
		}
		normalized[key] = strings.TrimSpace(value)
	}
	return normalized
}

func EncodeBuildArgs(values map[string]string) string {
	normalized := NormalizeBuildArgs(values)
	if len(normalized) == 0 {
		return ""
	}
	keys := make([]string, 0, len(normalized))
	for key := range normalized {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	ordered := make(map[string]string, len(normalized))
	for _, key := range keys {
		ordered[key] = normalized[key]
	}
	content, err := json.Marshal(ordered)
	if err != nil {
		return ""
	}
	return string(content)
}

func IsBuildArgKey(value string) bool {
	if value == "" || len(value) > 128 {
		return false
	}
	for index, char := range value {
		if index == 0 {
			if char == '_' || char >= 'A' && char <= 'Z' || char >= 'a' && char <= 'z' {
				continue
			}
			return false
		}
		if char == '_' || char >= 'A' && char <= 'Z' || char >= 'a' && char <= 'z' || char >= '0' && char <= '9' {
			continue
		}
		return false
	}
	return true
}
