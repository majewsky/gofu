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
	"strings"

	"github.com/majewsky/gofu/pkg/cli"
)

//Exec executes the prettyprompt applet and returns an exit code (0 for
//success, >0 for error).
func Exec() int {
	fields := []string{
		getLoginField(),
	}
	line := strings.Join(fields, " ")

	os.Stdout.Write([]byte(line + "\n"))
	return 0
}

func getenvOrDefault(key, defaultValue string) (value string) {
	value = os.Getenv(key)
	if value == "" {
		value = defaultValue
	}
	return
}

func handleError(err error) {
	if err != nil {
		os.Stderr.Write([]byte("\x1B[1;31mPrompt error: " + err.Error() + "\x1B[0m\n"))
	}
}

func getPrintableLength(text string) int {
	return len(cli.AnsiEscapeRx.ReplaceAllString(text, ""))
}

//withColor adds ANSI escape sequences to the string to display it with a
//certain color. The color is given as the semicolon-separated list of
//arguments to the ANSI escape sequence SGR, e.g. "1;41" for bold with red
//background.
func withColor(color, text string) string {
	if color == "0" {
		return text
	}
	return "\x1B[" + color + "m" + text + "\x1B[0m"
}
