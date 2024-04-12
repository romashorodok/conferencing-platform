package mcu

// #cgo pkg-config: media-server-mcu
// #include <main.h>
import "C"

func Setup() {
    C.MCU_setup()
}

func Version() {
    C.MCU_version()
}
