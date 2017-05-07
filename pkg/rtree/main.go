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

	"github.com/majewsky/gofu/pkg/util"
)

//Exec executes the rtree applet and does not return. The argument is os.Args
//minus the leading "rtree" or "gofu rtree".
func Exec(args []string) int {
	if len(args) == 0 {
		return usage()
	}

	index, errs := ReadIndex()
	if len(errs) > 0 {
		for _, err := range errs {
			util.ShowError(err)
		}
		return 255
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
		return commandEach(index, args[1], args[2:])
	default:
		return usage()
	}

	if err == nil {
		return 0
	}
	util.ShowError(err)
	return 1
}

func usage() int {
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, "  rtree [get|drop] <url>")
	fmt.Fprintln(os.Stderr, "  rtree [index|repos|remotes]")
	fmt.Fprintln(os.Stderr, "  rtree import <path>")
	fmt.Fprintln(os.Stderr, "  rtree each <command>")
	return 1
}

func commandGet(index *Index, url string) error {
	repo, err := index.InteractiveFindRepo(url, true)
	if err != nil {
		return err
	}
	fmt.Println(repo.AbsolutePath())
	return nil
}

func commandDrop(index *Index, url string) error {
	repo, err := index.InteractiveFindRepo(url, true)
	if err != nil {
		return err
	}
	err = index.InteractiveDropRepo(repo)
	if err != nil {
		return err
	}
	return index.Write()
}

func commandIndex(index *Index) error {
	err := index.InteractiveRebuild()
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
	util.ShowSorted(items)
}

func commandRemotes(index *Index) {
	var items []string
	for _, repo := range index.Repos {
		for _, remote := range repo.Remotes {
			items = append(items, remote.URL)
		}
	}
	util.ShowSorted(items)
}

func commandEach(index *Index, command string, args []string) int {
	allOK := true
	for _, repo := range index.Repos {
		ok := repo.InteractiveExec(command, args...)
		if !ok {
			allOK = false
		}
	}

	if allOK {
		return 0
	}
	return 1
}

func commandImport(index *Index, dirPath string) error {
	err := index.InteractiveImportRepo(dirPath)
	if err != nil {
		return err
	}
	return index.Write()
}
