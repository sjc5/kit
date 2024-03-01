package envutil

import (
	"fmt"
	"os"
	"strconv"
)

func GetStr(key string, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return defaultValue
}

func GetInt(key string, defaultValue int) int {
	strValue := GetStr(key, strconv.Itoa(defaultValue))
	value, err := strconv.Atoi(strValue)
	if err == nil {
		return value
	}
	fmt.Printf("error parsing %s, defaulting to %d. error: %s", key, defaultValue, err)
	return defaultValue
}
