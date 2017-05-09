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
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"golang.org/x/crypto/ssh/terminal"
)

//NewInterface creates an Interface instance.
func NewInterface(stdin, stdout, stderr *os.File) *Interface {
	i := &Interface{
		stdin:    stdin,
		stdout:   stdout,
		stderr:   stderr,
		stdinBuf: bufio.NewReader(stdin),
	}

	if terminal.IsTerminal(int(stdin.Fd())) {
		i.tui = &terminalTUI{i}
	} else {
		i.tui = &pipeTUI{i}
	}

	return i
}

//Interface wraps access to the CLI, including input, output and subprocesses.
type Interface struct {
	//TODO: flag isStdinTerminal that disables color output and swaps out the TUI instance
	stdin    io.Reader
	stdout   io.Writer
	stderr   io.Writer
	stdinBuf *bufio.Reader
	tui      TUI
	//If this flag is set, only ShowResult() will write into stdout; everything
	//else that usually goes to stdout goes to stderr instead.
	//
	//This is useful when gofu is expected to output a certain value to stdout
	//which is used by the next program in the pipe, and additional output from
	//subprocesses could confuse the stdout handler.
	StdoutProtected bool
}

//TUI provides the interactive parts of the cli.Interface, so that these can be
//easily swapped out for mock implementations in unit tests.
type TUI interface {
	//ReadLine reads a line from stdin (if tty: uses canonical mode).
	ReadLine(prompt string) (string, error)
	//Confirm displays a yes/no question and returns whether the user answered "yes".
	Confirm(question string) (bool, error)
	//Query displays a question and a set of answers and allows the user to select
	//one of the answers. Returns the Return attribute of the selected Choice.
	Query(prompt string, choices ...Choice) (string, error)
}

func (i *Interface) safeStdout() io.Writer {
	if i.StdoutProtected {
		return i.stderr
	}
	return i.stdout
}

////////////////////////////////////////////////////////////////////////////////
// input

//ReadLine reads a line from stdin (if tty: uses canonical mode).
func (i *Interface) ReadLine(prompt string) (string, error) {
	return i.tui.ReadLine(prompt)
}

//Confirm displays a yes/no question and returns whether the user answered "yes".
func (i *Interface) Confirm(question string) (bool, error) {
	return i.tui.Confirm(question)
}

//Query displays a question and a set of answers and allows the user to select
//one of the answers. Returns the Return attribute of the selected Choice.
func (i *Interface) Query(prompt string, choices ...Choice) (string, error) {
	return i.tui.Query(prompt, choices...)
}

////////////////////////////////////////////////////////////////////////////////
// subprocesses

//Run executes the given command on the same stdout and stderr.
func (i *Interface) Run(c Command) error {
	return c.run(i.safeStdout(), i.stderr)
}

//CaptureStdout executes the given command on the same stderr and captures its stdout.
func (i *Interface) CaptureStdout(c Command) (string, error) {
	var buf bytes.Buffer
	err := c.run(&buf, i.stderr)
	return string(buf.Bytes()), err
}

////////////////////////////////////////////////////////////////////////////////
// output

//ShowResult displays the result of a computation on stdout.
func (i *Interface) ShowResult(str string) {
	str = strings.TrimSpace(str) + "\n"
	i.stdout.Write([]byte(str))
}

//ShowResultsSorted calls ShowResult() on each of the results after sorting them.
func (i *Interface) ShowResultsSorted(strs []string) {
	sort.Strings(strs)
	for _, str := range strs {
		i.ShowResult(str)
	}
}

//ShowProgress displays a progress message on stderr.
func (i *Interface) ShowProgress(str string) {
	fmt.Fprintf(i.stderr, "\x1B[0;1;36m>>\x1B[0;36m %s\x1B[0m", strings.TrimSpace(str))
}

//ShowWarning displays a warning message on stderr.
func (i *Interface) ShowWarning(str string) {
	fmt.Fprintf(i.stderr, "\x1B[0;1;33m!!\x1B[0;36m %s\x1B[0m", strings.TrimSpace(str))
}

//ShowError displays an error message on stderr.
func (i *Interface) ShowError(str string) {
	fmt.Fprintf(i.stderr, "\x1B[0;1;31m!!\x1B[0;36m %s\x1B[0m", strings.TrimSpace(str))
}

//ShowUsage displays a usage synopsis on stderr.
func (i *Interface) ShowUsage(str string) {
	str = strings.TrimSpace(str) + "\n"
	i.stderr.Write([]byte(str))
}
