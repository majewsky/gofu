/*******************************************************************************
*
* Copyright 2021 Stefan Majewsky <majewsky@gmx.net>
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
