// SPDX-FileCopyrightText: 2017 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: GPL-3.0-only

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/majewsky/gofu/internal/mdedit"
	"github.com/majewsky/gofu/internal/prompt"
	"github.com/majewsky/gofu/internal/rtree"
)

func main() {
	os.Exit(execApplet(filepath.Base(os.Args[0]), os.Args[1:], true))
}

func execApplet(applet string, args []string, allowGofu bool) int {
	//allow explicit specification of applet as "./build/gofu <applet> <args>"
	if allowGofu && applet == "gofu" {
		if len(args) == 0 {
			fmt.Fprintln(os.Stderr, "Usage: gofu <applet> [args...]")
			return 1
		}
		return execApplet(args[0], args[1:], false)
	}

	switch applet {
	case "mdedit":
		return mdedit.Exec(args)
	case "prettyprompt":
		return prompt.Exec(args)
	case "rtree":
		return rtree.Exec(args)
	default:
		fmt.Fprintln(os.Stderr, "ERROR: unknown applet: "+applet)
		return 255
	}
}
