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

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

//Returns empty string if `path` is not inside a Git repo.
func findRepoRootPath(path string) string {
	_, err := os.Stat(filepath.Join(path, ".git"))
	if err == nil {
		return path
	}
	if path == "/" {
		return ""
	}
	return findRepoRootPath(filepath.Dir(path))
}

func getRepoStatusField(repoRootPath string) string {
	if repoRootPath == "" {
		return ""
	}

	bytes, err := ioutil.ReadFile(filepath.Join(repoRootPath, ".git/HEAD"))
	if err != nil {
		handleError(err)
		return withType("git", withColor("1;41", "unknown"))
	}
	refSpec := strings.TrimSpace(string(bytes))

	//is current HEAD detached?
	if !strings.HasPrefix(refSpec, "ref: refs/") {
		return formatRepoStatusField(withColor("1;41", "detached"), refSpec)
	}

	//current HEAD is a ref
	refSpec = strings.TrimPrefix(refSpec, "ref: ")
	refSpecDisplay := strings.TrimPrefix(refSpec, "refs/")
	refSpecDisplay = strings.TrimPrefix(refSpecDisplay, "heads/")

	//read file corresponding to refspec to find commit
	bytes, err = ioutil.ReadFile(filepath.Join(repoRootPath, ".git", refSpec))
	commitID := strings.TrimSpace(string(bytes))
	if err != nil {
		if os.IsNotExist(err) {
			commitID = withColor("37", "blank")
		} else {
			handleError(err)
			commitID = withColor("1;41", "unknown")
		}
	}

	return formatRepoStatusField(refSpecDisplay, commitID)
}

func formatRepoStatusField(refSpec, commitID string) string {
	//shorten plain commit IDs from 40 to 10 bytes
	if len(commitID) == 40 && !strings.Contains(commitID, "\x1B") {
		commitID = commitID[0:10]
	}
	return withType("git", refSpec+"/"+commitID)
}
