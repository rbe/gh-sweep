package project

import (
	"fmt"
	"strconv"
	"strings"
)

// sizeUnits are the units ParseSize reads, each 1024 times the one before.
var sizeUnits = []struct {
	suffix string
	bytes  int64
}{{"GB", 1 << 30}, {"MB", 1 << 20}, {"KB", 1 << 10}, {"B", 1}}

// ParseSize reads a size in bytes, such as 1048576, or a whole number with
// KB, MB, or GB, such as 50MB. The unit may be lower case. A negative size,
// a fraction, and another unit are an error.
func ParseSize(s string) (int64, error) {
	text := strings.ToUpper(strings.TrimSpace(s))
	unit := int64(1)
	for _, u := range sizeUnits {
		if strings.HasSuffix(text, u.suffix) {
			text, unit = strings.TrimSpace(strings.TrimSuffix(text, u.suffix)), u.bytes
			break
		}
	}
	n, err := strconv.ParseInt(text, 10, 64)
	if err != nil || n < 0 || n > (1<<62)/unit {
		return 0, fmt.Errorf("a size is a whole number of bytes, or a number with KB, MB, or GB, not %q", s)
	}
	return n * unit, nil
}
