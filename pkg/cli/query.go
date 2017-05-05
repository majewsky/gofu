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

package cli

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	terminal "golang.org/x/crypto/ssh/terminal"
)

//Confirm displays a yes/no question and returns whether the user answered "yes".
func Confirm(question string) bool {
	os.Stdout.Write([]byte(strings.TrimSpace(question) + " [y/n] "))

	buf := buffer{Input: os.Stdin}
	for {
		switch string(buf.getNextInput()) {
		case "y", "Y":
			os.Stdout.Write([]byte("-> yes\n"))
			return true
		case "n", "N":
			os.Stdout.Write([]byte("-> no\n"))
			return false
		case "\x03": // Ctrl-C
			fmt.Fprintln(os.Stderr, "\nInterrupted!")
			os.Exit(255)
		}
	}
}

//Choice is a thing that the user can choose during Query().
type Choice struct {
	//If given, the choice can be selected by pressing the key that
	//produces this character.
	Shortcut byte
	//The display string that describes this choice.
	Text string
}

func (c Choice) hasShortcut() bool {
	return c.Shortcut != '\000'
}

//Query displays a question and a set of answers and allows the user to select
//one of the answers. Returns the selected Choice instance, as well as its
//index in the original choices list (starting from 0).
func Query(prompt string, choices ...Choice) (Choice, int) {
	if len(choices) == 0 {
		return Choice{}, -1
	}

	//disable line wrap; unexpected wrapping would confuse our cursor-moving code
	os.Stdout.Write([]byte("\x1B[?7l"))
	defer os.Stdout.Write([]byte("\x1B[?7h"))

	//display question
	os.Stdout.Write([]byte(strings.TrimSuffix(prompt, "\n") + "\n"))
	selected := 0

	buf := buffer{Input: os.Stdin}
OUTER:
	for {
		displayChoices(os.Stdout, choices, selected)

		input := buf.getNextInput()
		if len(input) == 1 {
			//single character entered - check if a shortcut matches
			for idx, choice := range choices {
				if choice.Shortcut == input[0] {
					selected = idx
					break OUTER
				}
			}
		}

		switch string(input) {
		case "\r", "\n":
			break OUTER
		case "\x1B[A": // Up arrow key
			selected--
			if selected < 0 {
				selected = 0
			}
		case "\x1B[B": // Down arrow key
			selected++
			if selected >= len(choices) {
				selected = len(choices) - 1
			}
		case "\x03": // Ctrl-C
			fmt.Fprintln(os.Stderr, "Interrupted!")
			os.Exit(255)
		}

		//prepare to re-render choices
		removeDisplayLines(len(choices))
	}

	//clear query display
	removeDisplayLines(len(choices) + 1)

	//display question + chosen answer
	fmt.Fprintf(os.Stdout, "%s -> %s\n",
		strings.TrimSuffix(prompt, "\n"),
		strings.TrimSpace(choices[selected].Text),
	)

	return choices[selected], selected
}

func removeDisplayLines(n int) {
	for idx := 0; idx < n; idx++ {
		os.Stdout.Write([]byte("\x1B[A\x1B[2K"))
	}
}

func displayChoices(out io.Writer, choices []Choice, selectedIndex int) {
	hasShortcuts := false
	for _, choice := range choices {
		if choice.hasShortcut() {
			hasShortcuts = true
			break
		}
	}

	for idx, choice := range choices {
		text := " " + strings.TrimSpace(choice.Text) + " \n"
		if hasShortcuts {
			shortcut := choice.Shortcut
			if !choice.hasShortcut() {
				shortcut = ' '
			}
			text = fmt.Sprintf(" [%c]%s", shortcut, text)
		}

		if idx == selectedIndex {
			out.Write(Styled(text, AnsiInverse).DisplayString(true))
		} else {
			out.Write([]byte(text))
		}
	}
}

var ansiEscapeRx = regexp.MustCompile(`^\x1B\[[\x20-\x3F]*[\x40-\x7E]`)

type buffer struct {
	Input io.Reader
	buf   [128]byte
	fill  int
}

func (b *buffer) getNextInput() []byte {
	//do we have a simple input character?
	if b.fill > 0 && b.buf[0] != '\x1B' {
		result := append([]byte(nil), b.buf[0])
		copy(b.buf[0:], b.buf[1:])
		b.fill--
		return result
	}

	//do we have a full ANSI escape sequence?
	match := ansiEscapeRx.Find(b.buf[0:b.fill])
	if match != nil {
		result := append([]byte(nil), match...)
		copy(b.buf[0:], b.buf[len(match):])
		b.fill -= len(match)
		return result
	}

	oldState, err := terminal.MakeRaw(0)
	if err != nil {
		panic(err)
	}
	defer terminal.Restore(0, oldState)

	//fill buffer some more
	n, err := os.Stdin.Read(b.buf[b.fill:])
	if err != nil {
		panic(err)
	}
	b.fill += n

	return b.getNextInput() //restart
}
