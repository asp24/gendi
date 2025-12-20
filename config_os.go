package di

// This indirection allows tests to replace file reads without touching os.
// It is kept minimal to avoid bringing in extra dependencies.
func init() {
	osReadFile = readFileDefault
}

