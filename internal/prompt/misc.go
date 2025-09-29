// SPDX-FileCopyrightText: 2017 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: GPL-3.0-only

package prompt

import (
	"os"
	"strconv"
)

func getTerminalField() string {
	termName := os.Getenv("TERM")
	if termName == "xterm-256color" {
		return ""
	}
	if termName == "" {
		termName = withColor("1;41", "not set")
	}
	return withType("term", termName)
}

func getExitCodeField(arg string) string {
	exitCode, err := strconv.Atoi(arg)
	if err == nil && exitCode > 0 {
		return withColor("1;31", "exit:"+arg)
	}
	return ""
}
