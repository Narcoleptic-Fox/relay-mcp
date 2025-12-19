package utils

// Truncate truncates a string to the specified maximum length
// and appends "..." if truncation occurred.
func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// IntPtr returns a pointer to an int value
func IntPtr(v int) *int {
	return &v
}

// FloatPtr returns a pointer to a float64 value
func FloatPtr(v float64) *float64 {
	return &v
}
