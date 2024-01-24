package variables

import (
	"log"
	"os"
	"strconv"
)

const (
	HTTP_PORT_DEFAULT = "8080"
	HTTP_PORT_NAME    = "HTTP_PORT"

	WEBRTC_ONE_TO_NAT_PUBLIC_IP_DEFAULT = ""
	WEBRTC_ONE_TO_NAT_PUBLIC_IP         = "WEBRTC_ONE_TO_NAT_PUBLIC_IP"

	WEBRTC_UDP_PORT_DEFAULT = "3478"
	WEBRTC_UDP_PORT         = "WEBRTC_UDP_PORT"
)

func ParseInt(value string) (int, error) {
	result, err := strconv.ParseUint(value, 10, 16)
	if err != nil {
		return 0, err
	}
	cast := int(result)
	return cast, nil
}

func Env(variableName, defaultValue string) string {
	if variable := os.Getenv(variableName); variable != "" {
		log.Printf("[%s]: %s", variableName, variable)
		return variable
	}
	log.Printf("[%s_DEFAULT]: %s", variableName, defaultValue)
	return defaultValue
}
