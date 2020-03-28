/*******************************************************************************
*
* Copyright 2020 Stefan Majewsky <majewsky@gmx.net>
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

import "testing"

var remoteAliasesForExpansionTest = []*RemoteAlias{
	{Alias: "gh:f:", Replacement: "https://github.com/foo/"},
	{Alias: "gh:", Replacement: "https://github.com/"},
	{Alias: "gh:b:", Replacement: "https://github.com/bar/"},
	{Alias: "gl:", Replacement: "git://gitlab.com/"},
	{Alias: "test:", Replacement: "test:test:"},
}

var testExpansions = map[string]string{
	"gh:foo/bar":                 "https://github.com/foo/bar",
	"gh:f:bar":                   "https://github.com/foo/bar",
	"gh:b:foo":                   "https://github.com/bar/foo",
	"gh:gh:foo/bar":              "https://github.com/gh:foo/bar", //do not expand multiple times
	"test:foo":                   "test:test:foo",                 //do not expand recursively
	"https://github.com/foo/bar": "https://github.com/foo/bar",    //no expansion at all
}

func TestExpandRemoteURL(t *testing.T) {
	RemoteAliases = remoteAliasesForExpansionTest
	for input, expected := range testExpansions {
		actual := ExpandRemoteURL(input)
		if actual != expected {
			t.Errorf("expected %q to expand into %q, but got %q", input, expected, actual)
		}
	}
}

//Most of those are just reversed from `testExpansions`.
var testContractions = map[string]string{
	"https://github.com/foo/bar":      "gh:f:bar",
	"https://github.com/bar/foo":      "gh:b:foo",
	"https://github.com/qux/foobar":   "gh:qux/foobar",
	"https://github.com/gh:foo/bar":   "gh:gh:foo/bar",
	"test:test:foo":                   "test:foo",
	"git://somewhereelse.com/foo/bar": "git://somewhereelse.com/foo/bar",
}

func TestContractRemoteURL(t *testing.T) {
	RemoteAliases = remoteAliasesForExpansionTest
	for input, expected := range testContractions {
		actual := ContractRemoteURL(input)
		if actual != expected {
			t.Errorf("expected %q to contract into %q, but got %q", input, expected, actual)
		}
	}
}
