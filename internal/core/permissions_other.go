//go:build !windows

package core

import "os"

func secureDirectory(path string) error { return os.Chmod(path, 0700) }
