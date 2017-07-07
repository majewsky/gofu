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

package rtree

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/majewsky/gofu/pkg/cli"
	yaml "gopkg.in/yaml.v2"
)

//Path to a directory where tests can put their index files.
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

//Test describes a call to Main(), the environment that's given to it, and the
//assertions that are checked after the call returns.
type Test struct {
	Args          []string
	Input         string
	Index         Index
	ExpectFailure bool
	ExpectOutput  string
	ExpectError   string
	ExpectIndex   *Index //if nil, .Index will be used instead
}

func (test Test) Run(t *testing.T, testName string) {
	//write index file, if any
	IndexPath = filepath.Join(indexTmpDir, testName+".yaml")
	if test.Index.Repos != nil {
		err := test.Index.Write()
		if err != nil {
			t.Fatalf("%s: cannot write index to %s: %s", testName, IndexPath, err.Error())
		}
	}

	//setup cli.Interface for test
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cli.SetupInterface(bytes.NewReader([]byte(test.Input)), &stdout, &stderr, nil)

	//check exit code
	exitCode := Exec(test.Args)
	switch {
	case exitCode == 0 && test.ExpectFailure:
		t.Errorf("%s: expected failure, but returned success", testName)
	case exitCode != 0 && !test.ExpectFailure:
		t.Errorf("%s: expected success, but returned failure", testName)
	}

	//check output
	output := string(stdout.Bytes())
	if output != test.ExpectOutput {
		t.Errorf("%s: expected stdout %#v, but got %#v", testName, test.ExpectOutput, output)
	}
	output = string(stderr.Bytes())
	if output != test.ExpectError {
		t.Errorf("%s: expected stderr %#v, but got %#v", testName, test.ExpectError, output)
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
	actualIdxStr, err := ioutil.ReadFile(IndexPath)
	if err != nil {
		t.Fatalf("%s: could not read index from %s: %s", testName, IndexPath, err.Error())
	}
	if string(expectedIdxStr) != string(actualIdxStr) {
		t.Errorf("%s: index does not match expectation after test; diff follows", testName)
		cmd := exec.Command("diff", "-u", "-", IndexPath)
		cmd.Stdin = bytes.NewReader([]byte(expectedIdxStr))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Wait()
		if err != nil {
			t.Fatal(err.Error())
		}
	}
}
