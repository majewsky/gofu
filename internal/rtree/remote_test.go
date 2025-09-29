// SPDX-FileCopyrightText: 2020 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: GPL-3.0-only

package rtree

import "testing"

var remoteAliasesForExpansionTest = []*RemoteAlias{
	{Alias: "gh:f:", Replacement: "https://github.com/foo/"},
	{Alias: "gh:", Replacement: "https://github.com/"},
	{Alias: "gh:b:", Replacement: "https://github.com/bar/"},
	{Alias: "gl:", Replacement: "git://gitlab.com/"},
	{Alias: "test:", Replacement: "test:test:"},
}

var testExpansions = map[string]RemoteURL{
	"gh:foo/bar":                 "https://github.com/foo/bar",
	"gh:f:bar":                   "https://github.com/foo/bar",
	"gh:b:foo":                   "https://github.com/bar/foo",
	"gh:gh:foo/bar":              "https://github.com/gh:foo/bar", //do not expand multiple times
	"test:foo":                   "test:test:foo",                 //do not expand recursively
	"https://github.com/foo/bar": "https://github.com/foo/bar",    //no expansion at all
}

func TestParseRemoteURL(t *testing.T) {
	RemoteAliases = remoteAliasesForExpansionTest
	for input, expected := range testExpansions {
		actual := ParseRemoteURL(input)
		if actual != expected {
			t.Errorf("expected %q to expand into %q, but got %q", input, expected, actual)
		}
	}
}

// Most of those are just reversed from `testExpansions`.
var testContractions = map[RemoteURL]string{
	"https://github.com/foo/bar":      "gh:f:bar",
	"https://github.com/bar/foo":      "gh:b:foo",
	"https://github.com/qux/foobar":   "gh:qux/foobar",
	"https://github.com/gh:foo/bar":   "gh:gh:foo/bar",
	"test:test:foo":                   "test:foo",
	"git://somewhereelse.com/foo/bar": "git://somewhereelse.com/foo/bar",
}

func TestCompactRemoteURL(t *testing.T) {
	RemoteAliases = remoteAliasesForExpansionTest
	for input, expected := range testContractions {
		actual := input.CompactURL()
		if actual != expected {
			t.Errorf("expected %q to contract into %q, but got %q", input, expected, actual)
		}
	}
}
