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

import "os"

//#include <sys/types.h>
//#include <pwd.h>
//#include <unistd.h>
import "C"

func getLoginField() string {
	var result string

	//show user name
	userName := getUserName()
	commonUser := getenvOrDefault("PRETTYPROMPT_COMMONUSER", "stefan")
	if commonUser != userName {
		color := "0"
		if userName == "root" {
			color = "1;41"
		}
		result = withColor(color, userName) + "@"
	}

	//show hostname
	hostname, err := os.Hostname()
	if err != nil {
		handleError(err)
		hostname = "<unknown>"
	}
	return result + withColor(
		getenvOrDefault("PRETTYPROMPT_HOSTCOLOR", "0;33"),
		hostname,
	)
}

func getUserName() string {
	//try to find username via getpwuid(getuid())
	pw, err := C.getpwuid(C.getuid())
	if err == nil {
		return C.GoString(pw.pw_name)
	}
	//fallback to $USER, if set
	if name := os.Getenv("USER"); name != "" {
		return name
	}
	handleError(err)
	return "<unknown>"
}
