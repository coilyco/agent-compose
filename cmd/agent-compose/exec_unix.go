//go:build !windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/coilyco/agent-compose/v2/internal/launch"
)

// execReal replaces this process with the target command; the sentinel in
// the child environment is the wrapper-recursion guard.
func execReal(argv []string, extraEnv ...string) error {
	path, err := exec.LookPath(argv[0])
	if err != nil {
		return fmt.Errorf("launch target: %w", err)
	}
	env := append(os.Environ(), launch.EnvSentinel+"=1")
	env = append(env, extraEnv...)
	return syscall.Exec(path, argv, env)
}
