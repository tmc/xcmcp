// Package spacedetect reports whether a CGWindowID lives on a macOS Space
// other than the user's currently active Space.
//
// macOS does not expose Space membership through any public API. The detector
// here resolves three private SkyLight symbols at runtime
// (SLSMainConnectionID, SLSGetActiveSpace, SLSCopySpacesForWindows) and
// reports off-Space residency as plain metadata. Cross-Space migration is
// deliberately out of scope: it requires a private WindowServer entitlement
// that Apple does not grant outside its own processes.
//
// IsOffSpace returns errors that wrap ErrSkyLightUnavailable when the
// framework or any of the three symbols cannot be resolved. Callers should
// branch via errors.Is(err, ErrSkyLightUnavailable) — bare == comparison
// will silently miss every real error, since errors are joined with
// fmt.Errorf("%w: ...", ErrSkyLightUnavailable, cause).
//
// This package is the smallest possible adoption of the SkyLight dlsym
// pattern; landing it derisks the binding workflow before the larger
// SLEventPostToPid and AXObserverAddNotificationAndCheckRemote adoptions.
package spacedetect
