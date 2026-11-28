package project

import "fmt"

// Skipped is a repository that a report leaves out, because gh gave an
// error for it. Reason is the first line of that error.
type Skipped struct {
	NWO    string
	Reason string
}

// Line is the report line of a skipped repository.
func (s Skipped) Line() string {
	return fmt.Sprintf("%s: skipped, %s", s.NWO, s.Reason)
}
