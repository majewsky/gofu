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
	"strings"

	"github.com/majewsky/gofu/internal/cli"
)

//Exec executes the rtree applet and returns an exit code (0 for success, >0
//for error). The argument is os.Args minus the leading "rtree" or "gofu
//rtree". All side-effects (reading from stdin, writing to stdout/stderr,
//executing other programs) pass through cli.Interface and can be intercepted
//there for the purpose of testing.
func Exec(args []string) int {
	if !Init() {
		return 1
	}

	if len(args) == 0 {
		return usage()
	}

	index, errs := ReadIndex()
	if len(errs) > 0 {
		for _, err := range errs {
			cli.Interface.ShowError(err.Error())
		}
		return 1
	}

	var err error
	switch args[0] {
	case "get":
		if len(args) != 2 {
			return usage()
		}
		err = commandGet(index, args[1])
	case "drop":
		if len(args) != 2 {
			return usage()
		}
		err = commandDrop(index, args[1])
	case "index":
		if len(args) != 1 {
			return usage()
		}
		err = commandIndex(index)
	case "repos":
		if len(args) != 1 {
			return usage()
		}
		commandRepos(index)
	case "remotes":
		if len(args) != 1 {
			return usage()
		}
		commandRemotes(index)
	case "import":
		if len(args) != 2 {
			return usage()
		}
		err = commandImport(index, args[1])
	case "each":
		if len(args) < 2 {
			return usage()
		}
		return commandEach(index, args[1:])
	default:
		return usage()
	}

	if err == nil {
		return 0
	}
	cli.Interface.ShowError(err.Error())
	return 1
}

var usageStr = strings.TrimSpace(`
Usage:
  rtree [get|drop] <url>
  rtree [index|repos|remotes]
  rtree import <path>
  rtree each <command>
`)

func usage() int {
	cli.Interface.ShowUsage(usageStr)
	return 1
}

func commandGet(index *Index, url string) error {
	repo, err := index.FindRepo(url, true)
	if err != nil {
		return err
	}
	cli.Interface.ShowResult(repo.AbsolutePath())
	return nil
}

func commandDrop(index *Index, url string) error {
	repo, err := index.FindRepo(url, false)
	if err != nil {
		return err
	}
	return index.DropRepo(repo)
}

func commandIndex(index *Index) error {
	err := index.Rebuild()
	if err != nil {
		return err
	}
	return index.Write()
}

func commandRepos(index *Index) {
	var items []string
	for _, repo := range index.Repos {
		items = append(items, repo.CheckoutPath)
	}
	cli.Interface.ShowResultsSorted(items)
}

func commandRemotes(index *Index) {
	var items []string
	for _, repo := range index.Repos {
		for _, remote := range repo.Remotes {
			items = append(items, remote.URL)
		}
	}
	cli.Interface.ShowResultsSorted(items)
}

func commandEach(index *Index, cmdline []string) (exitCode int) {
	exitCode = 0
	for _, repo := range index.Repos {
		err := repo.Exec(cmdline...)
		if err != nil {
			cli.Interface.ShowError(err.Error())
			exitCode = 1
		}
	}
	return
}

func commandImport(index *Index, dirPath string) error {
	err := index.ImportRepo(dirPath)
	if err != nil {
		return err
	}
	return index.Write()
}
