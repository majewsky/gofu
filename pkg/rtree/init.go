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
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/majewsky/gofu/pkg/cli"
)

//RemoteAlias describes an alias that can be used in a Git remote URL (as
//defined by the "url.<base>.insteadOf" directive in man:git-config(1)).
type RemoteAlias struct {
	Alias       string
	Replacement string
}

//IndexPath is where the index file is stored.
var IndexPath string

//RootPath is the directory below which all repositories are located. Its value
//is $GOPATH/src to match the repository layout created by `go get`.
var RootPath string

//RemoteAliases is the list of remote aliases that is used by ExpandRemoteURL().
var RemoteAliases []*RemoteAlias

//Init initializes the global variables of this package to their standard
//values, unless they are already populated. Unit tests shall set IndexPath,
//RootPath etc. before calling Exec(), such that this function becomes a no-op
//when called by Exec().
//
//Returns false if initialization failed.
func Init() bool {
	ok := true //until shown otherwise

	if IndexPath == "" {
		homeDir := os.Getenv("HOME")
		if homeDir == "" {
			cli.Interface.ShowError("$HOME is not set (rtree needs the HOME variable to locate its index file)")
			ok = false //but keep going to report all errors at once
		} else {
			IndexPath = filepath.Join(homeDir, ".rtree/index.yaml")
		}
	}

	if RootPath == "" {
		gopath := os.Getenv("GOPATH")
		if gopath == "" {
			cli.Interface.ShowError("$GOPATH is not set (rtree needs the GOPATH variable to know where to look for and place repos)")
			ok = false //but keep going to report all errors at once
		} else {
			RootPath = filepath.Join(gopath, "src")
		}
	}

	if RemoteAliases == nil {
		out, err := cli.Interface.CaptureStdout(cli.Command{
			Program: []string{"git", "config", "--global", "-l"},
		})
		if err != nil {
			cli.Interface.ShowError(err.Error())
			ok = false //but keep going to report all errors at once
		}

		rx := regexp.MustCompile(`^url\.([^=]+)\.insteadof=(.+)$`)
		for _, line := range strings.Split(out, "\n") {
			match := rx.FindStringSubmatch(line)
			if match == nil {
				continue
			}
			RemoteAliases = append(RemoteAliases, &RemoteAlias{
				Alias:       match[2],
				Replacement: match[1],
			})
		}
	}

	return ok
}
