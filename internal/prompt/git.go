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
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type gitRepo struct {
	RootPath string
	GitDir   string
}

// Returns two empty strings if `path` is not inside a Git repo.
func findRepo(path string) (*gitRepo, error) {
	//find .git directory or file
	gitEntry := filepath.Join(path, ".git")
	fi, err := os.Stat(gitEntry)
	switch {
	case err == nil:
		//found - continue below with further checks
	case !os.IsNotExist(err):
		return nil, err
	case path == "/":
		return nil, nil
	default:
		return findRepo(filepath.Dir(path))
	}

	//found .git - what is it?
	if fi.Mode().IsDir() {
		//normal case - .git is a directory
		return &gitRepo{RootPath: path, GitDir: gitEntry}, nil
	}

	//.git is a file (e.g. for submodules) - it contains a line like "gitdir: path/to/gitdir"
	bytes, err := os.ReadFile(gitEntry)
	if err != nil {
		return nil, err
	}
	for line := range strings.SplitSeq(string(bytes), "\n") {
		line = strings.TrimSpace(line)
		if gitDir, ok := strings.CutPrefix(line, "gitdir:"); ok {
			return &gitRepo{
				RootPath: path,
				GitDir:   filepath.Join(path, strings.TrimSpace(gitDir)),
			}, nil
		}
	}

	return nil, fmt.Errorf("read %s: missing gitdir directive", gitEntry)
}

func getRepoStatusField(repo *gitRepo) string {
	if repo == nil {
		return ""
	}

	bytes, err := os.ReadFile(filepath.Join(repo.GitDir, "HEAD"))
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

	//read file corresponding to refspec to find commit ID
	bytes, err = os.ReadFile(filepath.Join(repo.GitDir, refSpec))
	commitID := strings.TrimSpace(string(bytes))
	if err != nil {
		if os.IsNotExist(err) {
			commitID = tryReadFromPackedRefs(repo, refSpec)
			if commitID == "" {
				commitID = withColor("37", "blank")
			}
		} else {
			handleError(err)
			commitID = withColor("1;41", "unknown")
		}
	}

	return formatRepoStatusField(refSpecDisplay, commitID)
}

func tryReadFromPackedRefs(repo *gitRepo, refSpec string) string {
	bytes, err := os.ReadFile(filepath.Join(repo.GitDir, "packed-refs"))
	if err != nil {
		return ""
	}
	for line := range strings.SplitSeq(string(bytes), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == refSpec {
			return fields[0]
		}
	}
	return ""
}

func formatRepoStatusField(refSpec, commitID string) string {
	//shorten plain commit IDs from 40 to 10 bytes
	if len(commitID) == 40 && !strings.Contains(commitID, "\x1B") {
		commitID = commitID[0:10]
	}
	return withType("git", refSpec+"/"+commitID)
}
