// SPDX-FileCopyrightText: 2017 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: GPL-3.0-only

package rtree

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"git.xyrillian.de/gofu/internal/cli"
)

// RemoteAlias describes an alias that can be used in a Git remote URL (as
// defined by the "url.<base>.insteadOf" directive in man:git-config(1)).
type RemoteAlias struct {
	Alias       string
	Replacement string
}

// IndexPath is where the index file is stored.
var IndexPath string

// OldIndexPath is where the old index file was stored.
// This is only used to detect if a system has not been upgraded to the new index format yet.
// TODO: remove this after some time
var OldIndexPath string

// RootPath is the directory below which all repositories are located. Its value
// is $GOPATH/src to match the repository layout created by `go get`.
var RootPath string

// RemoteAliases is the list of remote aliases that is used by ExpandRemoteURL().
var RemoteAliases []*RemoteAlias

// Init initializes the global variables of this package to their standard
// values, unless they are already populated. Unit tests shall set IndexPath,
// RootPath etc. before calling Exec(), such that this function becomes a no-op
// when called by Exec().
//
// Returns false if initialization failed.
func Init() bool {
	ok := true //until shown otherwise

	if IndexPath == "" {
		homeDir := os.Getenv("HOME")
		if homeDir == "" {
			cli.Interface.ShowError("$HOME is not set (rtree needs the HOME variable to locate its index file)")
			ok = false //but keep going to report all errors at once
		} else {
			IndexPath = filepath.Join(homeDir, ".config/rtree/index.json")
			OldIndexPath = filepath.Join(homeDir, ".rtree/index.yaml")
		}
	}

	if RootPath == "" {
		gopath := os.Getenv("GOPATH")
		if gopath == "" {
			cli.Interface.ShowError("$GOPATH is not set (rtree needs the GOPATH variable to know where to look for and place repos)")
			ok = false //but keep going to report all errors at once
		} else {
			RootPath = filepath.Join(gopath, "src")
		}
	}

	if RemoteAliases == nil {
		out, err := cli.Interface.CaptureStdout(cli.Command{
			Program: []string{"git", "config", "--global", "-l"},
		})
		if err != nil {
			cli.Interface.ShowError(err.Error())
			ok = false //but keep going to report all errors at once
		}

		rx := regexp.MustCompile(`^url\.([^=]+)\.insteadof=(.+)$`)
		for line := range strings.SplitSeq(out, "\n") {
			match := rx.FindStringSubmatch(line)
			if match == nil {
				continue
			}
			RemoteAliases = append(RemoteAliases, &RemoteAlias{
				Alias:       match[2],
				Replacement: match[1],
			})
		}
	}

	return ok
}
