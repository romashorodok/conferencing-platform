package cpppipelines

/*
#include "pipelines/pipelines.h"
#include "pipelines/rtpvp8/rtpvp8.h"
*/
import "C"

func GstreamerMainLoopSetup() {
	C.setup()
	C.print_version()
}
