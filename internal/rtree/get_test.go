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
	"fmt"
	"path/filepath"
	"testing"
)

var testIndexWithTwoRepos = Index{
	Repos: []*Repo{
		{
			CheckoutPath: "github.com/foo/bar",
			Remotes: []Remote{
				{Name: "origin", URL: "https://github.com/foo/bar"},
			},
		},
		{
			CheckoutPath: "github.com/git/git",
			Remotes: []Remote{
				{Name: "origin", URL: "gh:git/git"},
			},
		},
	},
}

func TestGetExistingRepo(t *testing.T) {
	Test{
		Args:         []string{"get", "gh:git/git"},
		Index:        testIndexWithTwoRepos,
		ExpectOutput: filepath.Join(RootPath, "/github.com/git/git") + "\n",
	}.Run(t)
}

func TestGetExistingRepoWithoutShortcut(t *testing.T) {
	Test{
		Args:         []string{"get", "https://github.com/git/git"},
		Index:        testIndexWithTwoRepos,
		ExpectOutput: filepath.Join(RootPath, "/github.com/git/git") + "\n",
	}.Run(t)
}

func TestGetNewRepo(t *testing.T) {
	target := filepath.Join(RootPath, "/github.com/another/repo")

	Test{
		Args:            []string{"get", "gh:another/repo"},
		Index:           testIndexWithTwoRepos,
		ExpectOutput:    target + "\n",
		ExpectExecution: Recorded("git clone gh:another/repo " + target),
		ExpectIndex: &Index{
			Repos: []*Repo{
				{
					CheckoutPath: "github.com/another/repo",
					Remotes: []Remote{
						{Name: "origin", URL: "gh:another/repo"},
					},
				},
				testIndexWithTwoRepos.Repos[0],
				testIndexWithTwoRepos.Repos[1],
			},
		},
	}.Run(t)
}

func TestGetNewForkAsRemote(t *testing.T) {
	target := filepath.Join(RootPath, "/github.com/git/git")
	Test{
		Args:         []string{"get", "https://example.com/git"},
		Index:        testIndexWithTwoRepos,
		Input:        fmt.Sprintf("add as remote to %s\nmyfork\n", target),
		ExpectOutput: target + "\n",
		ExpectError: fmt.Sprintf(
			"Found possible fork candidates. What to do? -> add as remote to %s\n"+
				"Existing remotes:\n\t(origin) gh:git/git\n"+
				"Enter remote name for https://example.com/git: myfork\n",
			target,
		),
		ExpectExecution: Recorded(
			"@"+target+" git remote add myfork https://example.com/git",
			"@"+target+" git remote update myfork",
		),
		ExpectIndex: &Index{
			Repos: []*Repo{
				testIndexWithTwoRepos.Repos[0],
				{
					CheckoutPath: "github.com/git/git",
					Remotes: []Remote{
						{Name: "origin", URL: "gh:git/git"},
						{Name: "myfork", URL: "https://example.com/git"},
					},
				},
			},
		},
	}.Run(t)
}

func TestGetNewForkAsSeparate(t *testing.T) {
	target := filepath.Join(RootPath, "/example.com/git")
	Test{
		Args:         []string{"get", "https://example.com/git"},
		Index:        testIndexWithTwoRepos,
		Input:        fmt.Sprintf("clone to %s\n", target),
		ExpectOutput: target + "\n",
		ExpectError: fmt.Sprintf(
			"Found possible fork candidates. What to do? -> clone to %s\n",
			target,
		),
		ExpectExecution: Recorded("git clone https://example.com/git " + target),
		ExpectIndex: &Index{
			Repos: []*Repo{
				{
					CheckoutPath: "example.com/git",
					Remotes: []Remote{
						{Name: "origin", URL: "https://example.com/git"},
					},
				},
				testIndexWithTwoRepos.Repos[0],
				testIndexWithTwoRepos.Repos[1],
			},
		},
	}.Run(t)
}
