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
	"path/filepath"
	"testing"
)

func TestGetRepoFromIndex(t *testing.T) {
	idx := Index{
		Repos: []*Repo{
			{
				CheckoutPath: "github.com/git/git",
				Remotes: []Remote{
					{Name: "origin", URL: "gh:git/git"},
				},
			},
		},
	}
	Test{
		Args:         []string{"get", "gh:git/git"},
		Index:        idx,
		ExpectOutput: filepath.Join(RootPath, "/github.com/git/git") + "\n",
	}.Run(t, "TestGetRepoFromIndex")
}
