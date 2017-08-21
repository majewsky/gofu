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
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/majewsky/gofu/pkg/cli"
)

//Exec executes the prettyprompt applet and returns an exit code (0 for
//success, >0 for error).
func Exec(args []string) int {
	fields := []string{
		getLoginField(),
	}
	cwd := CurrentDirectory()
	fields = appendUnlessEmpty(fields, getDirectoryField(cwd))
	fields = appendUnlessEmpty(fields, getDeletedMessageField(cwd))
	fields = appendUnlessEmpty(fields, getRepoStatusField(cwd.RepoRootPath))
	fields = appendUnlessEmpty(fields, getTerminalField())
	fields = appendUnlessEmpty(fields, getOpenstackField())
	fields = appendUnlessEmpty(fields, getKubernetesField())
	//this field should always be last
	if len(args) > 0 {
		fields = appendUnlessEmpty(fields, getExitCodeField(args[0]))
	}

	line := strings.Join(fields, " ")
	lineWidth := getPrintableLength(line)

	//add dashes to expand `line` to fill the terminal's width
	termWidth, _, err := terminal.GetSize(0)
	if err != nil {
		termWidth = 80
	}
	if termWidth > lineWidth {
		line += " "
		lineWidth++
	}
	if termWidth > lineWidth {
		dashes := make([]byte, termWidth-lineWidth)
		for idx := range dashes {
			dashes[idx] = '-'
		}
		line += withColor("1", string(dashes))
	}

	os.Stdout.Write([]byte(line + "\n"))

	//print second line: a letter identifying the shell, and the final "$ ")
	shellIdent := ""
	switch os.Getenv("PRETTYPROMPT_SHELL") {
	case "zsh":
		shellIdent = "Z"
	case "bash":
		shellIdent = "B"
	}
	os.Stdout.Write([]byte(shellIdent + "$ "))

	return 0
}

func getenvOrDefault(key, defaultValue string) (value string) {
	value = os.Getenv(key)
	if value == "" {
		value = defaultValue
	}
	return
}

func appendUnlessEmpty(list []string, val string) []string {
	if val == "" {
		return list
	}
	return append(list, val)
}

func handleError(err error) {
	if err != nil {
		os.Stderr.Write([]byte("\x1B[1;31mPrompt error: " + err.Error() + "\x1B[0m\n"))
	}
}

func getPrintableLength(text string) int {
	return len(cli.AnsiColorCodeRx.ReplaceAllString(text, ""))
}

//withColor adds ANSI escape sequences to the string to display it with a
//certain color. The color is given as the semicolon-separated list of
//arguments to the ANSI escape sequence SGR, e.g. "1;41" for bold with red
//background.
func withColor(color, text string) string {
	if color == "0" {
		return text
	}
	return fmt.Sprintf("\x1B[%sm%s\x1B[0m", color, text)
}

//withType adds a type annotation with a standardized format to the text.
func withType(typeStr, text string) string {
	return fmt.Sprintf("\x1B[37m%s:\x1B[0m%s", typeStr, text)
}
