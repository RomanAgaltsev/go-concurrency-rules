// Package r12 is rule R12: never copy a sync value.
//
// broken.go and fixed.go both define Stats; the build tag "broken" selects
// which one is compiled. The proof is go vet's copylocks analyzer: it must
// report the broken variant and stay silent on the fixed one.
package r12
