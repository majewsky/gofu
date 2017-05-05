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
func Exec(args []string) {
	if len(args) == 0 {
		usageAndExit()
	}
	switch args[0] {
	case "get":
		if len(args) != 2 {
			usageAndExit()
		}
		commandGet(args[1])
	case "drop":
		panic("unimplemented")
	case "index":
		if len(args) != 1 {
			usageAndExit()
		}
		commandIndex()
	case "repos":
		if len(args) != 1 {
			usageAndExit()
		}
		commandRepos()
	case "remotes":
		if len(args) != 1 {
			usageAndExit()
		}
		commandRemotes()
	case "import":
		if len(args) != 2 {
			usageAndExit()
		}
		commandImport(args[1])
	case "each":
		if len(args) < 2 {
			usageAndExit()
		}
		commandEach(args[1], args[2:])
	default:
		usageAndExit()
	}

	os.Exit(0)
}

func usageAndExit() {
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, "  rtree [get|drop] <url>")
	fmt.Fprintln(os.Stderr, "  rtree [index|repos|remotes]")
	fmt.Fprintln(os.Stderr, "  rtree import <path>")
	fmt.Fprintln(os.Stderr, "  rtree each <command>")
	os.Exit(1)
}

func commandGet(url string) {
	index := ReadIndex()
	repo := index.InteractiveFindRepo(url, true)
	if repo != nil {
		fmt.Println(repo.AbsolutePath())
	}
}

func commandIndex() {
	index := ReadIndex()
	util.FatalIfError(index.InteractiveRebuild())
	index.Write()
}

func commandRepos() {
	index := ReadIndex()
	var items []string
	for _, repo := range index.Repos {
		items = append(items, repo.CheckoutPath)
	}
	util.ShowSorted(items)
}

func commandRemotes() {
	index := ReadIndex()
	var items []string
	for _, repo := range index.Repos {
		for _, remote := range repo.Remotes {
			items = append(items, remote.URL)
		}
	}
	util.ShowSorted(items)
}

func commandEach(command string, args []string) {
	allOK := true
	for _, repo := range ReadIndex().Repos {
		ok := repo.InteractiveExec(command, args...)
		if !ok {
			allOK = false
		}
	}

	if !allOK {
		os.Exit(1)
	}
}

func commandImport(dirPath string) {
	index := ReadIndex()
	index.InteractiveImportRepo(dirPath)
	index.Write()
}
