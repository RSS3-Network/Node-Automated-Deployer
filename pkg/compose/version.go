package compose

import "os"

// NodeVersion read env from NODE_VERSION
// if NODE_VERSION is not set, return "beta"
func NodeVersion() (string, error) {
	if env := os.Getenv("NODE_VERSION"); env != "" {
		return env, nil
	}

	return "beta", nil
}
