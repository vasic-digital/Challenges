package challenge

// Ternary returns t if cond is true, f otherwise.
func Ternary(cond bool, t, f string) string {
	if cond {
		return t
	}
	return f
}
