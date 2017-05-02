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
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/majewsky/gofu/pkg/util"
)

//RootPath is the directory below which all repositories are located. Its value
//is $GOPATH/src to match the repository layout created by `go get`.
var RootPath string

func init() {
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		util.FatalIfError(errors.New("$GOPATH is not set (rtree needs the GOPATH variable to know where to look for and place repos)"))
	}
	RootPath = filepath.Join(gopath, "src")
}

//Repo describes the entry for a repository in the index file.
type Repo struct {
	//CheckoutPath shall be relative to the RootPath.
	CheckoutPath string `yaml:"path"`
	//Remotes maps remote names (as noted in the .git/config of the repo) to
	//remote URLs (as they appear in the .git/config of the repo, i.e. possibly
	//abbreviated).
	Remotes []Remote `yaml:"remotes"`
}

//Remote describes a remote that is configured in a Repo.
type Remote struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url"`
}

//AbsolutePath returns the absolute CheckoutPath of this repo.
func (r Repo) AbsolutePath() string {
	return filepath.Join(RootPath, r.CheckoutPath)
}

//NewRepoFromAbsolutePath initializes a Repo instance by scanning the existing
//checkout at the given path.
func NewRepoFromAbsolutePath(path string) (repo Repo, err error) {
	repo.CheckoutPath, err = filepath.Rel(RootPath, path)
	if err != nil {
		return
	}

	//list remotes
	cmd := exec.Command("git", "-C", path, "config", "-l")
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return repo, fmt.Errorf("exec `git config -l` in %s: %s", path, err.Error())
	}

	for _, line := range strings.Split(string(buf.Bytes()), "\n") {
		match := remoteConfigRx.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		repo.Remotes = append(repo.Remotes, Remote{
			Name: match[1],
			URL:  match[2],
		})
	}
	return
}

var remoteConfigRx = regexp.MustCompile(`remote\.([^=]+)\.url=(.+)`)

//ForeachPhysicalRepo walks over the repository tree, executing the action
//function once for every repo encountered (but *not* for repos contained
//within other repos, e.g. submodules).
func ForeachPhysicalRepo(action func(repo Repo) error) error {
	return filepath.Walk(RootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		//look for repos, i.e. directories containing a .git directory
		if !info.IsDir() {
			return nil
		}
		_, err = os.Stat(filepath.Join(path, ".git"))
		if err != nil {
			return nil
		}

		//appears to be a repo
		repo, err := NewRepoFromAbsolutePath(path)
		if err == nil {
			err = action(repo)
		}
		if err != nil {
			return err
		}
		//do not traverse further down into submodules etc.
		return filepath.SkipDir
	})
}

//Checkout creates the repo in the given path with the given remotes. The
//working copy will only be initialized if there is an "origin" remote.
func (r Repo) Checkout() error {
	//check if we have an "origin" remote to clone from
	var originURL string
	for _, remote := range r.Remotes {
		if remote.Name == "origin" {
			originURL = remote.URL
			break
		}
	}

	if originURL == "" {
		cmd := exec.Command("git", "init", r.AbsolutePath())
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr, "warning: will not checkout anything since there is no remote named \"origin\"")
	} else {
		cmd := exec.Command("git", "clone", originURL, r.AbsolutePath())
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			return err
		}
	}

	remotesAdded := false
	for _, remote := range r.Remotes {
		if remote.Name != "origin" {
			cmd := exec.Command("git", "-C", r.AbsolutePath(), "remote", "add", remote.Name, remote.URL)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err := cmd.Run()
			if err != nil {
				return err
			}
			remotesAdded = true
		}
	}
	if remotesAdded {
		cmd := exec.Command("git", "-C", r.AbsolutePath(), "remote", "update")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	return nil
}

//InteractiveExec implements the meat of the `rtree exec` command. It returns
//true iff the command exited successfully.
func (r Repo) InteractiveExec(command string, args ...string) (ok bool) {
	fmt.Fprintf(os.Stdout, "\x1B[1;36m>> \x1B[0;36m%s\x1B[0m\n", r.AbsolutePath())
	cmd := exec.Command(command, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = r.AbsolutePath()
	err := cmd.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "\x1B[1;31m!! \x1B[0;31m%s\x1B[0m\n", err.Error())
		return false
	}
	return true
}
