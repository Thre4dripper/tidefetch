//go:build !darwin && !linux

package ui

import "errors"

// diskUsage is unsupported on this platform.
func diskUsage(path string) (free, total int64, err error) {
	return 0, 0, errors.New("disk usage unsupported on this platform")
}
