// Package r01 is rule R01: start sequential; concurrency is a cost you must
// measure.
//
// broken.go and fixed.go both define SumSquares; the build tag "broken"
// selects which one is compiled. rule_test.go checks both are correct;
// bench_test.go measures what each costs.
package r01
