package helpers

import (
	"github.com/grokify/mogo/encoding/base36"
	"regexp"
	"strings"
)

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
	re := regexp.MustCompile(`[^a-zA-Z0-9_]+`)
	result := string(re.ReplaceAll([]byte(strings.Replace(value, "-", "_", -1)), []byte("")))

	return result
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
	result := make([]string, len(v))
	for i := range v {
		result = append(result, i)
	}
	return result
}
