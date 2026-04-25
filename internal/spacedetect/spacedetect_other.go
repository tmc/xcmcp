//go:build !darwin

package spacedetect

import "errors"

// ErrSkyLightUnavailable is always returned on non-darwin platforms.
var ErrSkyLightUnavailable = errors.New("spacedetect: SkyLight unavailable")

// IsOffSpace always returns ErrSkyLightUnavailable on non-darwin platforms.
func IsOffSpace(windowID uint32) (bool, error) {
	return false, ErrSkyLightUnavailable
}
