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

package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) == 1 {
		usageAndExit()
	}
	switch os.Args[1] {
	case "get":
		panic("unimplemented")
	case "drop":
		panic("unimplemented")
	case "index":
		if len(os.Args) != 2 {
			usageAndExit()
		}
		commandIndex()
	case "repos":
		panic("unimplemented")
	case "remotes":
		panic("unimplemented")
	case "import":
		panic("unimplemented")
	case "each":
		panic("unimplemented")
	default:
		usageAndExit()
	}
}

func usageAndExit() {
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, "  rtree [get|drop] <url>")
	fmt.Fprintln(os.Stderr, "  rtree [index|repos|remotes]")
	fmt.Fprintln(os.Stderr, "  rtree import <path>")
	fmt.Fprintln(os.Stderr, "  rtree each <command>")
	os.Exit(1)
}

func commandIndex() {
}
