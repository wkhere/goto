package task

import (
	"fmt"
)

func warnf(format string, a ...any) {
	fmt.Printf("WARN "+format+"\n", a...)
}
