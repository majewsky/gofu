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
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/majewsky/gofu/internal/cli"
)

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
	Name string    `yaml:"name"`
	URL  RemoteURL `yaml:"url"`
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
	out, err := cli.Interface.CaptureStdout(cli.Command{
		Program: []string{"git", "config", "-l"},
		WorkDir: path,
	})
	if err != nil {
		return
	}

	for _, line := range strings.Split(out, "\n") {
		match := remoteConfigRx.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		repo.Remotes = append(repo.Remotes, Remote{
			Name: match[1],
			URL:  ParseRemoteURL(match[2]),
		})
	}
	return
}

//NewRepoFromRemoteURL initializes a Repo instance for checking out a remote
//for the first time. The checkout does not happen until Checkout() is called.
func NewRepoFromRemoteURL(remoteURL RemoteURL) (Repo, error) {
	checkoutPath, err := remoteURL.CheckoutPath()
	return Repo{
		CheckoutPath: checkoutPath,
		Remotes: []Remote{
			{
				Name: "origin",
				URL:  remoteURL,
			},
		},
	}, err
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
	var originURL RemoteURL
	for _, remote := range r.Remotes {
		if remote.Name == "origin" {
			originURL = remote.URL
			break
		}
	}

	if originURL == "" {
		err := cli.Interface.Run(cli.Command{
			Program: []string{"git", "init", r.AbsolutePath()},
		})
		if err != nil {
			return err
		}
		cli.Interface.ShowWarning(`will not checkout anything since there is no remote named "origin"`)
	} else {
		err := cli.Interface.Run(cli.Command{
			Program: []string{"git", "clone", originURL.CompactURL(), r.AbsolutePath()},
		})
		if err != nil {
			return err
		}
	}

	remotesAdded := false
	for _, remote := range r.Remotes {
		if remote.Name != "origin" {
			err := cli.Interface.Run(cli.Command{
				Program: []string{"git", "remote", "add", remote.Name, remote.URL.CompactURL()},
				WorkDir: r.AbsolutePath(),
			})
			if err != nil {
				return err
			}
			remotesAdded = true
		}
	}
	if remotesAdded {
		return cli.Interface.Run(cli.Command{
			Program: []string{"git", "remote", "update"},
			WorkDir: r.AbsolutePath(),
		})
	}

	return nil
}

//Exec implements the meat of the `rtree exec` command. It returns
//true iff the command exited successfully.
func (r Repo) Exec(cmdline ...string) error {
	cli.Interface.ShowProgress(r.AbsolutePath())
	return cli.Interface.Run(cli.Command{
		Program: cmdline,
		WorkDir: r.AbsolutePath(),
	})
}

//Move sets the CheckoutPath to the given value and moves the existing repo
//from the old to the new checkoutPath. If makeSymlink is given, a symlink will
//be created from the old to the new location.
func (r *Repo) Move(checkoutPath string, makeSymlink bool) error {
	sourcePath := filepath.Join(RootPath, r.CheckoutPath)
	targetPath := filepath.Join(RootPath, checkoutPath)

	//ensure that target does not exist
	_, err := os.Lstat(targetPath)
	if err == nil {
		return fmt.Errorf("cannot move %s to %s: target exists in filesystem", sourcePath, targetPath)
	}
	if !os.IsNotExist(err) {
		return err
	}

	//prepare directory to move repo into
	err = os.MkdirAll(filepath.Dir(targetPath), 0755)
	if err != nil {
		return err
	}

	//move directory
	err = os.Rename(sourcePath, targetPath)
	if err != nil {
		return err
	}
	r.CheckoutPath = checkoutPath

	//if requested, make compatibility symlink
	if makeSymlink {
		return os.Symlink(targetPath, sourcePath)
	}
	return nil
}
