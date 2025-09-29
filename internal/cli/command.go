// SPDX-FileCopyrightText: 2017 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: GPL-3.0-only

package cli

import (
	"fmt"
	"io"
	"os/exec"
	"strings"
)

//Command describes a command that can be run using the methods in the
//Implementation interface.
type Command struct {
	Program []string
	WorkDir string
}

type commandError struct {
	Cmd Command
	Err error
}

func (e commandError) Error() string {
	cmdline := strings.Join(e.Cmd.Program, " ")
	if e.Cmd.WorkDir == "" {
		return fmt.Sprintf("exec `%s`: %s",
			cmdline, e.Err.Error(),
		)
	}
	return fmt.Sprintf("exec `%s` in %s: %s",
		cmdline, e.Cmd.WorkDir, e.Err.Error(),
	)
}

//CommandRunner is a function that can execute commands given to it.
//This interface is only useful for unit tests; the default CommandRunner
//suffices for all regular operation.
type CommandRunner func(c Command, stdin io.Reader, stdout, stderr io.Writer) error

//DefaultCommandRunner is a CommandRunner that actually executes the command.
func DefaultCommandRunner(c Command, stdin io.Reader, stdout, stderr io.Writer) error {
	cmd := exec.Command(c.Program[0], c.Program[1:]...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Dir = c.WorkDir

	err := cmd.Run()
	if err != nil {
		err = commandError{c, err}
	}
	return err
}
