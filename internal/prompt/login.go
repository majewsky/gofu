// SPDX-FileCopyrightText: 2017 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: GPL-3.0-only

package prompt

import "os"

//#include <sys/types.h>
//#include <pwd.h>
//#include <unistd.h>
import "C"

func getLoginField(tt *TerminalTitle) string {
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
	tt.HostName = hostname
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
