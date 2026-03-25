package helpers

import (
	"regexp"
	"strings"

	"github.com/grokify/mogo/encoding/base36"
)

var sanitiseDbValueRe = regexp.MustCompile(`[^a-zA-Z0-9_]+`)

func SliceContainsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func RemoveStringFromSlice(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}

func SanitiseDbValue(value string) string {
	return string(sanitiseDbValueRe.ReplaceAll([]byte(strings.ReplaceAll(value, "-", "_")), []byte("")))
}

func SanitiseAndShortenDbValue(value string, maxLength int) string {
	value = SanitiseDbValue(value)
	if len(value) > maxLength {
		value = base36.Md5Base36(value)[0:10]
	}
	return value
}

func ShortenHumanReadableValue(value string, maxLength int) string {
	if len(value) > maxLength {
		if maxLength <= 11 {
			return base36.Md5Base36(value)[0:10]
		}
		value = value[0:maxLength-11] + "-" + base36.Md5Base36(value)[0:10]
	}
	return value
}

func MakeObjectName(baseName string, suffixes ...string) string {
	suffix := ""
	for _, s := range suffixes {
		suffix = suffix + "-" + s
	}
	return ShortenHumanReadableValue(baseName, 63-len(suffix)) + suffix
}

func GetKeysFromStringBoolMap(v map[string]bool) []string {
	result := make([]string, 0, len(v))
	for i := range v {
		result = append(result, i)
	}
	return result
}
