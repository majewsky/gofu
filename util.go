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
	"bufio"
	"fmt"
	"os"
	"strings"
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

var stdin = bufio.NewReader(os.Stdin)

//Prompt prints the question, then waits for the user to press one of the
//possible answer keys. Answer keys will automatically be converted to lower
//case and returned as such.
//
//    choice := Prompt("(y)es or (n)o", []string{"y","n"})
//    //choice is either "y" or "n"
func Prompt(question string, answers []string) string {
	for idx, answer := range answers {
		answers[idx] = strings.ToLower(answer)
	}

	os.Stdout.Write([]byte(">> " + strings.TrimSpace(question) + " "))
	for {
		input, err := stdin.ReadString('\n')
		FatalIfError(err)
		input = strings.TrimSpace(input)
		for _, answer := range answers {
			if strings.ToLower(input) == answer {
				return answer
			}
		}

		//user typed gibberish - ask again
		os.Stdout.Write([]byte("Please type "))
		for idx, answer := range answers {
			if idx > 0 {
				if idx == len(answers)-1 {
					os.Stdout.Write([]byte(" or "))
				} else {
					os.Stdout.Write([]byte(", "))
				}
			}
			os.Stdout.Write([]byte("'" + answer + "'"))
		}
		os.Stdout.Write([]byte(": "))
	}
}
