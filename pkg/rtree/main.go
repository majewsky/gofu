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

	"github.com/majewsky/gofu/pkg/cli"
)

//Exec executes the rtree applet and does not return. The argument is os.Args
//minus the leading "rtree" or "gofu rtree".
func Exec(ci *cli.Interface, args []string) int {
	if len(args) == 0 {
		return usage(ci)
	}

	index, errs := ReadIndex()
	if len(errs) > 0 {
		for _, err := range errs {
			ci.ShowError(err.Error())
		}
		return 1
	}

	var err error
	switch args[0] {
	case "get":
		if len(args) != 2 {
			return usage(ci)
		}
		err = commandGet(ci, index, args[1])
	case "drop":
		if len(args) != 2 {
			return usage(ci)
		}
		err = commandDrop(ci, index, args[1])
	case "index":
		if len(args) != 1 {
			return usage(ci)
		}
		err = commandIndex(ci, index)
	case "repos":
		if len(args) != 1 {
			return usage(ci)
		}
		commandRepos(ci, index)
	case "remotes":
		if len(args) != 1 {
			return usage(ci)
		}
		commandRemotes(ci, index)
	case "import":
		if len(args) != 2 {
			return usage(ci)
		}
		err = commandImport(ci, index, args[1])
	case "each":
		if len(args) < 2 {
			return usage(ci)
		}
		return commandEach(ci, index, args[1:])
	default:
		return usage(ci)
	}

	if err == nil {
		return 0
	}
	ci.ShowError(err.Error())
	return 1
}

var usageStr = strings.TrimSpace(`
Usage:
  rtree [get|drop] <url>
  rtree [index|repos|remotes]
  rtree import <path>
  rtree each <command>
`)

func usage(ci *cli.Interface) int {
	ci.ShowUsage(usageStr)
	return 1
}

func commandGet(ci *cli.Interface, index *Index, url string) error {
	repo, err := index.FindRepo(ci, url, true)
	if err != nil {
		return err
	}
	ci.ShowResult(repo.AbsolutePath())
	return nil
}

func commandDrop(ci *cli.Interface, index *Index, url string) error {
	repo, err := index.FindRepo(ci, url, true)
	if err != nil {
		return err
	}
	return index.DropRepo(ci, repo)
}

func commandIndex(ci *cli.Interface, index *Index) error {
	err := index.Rebuild(ci)
	if err != nil {
		return err
	}
	return index.Write(ci)
}

func commandRepos(ci *cli.Interface, index *Index) {
	var items []string
	for _, repo := range index.Repos {
		items = append(items, repo.CheckoutPath)
	}
	ci.ShowResultsSorted(items)
}

func commandRemotes(ci *cli.Interface, index *Index) {
	var items []string
	for _, repo := range index.Repos {
		for _, remote := range repo.Remotes {
			items = append(items, remote.URL)
		}
	}
	ci.ShowResultsSorted(items)
}

func commandEach(ci *cli.Interface, index *Index, cmdline []string) (exitCode int) {
	exitCode = 0
	for _, repo := range index.Repos {
		err := repo.Exec(ci, cmdline...)
		if err != nil {
			ci.ShowError(err.Error())
			exitCode = 1
		}
	}
	return
}

func commandImport(ci *cli.Interface, index *Index, dirPath string) error {
	err := index.ImportRepo(ci, dirPath)
	if err != nil {
		return err
	}
	return index.Write(ci)
}
