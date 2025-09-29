// SPDX-FileCopyrightText: 2021 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: GPL-3.0-only

package prompt

import (
	"fmt"
	"io"
)

//TerminalTitle contains all pieces that go into the terminal title.
type TerminalTitle struct {
	HostName string
	Path     string
}

//PrintTo produces the VT command that sets the terminal title.
func (t TerminalTitle) PrintTo(stdout io.Writer) {
	//warn about missing pieces
	if t.HostName == "" {
		t.HostName = "N/A"
	}
	if t.Path == "" {
		t.Path = "N/A"
	}

	fmt.Fprintf(stdout, "\x1B]0;\u23A3%s\u23A6 %s\x07", t.HostName, t.Path)
}
