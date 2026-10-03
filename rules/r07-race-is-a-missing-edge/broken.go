//go:build broken

package r07

// --8<-- [start:counter]

// Counter counts events reported by many goroutines.
type Counter struct {
	n int
}

// Inc adds one. n++ is a read and a write; nothing orders one goroutine's
// pair against another's, so two Incs can read the same n and one is lost.
func (c *Counter) Inc() { c.n++ }

// Value returns the count.
func (c *Counter) Value() int { return c.n }

// --8<-- [end:counter]
