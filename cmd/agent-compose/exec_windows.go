//go:build windows

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/coilyco/agent-compose/v2/internal/launch"
)

// execReal runs the target as a child and mirrors its exit code; Windows has
// no exec, so the sentinel still guards recursion through the environment.
func execReal(argv []string, extraEnv ...string) error {
	path, err := exec.LookPath(argv[0])
	if err != nil {
		return fmt.Errorf("launch target: %w", err)
	}
	cmd := exec.Command(path, argv[1:]...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.Env = append(os.Environ(), launch.EnvSentinel+"=1")
	cmd.Env = append(cmd.Env, extraEnv...)
	if err := cmd.Run(); err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			os.Exit(exit.ExitCode())
		}
		return err
	}
	os.Exit(0)
	return nil
}
