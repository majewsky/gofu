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

package main

import (
	"fmt"
	"os"
)

//ShowError prints the given error on stderr if it is non-nil, or returns false otherwise.
func ShowError(err error) bool {
	if err == nil {
		return false
	}
	fmt.Fprintf(os.Stderr, "ERROR: %s\n", err.Error())
	return true
}

//FatalIfError prints the given error on stderr and exits with an error code.
func FatalIfError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: %s\n", err.Error())
		os.Exit(255)
	}
}
