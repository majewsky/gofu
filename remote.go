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
	"bytes"
	"os/exec"
	"regexp"
	"strings"
)

//remoteAlias describes an alias that can be used in a Git remote URL (as
//defined by the "url.<base>.insteadOf" directive in man:git-config(1)).
type remoteAlias struct {
	Alias       string
	Replacement string
}

var remoteAliases []*remoteAlias

func init() {
	cmd := exec.Command("git", "config", "--global", "-l")
	var buf bytes.Buffer
	cmd.Stdout = &buf
	FatalIfError(cmd.Run())

	rx := regexp.MustCompile(`^url\.([^=]+)\.insteadof=(.+)$`)
	for _, line := range strings.Split(string(buf.Bytes()), "\n") {
		match := rx.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		remoteAliases = append(remoteAliases, &remoteAlias{
			Alias:       match[2],
			Replacement: match[1],
		})
	}
}

//ExpandRemoteURL derive the canonical URL for a given remote by substituting
//aliases defined in the system-wide and user-global Git config. For example,
//with
//
//    $ cat /etc/gitconfig
//    [url "git://github.com/"]
//    insteadOf = gh:
//
//and the input "gh:foo/bar", this function returns "git://github.com/foo/bar".
func ExpandRemoteURL(url string) string {
	var best *remoteAlias
	for _, current := range remoteAliases {
		if strings.HasPrefix(url, current.Alias) {
			if best == nil || len(best.Alias) < len(current.Alias) {
				best = current
			}
		}
	}
	if best == nil {
		return url
	}
	return best.Replacement + strings.TrimPrefix(url, best.Alias)
}
