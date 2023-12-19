package variables

import (
	"log"
	"os"
)

const (
	HTTP_PORT_DEFAULT = "8080"
	HTTP_PORT_NAME    = "HTTP_PORT"
)

func Env(variableName, defaultValue string) string {
	if variable := os.Getenv(variableName); variable != "" {
		log.Printf("[%s]: %s", variableName, variable)
		return variable
	}
	log.Printf("[%s_DEFAULT]: %s", variableName, defaultValue)
	return defaultValue
}
