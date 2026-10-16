package task

import (
	"fmt"
	"math"
)

func sizeFmt(size int64) string {
	units := []string{"", "Ki", "Mi", "Gi"}
	var x = float64(size)
	var j int
	for {
		y := x / 1024
		if math.Abs(y) < 1 || j >= len(units) {
			break
		}
		x = y
		j++
	}
	var prec = 3
	if _, f := math.Modf(x); f == 0.0 {
		prec = 0
	}
	return fmt.Sprintf("%.*f%s", prec, x, units[j])
}
