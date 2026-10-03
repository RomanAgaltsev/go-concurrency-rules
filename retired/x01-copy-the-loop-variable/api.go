// Package x01 is retired rule X01: copy the loop variable before capturing it.
//
// The two variants are the same code. broken.go carries `//go:build broken &&
// go1.21`, which sets that file's language version to go1.21 — before
// per-iteration loop variables — so the old bug comes back for this one file.
package x01
