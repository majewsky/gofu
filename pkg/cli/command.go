/*******************************************************************************
*
* Copyright 2017 Stefan Majewsky <majewsky@gmx.net>
*
* This program is free software: you can redistribute it and/or modify it under
* the terms of the GNU General Public License as published by the Free Software
* Foundation, either version 3 of the License, or (at your option) any later
* version.
*
* This program is distributed in the hope that it will be useful, but WITHOUT ANY
* WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR
* A PARTICULAR PURPOSE. See the GNU General Public License for more details.
*
* You should have received a copy of the GNU General Public License along with
* this program. If not, see <http://www.gnu.org/licenses/>.
*
*******************************************************************************/

package cli

import (
	"fmt"
	"io"
	"os/exec"
	"strings"
)

//Command describes a command that can be run using the methods in the
//Interface interface.
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

func (c Command) run(stdout, stderr io.Writer) error {
	cmd := exec.Command(c.Program[0], c.Program[1:]...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Dir = c.WorkDir

	err := cmd.Run()
	if err != nil {
		err = commandError{c, err}
	}
	return err
}
