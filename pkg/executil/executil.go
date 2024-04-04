package executil

import (
	"os"
	"os/exec"
)

func MakeCmdRunner(commands ...string) func() error {
	return func() error {
		cmd := exec.Command(commands[0], commands[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
}
