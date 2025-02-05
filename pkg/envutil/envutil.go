package envutil

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
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
	return defaultValue
}

func GetBool(key string, defaultValue bool) bool {
	strValue := GetStr(key, strconv.FormatBool(defaultValue))
	value, err := strconv.ParseBool(strValue)
	if err == nil {
		return value
	}
	return defaultValue
}

type Env interface {
	GetIsDev() bool
	GetPort() int
}

const (
	ModeKey       = "MODE"
	ModeValueProd = "production"
	ModeValueDev  = "development"
)

type Base struct {
	Mode   string
	IsDev  bool
	IsProd bool
	Port   int
}

func (e *Base) GetIsDev() bool { return e.IsDev }
func (e *Base) GetPort() int   { return e.Port }

func MustInitBase(fallbackGetPortFunc func() int) Base {
	if err := godotenv.Load(); err != nil {
		panic(fmt.Sprintf("envutil: failed to load .env file: %v", err))
	}

	base := Base{}

	base.Mode = GetStr(ModeKey, ModeValueProd)

	if base.Mode != ModeValueDev && base.Mode != ModeValueProd {
		panic(fmt.Sprintf("envutil: invalid MODE value: %s", base.Mode))
	}

	base.IsDev = base.Mode == ModeValueDev
	base.IsProd = base.Mode == ModeValueProd

	fallbackPort := 8080
	if fallbackGetPortFunc != nil {
		fallbackPort = fallbackGetPortFunc()
	}

	base.Port = GetInt("PORT", fallbackPort)

	return base
}

// SetDevMode sets the MODE environment variable to "development".
func SetDevMode() error {
	return os.Setenv(ModeKey, ModeValueDev)
}
