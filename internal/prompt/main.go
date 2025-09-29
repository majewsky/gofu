// SPDX-FileCopyrightText: 2017 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: GPL-3.0-only

package prompt

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"

	"github.com/majewsky/gofu/internal/cli"
)

// Exec executes the prettyprompt applet and returns an exit code (0 for
// success, >0 for error).
func Exec(args []string) int {
	var tt TerminalTitle
	fields := []string{
		getLoginField(&tt),
	}
	cwd := CurrentDirectory()
	fields = appendUnlessEmpty(fields, getDirectoryField(cwd, &tt))
	fields = appendUnlessEmpty(fields, getDeletedMessageField(cwd))
	fields = appendUnlessEmpty(fields, getRepoStatusField(cwd.Repo))
	fields = appendUnlessEmpty(fields, getTerminalField())
	fields = appendUnlessEmpty(fields, getOpenstackField())
	fields = appendUnlessEmpty(fields, getKubernetesField())
	//this field should always be last
	if len(args) > 0 {
		fields = appendUnlessEmpty(fields, getExitCodeField(args[0]))
	}

	//print terminal title first, otherwise it confuses zsh's line length
	//computation and makes it misplace UI elements
	tt.PrintTo(os.Stdout)

	line := strings.Join(fields, " ")
	lineWidth := getPrintableLength(line)

	//add dashes to expand `line` to fill the terminal's width
	termWidth, _, err := term.GetSize(0)
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

// withColor adds ANSI escape sequences to the string to display it with a
// certain color. The color is given as the semicolon-separated list of
// arguments to the ANSI escape sequence SGR, e.g. "1;41" for bold with red
// background.
func withColor(color, text string) string {
	if color == "0" {
		return text
	}
	return fmt.Sprintf("\x1B[%sm%s\x1B[0m", color, text)
}

// withType adds a type annotation with a standardized format to the text.
func withType(typeStr, text string) string {
	return fmt.Sprintf("\x1B[37m%s:\x1B[0m%s", typeStr, text)
}
