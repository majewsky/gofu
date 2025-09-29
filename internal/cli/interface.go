// SPDX-FileCopyrightText: 2017 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: GPL-3.0-only

package cli

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"golang.org/x/term"
)

// Interface wraps access to the CLI, including input, output and subprocesses.
var Interface *Implementation

func init() {
	SetupInterface(os.Stdin, os.Stdout, os.Stderr, DefaultCommandRunner)
}

// SetupInterface prepares the Interface instance with nonstandard file streams
// or a nonstandard CommandRunner. This is only required for unit tests.
func SetupInterface(stdin io.Reader, stdout, stderr io.Writer, commandRunner CommandRunner) {
	Interface = &Implementation{
		stdin:         stdin,
		stdout:        stdout,
		stderr:        stderr,
		stdinBuf:      bufio.NewReader(stdin),
		commandRunner: commandRunner,
	}

	if stdinFile, ok := stdin.(*os.File); ok && term.IsTerminal(int(stdinFile.Fd())) {
		Interface.tui = &terminalTUI{Interface}
	} else {
		Interface.tui = &pipeTUI{Interface}
	}
}

// Implementation wraps access to the CLI, including input, output and subprocesses.
type Implementation struct {
	stdin         io.Reader
	stdout        io.Writer
	stderr        io.Writer
	stdinBuf      *bufio.Reader
	tui           TUI
	commandRunner CommandRunner
	//If this flag is set, only ShowResult() will write into stdout; everything
	//else that usually goes to stdout goes to stderr instead.
	//
	//This is useful when gofu is expected to output a certain value to stdout
	//which is used by the next program in the pipe, and additional output from
	//subprocesses could confuse the stdout handler.
	StdoutProtected bool
}

// TUI provides the interactive parts of the cli.Implementation, so that these can be
// easily swapped out for mock implementations in unit tests.
type TUI interface {
	//ReadLine reads a line from stdin (if tty: uses canonical mode).
	ReadLine(prompt string) (string, error)
	//Confirm displays a yes/no question and returns whether the user answered "yes".
	Confirm(question string) (bool, error)
	//Query displays a question and a set of answers and allows the user to select
	//one of the answers. Returns the Return attribute of the selected Choice.
	Query(prompt string, choices ...Choice) (string, error)
	//Print writes the given string (potentially including ANSI escape codes) to
	//the given writer. At this point, it can be decided whether to strip out the
	//ANSI escape codes.
	Print(w io.Writer, msg string)
}

func (i *Implementation) safeStdout() io.Writer {
	if i.StdoutProtected {
		return i.stderr
	}
	return i.stdout
}

////////////////////////////////////////////////////////////////////////////////
// input

// ReadLine reads a line from stdin (if tty: uses canonical mode).
func (i *Implementation) ReadLine(prompt string) (string, error) {
	return i.tui.ReadLine(prompt)
}

// Confirm displays a yes/no question and returns whether the user answered "yes".
func (i *Implementation) Confirm(question string) (bool, error) {
	return i.tui.Confirm(question)
}

// Query displays a question and a set of answers and allows the user to select
// one of the answers. Returns the Return attribute of the selected Choice.
func (i *Implementation) Query(prompt string, choices ...Choice) (string, error) {
	return i.tui.Query(prompt, choices...)
}

////////////////////////////////////////////////////////////////////////////////
// subprocesses

// Run executes the given command on the same stdout and stderr.
func (i *Implementation) Run(c Command) error {
	return i.commandRunner(c, nil, i.safeStdout(), i.stderr)
}

// CaptureStdout executes the given command on the same stderr and captures its stdout.
func (i *Implementation) CaptureStdout(c Command) (string, error) {
	var buf bytes.Buffer
	err := i.commandRunner(c, nil, &buf, i.stderr)
	return buf.String(), err
}

////////////////////////////////////////////////////////////////////////////////
// output

// ShowResult displays the result of a computation on stdout.
func (i *Implementation) ShowResult(str string) {
	str = strings.TrimSpace(str) + "\n"
	i.stdout.Write([]byte(str))
}

// ShowResultsSorted calls ShowResult() on each of the results after sorting them.
func (i *Implementation) ShowResultsSorted(strs []string) {
	sort.Strings(strs)
	for _, str := range strs {
		i.ShowResult(str)
	}
}

// ShowProgress displays a progress message on stderr.
func (i *Implementation) ShowProgress(str string) {
	i.tui.Print(i.stderr, fmt.Sprintf("\x1B[0;1;36m>>\x1B[0;36m %s\x1B[0m\n", strings.TrimSpace(str)))
}

// ShowWarning displays a warning message on stderr.
func (i *Implementation) ShowWarning(str string) {
	i.tui.Print(i.stderr, fmt.Sprintf("\x1B[0;1;33m!!\x1B[0;33m %s\x1B[0m\n", strings.TrimSpace(str)))
}

// ShowError displays an error message on stderr.
func (i *Implementation) ShowError(str string) {
	i.tui.Print(i.stderr, fmt.Sprintf("\x1B[0;1;31m!!\x1B[0;31m %s\x1B[0m\n", strings.TrimSpace(str)))
}

// ShowUsage displays a usage synopsis on stderr.
func (i *Implementation) ShowUsage(str string) {
	str = strings.TrimSpace(str) + "\n"
	i.stderr.Write([]byte(str))
}
