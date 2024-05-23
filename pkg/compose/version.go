package compose

import "os"

func NodeVersion() (string, error) {
	// read env from NODE_VERSION
	// if NODE_VERSION is not set, return "beta"

	if env := os.Getenv("NODE_VERSION"); env != "" {
		return env, nil
	}

	return "beta", nil
}
