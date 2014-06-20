package engineio

import (
	"fmt"
	"os"
)

var eioDebug bool = len(os.Getenv("EIO_DEBUG")) > 0

func debug(msg ...interface{}) {
	if eioDebug {
		fmt.Print("[\x1b[33;1mEIO DEBUG\x1b[0m] ")
		fmt.Println(msg...)
	}
}
