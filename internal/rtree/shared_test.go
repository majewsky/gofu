// SPDX-FileCopyrightText: 2017 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: GPL-3.0-only

package rtree

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	yaml "go.yaml.in/yaml/v3"

	"github.com/majewsky/gofu/internal/cli"
)

// Path to a directory where tests can put their index files.
var indexTmpDir = filepath.Join(os.TempDir(), fmt.Sprintf("rtree-test-%d", os.Getpid()))

func TestMain(m *testing.M) {
	//make sure that test does not accidentally access user's actual rtree or index
	os.Setenv("HOME", "")
	os.Setenv("GOPATH", "")
	//setup test configuration
	RootPath = "/unittest/gopath/src"
	RemoteAliases = []*RemoteAlias{
		{Alias: "gh:", Replacement: "https://github.com/"},
		{Alias: "my/", Replacement: "git@git.example.com:"},
	}

	exitCode := m.Run()

	//shared teardown
	os.RemoveAll(indexTmpDir)

	os.Exit(exitCode)
}

// Test describes a call to Main(), the environment that's given to it, and the
// assertions that are checked after the call returns.
type Test struct {
	Args            []string
	Input           string
	Index           Index
	ExpectFailure   bool
	ExpectOutput    string
	ExpectError     string
	ExpectIndex     *Index //if nil, .Index will be used instead
	ExpectExecution []RecordedCommand
}

func (test Test) Run(t *testing.T) {
	//write index file, if any
	IndexPath = filepath.Join(indexTmpDir, t.Name()+".yaml")
	if test.Index.Repos != nil {
		err := test.Index.Write()
		if err != nil {
			t.Fatalf("%s: cannot write index to %s: %s", t.Name(), IndexPath, err.Error())
		}
	}

	//setup cli.Interface for test
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cs := CommandSimulator{Cmd: test.ExpectExecution}
	cli.SetupInterface(bytes.NewReader([]byte(test.Input)), &stdout, &stderr, cs.Next)

	//check exit code
	exitCode := Exec(test.Args)
	switch {
	case exitCode == 0 && test.ExpectFailure:
		t.Errorf("%s: expected failure, but returned success", t.Name())
	case exitCode != 0 && !test.ExpectFailure:
		t.Errorf("%s: expected success, but returned failure", t.Name())
	}

	//check output
	output := stdout.String()
	if output != test.ExpectOutput {
		t.Errorf("%s: expected stdout %#v, but got %#v", t.Name(), test.ExpectOutput, output)
	}
	output = stderr.String()
	if output != test.ExpectError {
		t.Errorf("%s: expected stderr %#v, but got %#v", t.Name(), test.ExpectError, output)
	}

	//check index
	idx := &test.Index
	if test.ExpectIndex != nil {
		idx = test.ExpectIndex
	}
	expectedIdxStr, err := yaml.Marshal(idx)
	if err != nil {
		t.Fatal(err.Error())
	}
	actualIdxStr, err := os.ReadFile(IndexPath)
	if err != nil {
		t.Fatalf("%s: could not read index from %s: %s", t.Name(), IndexPath, err.Error())
	}
	if string(expectedIdxStr) != string(actualIdxStr) {
		t.Errorf("%s: index does not match expectation after test; diff follows", t.Name())
		cmd := exec.Command("diff", "-u", "-", IndexPath)
		cmd.Stdin = bytes.NewReader([]byte(expectedIdxStr))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			t.Fatal(err.Error())
		}
	}
}

type RecordedCommand struct {
	Cmd    cli.Command
	Stdout string
	Stderr string
	Fails  bool
}

// Recorded is a shortcut function for initializing a []RecordedCommand. It
// splits each line on whitespace to obtain the command line of that command,
// and recognizes a leading "@/some/path" to set the workdir.
//
// This function can only be used for RecordedCommands without output that do not fail.
func Recorded(lines ...string) (cs []RecordedCommand) {
	cs = make([]RecordedCommand, len(lines))
	for idx, line := range lines {
		cmdline := strings.Fields(line)
		if workDir, ok := strings.CutPrefix(cmdline[0], "@"); ok {
			cs[idx].Cmd.WorkDir = workDir
			cmdline = cmdline[1:]
		}
		cs[idx].Cmd.Program = cmdline
	}
	return
}

// CommandSimulator implements the cli.CommandRunner interface (via its Next
// method). When a cli.Command is given to Next(), it is matched with the next
// command in the .Cmd list, and the result from that RecordedCommand is
// returned. If the given Command is different from the one expected (or if the
// .Cmd list has been exhausted), an error is returned.
type CommandSimulator struct {
	Cmd []RecordedCommand
	idx int
}

func (s *CommandSimulator) Next(c cli.Command, stdin io.Reader, stdout, stderr io.Writer) error {
	//take next RecordedCommand from list
	if s.idx >= len(s.Cmd) {
		return errors.New("got command to execute, but recorded commands have been exhausted")
	}
	sc := s.Cmd[s.idx]
	s.idx++

	//check if the given Command matches the expectation
	if !areStringListsEqual(sc.Cmd.Program, c.Program) {
		return fmt.Errorf("expected command %#v, but got %#v",
			strings.Join(sc.Cmd.Program, " "), strings.Join(c.Program, " "),
		)
	}
	if sc.Cmd.WorkDir != c.WorkDir {
		return fmt.Errorf("expected command workdir %s, but got %s", sc.Cmd.WorkDir, c.WorkDir)
	}

	stdout.Write([]byte(sc.Stdout))
	stderr.Write([]byte(sc.Stderr))
	if sc.Fails {
		return fmt.Errorf("command %#v has failed", strings.Join(c.Program, " "))
	}
	return nil
}

func areStringListsEqual(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for idx := range a {
		if a[idx] != b[idx] {
			return false
		}
	}
	return true
}
