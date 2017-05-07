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
	"path/filepath"

	"github.com/majewsky/gofu/pkg/cli"
	"github.com/majewsky/gofu/pkg/rtree"
)

func main() {
	ci, err := cli.NewInterface(os.Stdin, os.Stdout, os.Stderr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "FATAL: initialization failed")
		os.Exit(255)
	}
	os.Exit(execApplet(ci, filepath.Base(os.Args[0]), os.Args[1:], true))
}

func execApplet(ci *cli.Interface, applet string, args []string, allowGofu bool) int {
	//allow explicit specification of applet as "./build/gofu <applet> <args>"
	if allowGofu && applet == "gofu" {
		if len(args) == 0 {
			fmt.Fprintln(os.Stderr, "Usage: gofu <applet> [args...]")
			return 1
		}
		return execApplet(ci, args[0], args[1:], false)
	}

	switch applet {
	case "rtree":
		return rtree.Exec(ci, args)
	default:
		fmt.Fprintln(os.Stderr, "ERROR: unknown applet: "+applet)
		return 255
	}
}
