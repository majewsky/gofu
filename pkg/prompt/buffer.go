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

import "bytes"

//LineBuffer wraps bytes.Buffer that measures the length of the displayed text
//(minus ANSI escape sequences).
type LineBuffer struct {
	buffer bytes.Buffer
	length int
}

//Bytes returns the complete string (including non-printable characters) in the
//buffer.
func (b *LineBuffer) Bytes() []byte {
	return b.buffer.Bytes()
}

//Length returns the number of printable characters in the buffer.
func (b *LineBuffer) Length() int {
	return b.length
}

//Write implements the io.Writer interface. Use this only for printable
//characters.
func (b *LineBuffer) Write(buf []byte) (n int, err error) {
	n, err = b.buffer.Write(buf)
	b.length += n
	return
}

//WriteNonprintable works like Write, but bytes written do not count towards
//Length().
func (b *LineBuffer) WriteNonprintable(buf []byte) (int, error) {
	return b.buffer.Write(buf)
}

//WriteWithColor writes a colored printable string. The color is given as the
//semicolon-separated list of arguments to the ANSI escape sequence SGR, e.g.
//"1;41" for bold with red background.
func (b *LineBuffer) WriteWithColor(text, color string) {
	if color == "0" {
		b.Write([]byte(text))
	} else {
		b.WriteNonprintable([]byte("\x1B[" + color + "m"))
		b.Write([]byte(text))
		b.WriteNonprintable([]byte("\x1B[0m"))
	}
}
