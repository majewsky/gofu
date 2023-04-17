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

package rtree

import (
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
)

// RemoteURL is the URL of a remote of a Git repository.
type RemoteURL string

// ParseRemoteURL parses the given remote URL by substituting aliases defined in
// the system-wide and user-global Git config. For example, with
//
//	$ cat /etc/gitconfig
//	[url "git://github.com/"]
//	insteadOf = gh:
//
// and the input "gh:foo/bar", the result has a canonical URL of
// "git://github.com/foo/bar".
func ParseRemoteURL(input string) RemoteURL {
	var best *RemoteAlias
	for _, current := range RemoteAliases {
		if strings.HasPrefix(input, current.Alias) {
			if best == nil || len(best.Alias) < len(current.Alias) {
				best = current
			}
		}
	}
	if best == nil {
		return RemoteURL(input)
	}
	return RemoteURL(best.Replacement + strings.TrimPrefix(input, best.Alias))
}

// CanonicalURL returns the URL where the remote will be fetched from.
func (u RemoteURL) CanonicalURL() string {
	return string(u)
}

// CompactURL returns the most compact representation of this remote URL,
// obtained by substituting the longest matching alias defined in the
// system-wide or user-global Git config. This function is mostly the reverse of
// ParseRemoteURL().
func (u RemoteURL) CompactURL() string {
	var best *RemoteAlias
	for _, current := range RemoteAliases {
		if strings.HasPrefix(string(u), current.Replacement) {
			if best == nil || len(best.Replacement) < len(current.Replacement) {
				best = current
			}
		}
	}
	if best == nil {
		return string(u)
	}
	return best.Alias + strings.TrimPrefix(string(u), best.Replacement)
}

// This regex recognizes the scp-like syntax for git remotes
// (i.e. "[user@]example.org:path/to/repo") as specified by the "GIT URLS"
// section of man:git-clone(1).
var scpSyntaxRx = regexp.MustCompile(`^(?:[^/@:]+@)?([^/:]+\.[^/:]+):(.+)$`)

// CheckoutPath derives the checkout path for a remote URL.
//
//	RemoteURL("https://example.org/foo/bar") -> "example.org/foo/bar"
//	RemoteURL("git@example.org:foo/bar.git") -> "example.org/foo/bar"
func (u RemoteURL) CheckoutPath() (string, error) {
	stripped := strings.TrimSuffix(u.CanonicalURL(), ".git")

	match := scpSyntaxRx.FindStringSubmatch(stripped)
	if match != nil {
		//match[1] is the hostname, match[2] is the path to the repo
		return filepath.Join(match[1], match[2]), nil
	}

	parsed, err := url.Parse(stripped)
	if err != nil {
		return "", err
	}
	return filepath.Join(parsed.Hostname(), parsed.Path), nil
}

// MarshalYAML implements the yaml.Marshaler interface.
func (u RemoteURL) MarshalYAML() (interface{}, error) {
	//store URLs in the index in the compact format
	return u.CompactURL(), nil
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (u *RemoteURL) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	err := unmarshal(&s)
	if err == nil {
		*u = ParseRemoteURL(s)
	}
	return err
}
