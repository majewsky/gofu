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
