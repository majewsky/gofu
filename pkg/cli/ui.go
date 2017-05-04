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
	"bytes"
	"strings"
)

//AnsiStyle offers a set of styles for the VT100 SGR (Select Graphic Recognition) command.
type AnsiStyle string

const (
	//AnsiNormal reverts all styles to default.
	AnsiNormal AnsiStyle = "0"
	//AnsiInverse swaps foreground and background color.
	AnsiInverse AnsiStyle = "7"
	//AnsiBlack is a foreground color.
	AnsiBlack AnsiStyle = "30"
	//AnsiRed is a foreground color.
	AnsiRed AnsiStyle = "31"
	//AnsiGreen is a foreground color.
	AnsiGreen AnsiStyle = "32"
	//AnsiYellow is a foreground color.
	AnsiYellow AnsiStyle = "33"
	//AnsiBlue is a foreground color.
	AnsiBlue AnsiStyle = "34"
	//AnsiMagenta is a foreground color.
	AnsiMagenta AnsiStyle = "35"
	//AnsiCyan is a foreground color.
	AnsiCyan AnsiStyle = "36"
	//AnsiWhite is a foreground color.
	AnsiWhite AnsiStyle = "37"
	//AnsiOnBlack is a background color.
	AnsiOnBlack AnsiStyle = "40"
	//AnsiOnRed is a background color.
	AnsiOnRed AnsiStyle = "41"
	//AnsiOnGreen is a background color.
	AnsiOnGreen AnsiStyle = "42"
	//AnsiOnYellow is a background color.
	AnsiOnYellow AnsiStyle = "43"
	//AnsiOnBlue is a background color.
	AnsiOnBlue AnsiStyle = "44"
	//AnsiOnMagenta is a background color.
	AnsiOnMagenta AnsiStyle = "45"
	//AnsiOnCyan is a background color.
	AnsiOnCyan AnsiStyle = "46"
	//AnsiOnWhite is a background color.
	AnsiOnWhite AnsiStyle = "47"
)

//StyledString is a string with attached style information for displaying on a terminal.
type StyledString struct {
	Text   string
	Styles []AnsiStyle
}

//Styled is syntactic sugar for a StyledString instance.
func Styled(text string, styles ...AnsiStyle) StyledString {
	return StyledString{text, styles}
}

//DisplayString returns the string for writing on a terminal emulator.
func (s StyledString) DisplayString(withReset bool) []byte {
	strs := make([]string, len(s.Styles))
	for idx, s := range s.Styles {
		strs[idx] = string(s)
	}
	str := "\x1B[" + strings.Join(strs, ";") + "m" + s.Text
	if withReset {
		str += "\x1B[0m"
	}
	return []byte(str)
}

//StyledText is a sequence of styled strings.
type StyledText []StyledString

//DisplayString returns the string for writing on a terminal emulator.
func (t StyledText) DisplayString(withReset bool) []byte {
	var buf bytes.Buffer
	for _, s := range t {
		buf.Write(s.DisplayString(false))
	}
	if withReset {
		buf.Write([]byte("\x1B[0m"))
	}
	return buf.Bytes()
}
