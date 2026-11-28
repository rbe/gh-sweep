package sweep

import (
	"fmt"
	"strconv"

	"github.com/rbe/gh-sweep/internal/project"
)

// FilterLabels name each filter for a reader of the action screen.
var FilterLabels = map[string]string{
	"older-than":   "older than, in days",
	"branch":       "branch",
	"status":       "run status or conclusion, such as failure",
	"key-prefix":   "cache key starts with",
	"keep":         "keep the newest",
	"untagged":     "only untagged versions: yes or no",
	"package":      "package name",
	"package-type": "package type",
}

// SetFilter parses the value of one filter into f. The command line and
// the action screen use it, so both refuse the same values.
func SetFilter(f *project.Filter, name, value string) error {
	switch name {
	case "older-than":
		days, err := strconv.Atoi(value)
		if err != nil || days < 1 {
			return fmt.Errorf("give a number of days of 1 or more")
		}
		f.OlderThan = days
	case "keep":
		n, err := strconv.Atoi(value)
		if err != nil || n < 0 {
			return fmt.Errorf("give a number of 0 or more")
		}
		f.Keep = n
	case "branch":
		f.Branch = value
	case "status":
		f.Status = value
	case "key-prefix":
		f.KeyPrefix = value
	case "untagged":
		switch value {
		case "yes":
			f.Untagged = true
		case "no":
			f.Untagged = false
		default:
			return fmt.Errorf("give yes or no")
		}
	case "package":
		f.Package = value
	case "package-type":
		f.PackageType = value
	default:
		return fmt.Errorf("--%s is not a filter", name)
	}
	return nil
}
