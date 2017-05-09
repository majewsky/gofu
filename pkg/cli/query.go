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
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	terminal "golang.org/x/crypto/ssh/terminal"
)

//cannot use `var errInterrupted = errors.New("Interrupted!")` because golint
//complains about the formatting of the error message
type errInterrupted struct{}

func (e errInterrupted) Error() string {
	return "Interrupted!"
}

////////////////////////////////////////////////////////////////////////////////
// TUI implementation for when stdin is a terminal

type terminalTUI struct {
	i *Interface
}

func (t terminalTUI) ReadLine(prompt string) (string, error) {
	if prompt != "" {
		t.i.safeStdout().Write([]byte(strings.TrimSpace(prompt) + " "))
	}
	str, err := t.i.stdinBuf.ReadString('\n')
	return strings.TrimSpace(str), err
}

func (t terminalTUI) Confirm(question string) (bool, error) {
	out := t.i.safeStdout()
	out.Write([]byte(strings.TrimSpace(question) + " [y/n] "))

	buf := buffer{Input: t.i.stdin}
	for {
		switch string(buf.getNextInput()) {
		case "y", "Y":
			out.Write([]byte("-> yes\n"))
			return true, nil
		case "n", "N":
			out.Write([]byte("-> no\n"))
			return false, nil
		case "\x03": // Ctrl-C
			return false, errInterrupted{}
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
	//The string to return from Interface.Query().
	Return string
}

func (c Choice) hasShortcut() bool {
	return c.Shortcut != '\000'
}

func (t terminalTUI) Query(prompt string, choices ...Choice) (string, error) {
	if len(choices) == 0 {
		panic("no choices")
	}

	//disable line wrap; unexpected wrapping would confuse our cursor-moving code
	out := t.i.safeStdout()
	out.Write([]byte("\x1B[?7l"))
	defer out.Write([]byte("\x1B[?7h"))

	//display question
	out.Write([]byte(strings.TrimSuffix(prompt, "\n") + "\n"))
	selected := 0

	buf := buffer{Input: t.i.stdin}
OUTER:
	for {
		displayChoices(out, choices, selected)

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
			return "", errInterrupted{}
		}

		//prepare to re-render choices
		removeDisplayLines(out, len(choices))
	}

	//clear query display
	removeDisplayLines(out, len(choices)+1)

	//display question + chosen answer
	fmt.Fprintf(out, "%s -> %s\n",
		strings.TrimSuffix(prompt, "\n"),
		strings.TrimSpace(choices[selected].Text),
	)

	return choices[selected].Return, nil
}

func removeDisplayLines(stdout io.Writer, n int) {
	for idx := 0; idx < n; idx++ {
		stdout.Write([]byte("\x1B[A\x1B[2K"))
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
			fmt.Fprintf(out, "\x1B[0;7m%s\x1B[0m", text)
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
	n, err := b.Input.Read(b.buf[b.fill:])
	if err != nil {
		panic(err)
	}
	b.fill += n

	return b.getNextInput() //restart
}

////////////////////////////////////////////////////////////////////////////////
// TUI implementation for when stdin is a pipe

type pipeTUI struct {
	i *Interface
}

func (t *pipeTUI) ReadLine(prompt string) (string, error) {
	str, err := t.i.stdinBuf.ReadString('\n')
	str = strings.TrimSpace(str)
	if err == nil {
		fmt.Fprintf(t.i.stderr, "%s %s\n", strings.TrimSpace(prompt), str)
	}
	return str, err
}

func (t *pipeTUI) Confirm(question string) (bool, error) {
	question = strings.TrimSpace(question)
	str, err := t.ReadLine(question)
	if err != nil {
		return false, err
	}
	//recognize /[01tTfF]|true|false/
	ok, err := strconv.ParseBool(str)
	if err != nil {
		//recognize /[yY]|yes|no/
		ok = strings.HasPrefix(question, "y") || strings.HasPrefix(question, "Y")
	}
	fmt.Fprintf(t.i.stderr, "%s -> %v (%s)\n", question, ok, str)
	return ok, nil
}

//Query for the pipe TUI is limited to exact matches of the choice text, or
//matching by choice shortcut.
func (t *pipeTUI) Query(prompt string, choices ...Choice) (string, error) {
	str, err := t.i.stdinBuf.ReadString('\n')
	if err != nil {
		return str, err
	}
	//prefer exact match on choice.Text
	for _, choice := range choices {
		if choice.Text == str {
			fmt.Fprintf(t.i.stderr, "%s -> %s\n", prompt, choice.Text)
			return choice.Return, nil
		}
	}
	//allow match on choice.Shortcut
	for _, choice := range choices {
		if string(choice.Shortcut) == str {
			fmt.Fprintf(t.i.stderr, "%s -> %s\n", prompt, choice.Text)
			return choice.Return, nil
		}
	}
	fmt.Fprintf(t.i.stderr, "%s -> [%s]\n", prompt, str)
	return "", errors.New("cannot match input with available choices")
}
