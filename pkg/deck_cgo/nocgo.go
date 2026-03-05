// Package deck_cgo — build tag stub when CGo is disabled.
//
// When CGo is not available (e.g. cross-compiling with CGO_ENABLED=0),
// this file provides a compile-error with a clear message instead of a
// cryptic linker failure.

//go:build !cgo

package deck_cgo

import _ "unsafe" // required for go:linkname in some toolchains

func init() {
	panic("deck_cgo: CGo is required. Recompile with CGO_ENABLED=1 and provide the sekai_deck_recommend_c library for your platform.")
}
