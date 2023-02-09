package utils

const (
	Byte     uint64 = 1
	KibiByte        = 1024 * Byte
	MebiByte        = 1024 * KibiByte
	GibiByte        = 1024 * MebiByte
	TebiByte        = 1024 * GibiByte
	PebiByte        = 1024 * TebiByte
)

// StringsContain reports whether substr is within s.
func StringsContain(s []string, substr string) bool {
	for _, a := range s {
		if a == substr {
			return true
		}
	}
	return false
}
