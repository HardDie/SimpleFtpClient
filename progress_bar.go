package main

import (
	"fmt"
	"strings"
)

func newProgressBar(width int64) func(int64, int64) string {
	width -= 2

	return func(progress, total int64) string {
		current := int64((float64(progress) / float64(total)) * float64(width))

		res := fmt.Sprintf("[%s", strings.Repeat("-", int(current)))
		if current != width {
			res += "\033[1;33m"
			if (current % 2) == 1 {
				res += "c"
			} else {
				res += "C"
			}
			res += "\033[0m"

			for i := int64(0); i <= (width - current - 2); i++ {
				if ((current + i) % 3) == 0 {
					res += "o"
				} else {
					res += " "
				}
			}
		}

		res += "]"

		return res
	}
}
