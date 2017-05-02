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
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/majewsky/gofu/pkg/util"

	yaml "gopkg.in/yaml.v2"
)

//Index represents the contents of the index file.
type Index struct {
	Repos []*Repo `yaml:"repos"`
}

func indexPath() string {
	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		util.FatalIfError(errors.New("$HOME is not set (rtree needs the HOME variable to locate its index file)"))
	}
	return filepath.Join(homeDir, ".rtree/index.yaml")
}

//ReadIndex reads the index file.
func ReadIndex() *Index {
	//read contents of index file
	path := indexPath()
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Index{Repos: nil}
		}
		util.FatalIfError(err)
	}

	//deserialize YAML
	var index Index
	util.FatalIfError(yaml.Unmarshal(buf, &index))

	//validate YAML
	valid := true
	for idx, repo := range index.Repos {
		if repo.CheckoutPath == "" {
			util.ShowError(fmt.Errorf("missing \"repos[%d].path\"", idx))
			valid = false
		}
		if len(repo.Remotes) == 0 {
			util.ShowError(fmt.Errorf("missing \"repos[%d].remotes\"", idx))
			valid = false
		}
		for idx2, remote := range repo.Remotes {
			switch {
			case remote.Name == "":
				util.ShowError(fmt.Errorf("missing \"repos[%d].remotes[%d].name\"", idx, idx2))
				valid = false
			case remote.URL == "":
				util.ShowError(fmt.Errorf("missing \"repos[%d].remotes[%d].url\"", idx, idx2))
				valid = false
			}
		}
	}

	if !valid {
		util.FatalIfError(errors.New("index file is corrupted; see errors above"))
	}

	sort.Sort(reposByAbsPath(index.Repos))
	return &index
}

type reposByAbsPath []*Repo

func (r reposByAbsPath) Len() int           { return len(r) }
func (r reposByAbsPath) Less(i, j int) bool { return r[i].AbsolutePath() < r[j].AbsolutePath() }
func (r reposByAbsPath) Swap(i, j int)      { r[i], r[j] = r[j], r[i] }

//Write writes the index file to disk.
func (i *Index) Write() {
	buf, err := yaml.Marshal(i)
	util.FatalIfError(err)
	path := indexPath()
	util.FatalIfError(os.MkdirAll(filepath.Dir(path), 0755))
	util.FatalIfError(ioutil.WriteFile(path, buf, 0644))
}

//InteractiveRebuild implements the `rtree index` subcommand.
func (i *Index) InteractiveRebuild() error {
	//check if existing index entries are still checked out
	existingRepos := make(map[string]*Repo)
	var newRepos []*Repo
	for _, repo := range i.Repos {
		gitDirPath := filepath.Join(repo.AbsolutePath(), ".git")
		fi, err := os.Stat(gitDirPath)
		if err == nil {
			if fi.IsDir() {
				//everything okay with this repo
				existingRepos[repo.CheckoutPath] = repo
				newRepos = append(newRepos, repo)
				continue
			}
			return fmt.Errorf("%s is not a directory: I'm seriously confused", gitDirPath)
		}
		if err != nil && !os.IsNotExist(err) {
			return err
		}

		//repo has been deleted - ask what to do
		fmt.Printf("repository %s has been deleted\n", filepath.Join(RootPath, repo.CheckoutPath))

		var remoteURLs []string
		for _, remote := range repo.Remotes {
			if remote.Name == "origin" {
				remoteURLs = []string{remote.URL}
				break
			}
			remoteURLs = append(remoteURLs, remote.URL)
		}

		var choice string
		if len(remoteURLs) == 0 {
			choice = util.Prompt(os.Stdout,
				"no remote to restore from; (d)elete from index or (s)kip?",
				[]string{"d", "s"},
			)
		} else {
			choice = util.Prompt(os.Stdout,
				fmt.Sprintf("(r)estore from %s, (d)elete from index, or (s)kip?", strings.Join(remoteURLs, " and ")),
				[]string{"r", "d", "s"},
			)
		}

		switch choice {
		case "r":
			err := repo.Checkout()
			if err != nil {
				return err
			}
			newRepos = append(newRepos, repo)
		case "d":
			continue
		case "s":
			newRepos = append(newRepos, repo)
		}
	}

	//index new repos
	err := ForeachPhysicalRepo(func(newRepo Repo) error {
		repo, exists := existingRepos[newRepo.CheckoutPath]
		if exists {
			//update the existing index entry with the new remotes
			repo.Remotes = newRepo.Remotes
		} else {
			newRepos = append(newRepos, &newRepo)
		}
		return nil
	})
	if err != nil {
		return err
	}

	i.Repos = newRepos
	return nil
}

var tenLetters = []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}

//InteractiveFindRepo locates the repo with the given remote if it exists on
//disk or (if allowClone is set) clones it and adds it to the index. This
//is the meat of `rtree get`, and is also used by `rtree drop`.
func (i *Index) InteractiveFindRepo(remoteURL string, allowClone bool) *Repo {
	//NOTE: This function prints *everything* on stderr, because stdout is
	//reserved for the result path during `rtree get`.

	expandedRemoteURL := ExpandRemoteURL(remoteURL)
	basename := path.Base(expandedRemoteURL)

	//is this remote already checked out directly? also look for repos with the
	//same basename that could be forks
	var candidates []*Repo
	for _, repo := range i.Repos {
		isCandidate := false
		for _, remote := range repo.Remotes {
			otherExpandedRemoteURL := ExpandRemoteURL(remote.URL)
			if expandedRemoteURL == otherExpandedRemoteURL {
				return repo
			}
			if basename == path.Base(otherExpandedRemoteURL) {
				isCandidate = true
			}
		}
		if isCandidate {
			candidates = append(candidates, repo)
		}
	}

	//double-check if the repo is already checked out, but we didn't notice it yet
	newRepo := NewRepoFromRemoteURL(remoteURL)
	if newRepo.ExistsOnDisk() {
		util.FatalIfError(fmt.Errorf(
			"%s already exists (if there is a repo there, try `rtree index`)",
			newRepo.AbsolutePath(),
		))
	}

	if !allowClone {
		return nil
	}

	//if no fork candidates found, clone as new repo
	if len(candidates) == 0 {
		util.FatalIfError(newRepo.Checkout())
		i.Repos = append(i.Repos, &newRepo)
		i.Write()
		return &newRepo
	}

	//if we found fork candidates, ask the user to match the repo with a fork
	//candidate (or confirm that the repo shall be cloned fresh)
	if len(candidates) > 10 {
		candidates = candidates[:10]
	}
	prompt := "Found possible fork candidates.\n"
	for idx, repo := range candidates {
		prompt += fmt.Sprintf("\t(%s) add as remote to %s\n", tenLetters[idx], repo.AbsolutePath())
	}
	prompt += fmt.Sprintf("\t(x) clone to %s\nSelect action:", newRepo.AbsolutePath())
	choices := append([]string{"x"}, tenLetters[:len(candidates)]...)
	choice := util.Prompt(os.Stderr, prompt, choices)

	if choice == "x" {
		util.FatalIfError(newRepo.Checkout())
		i.Repos = append(i.Repos, &newRepo)
		i.Write()
		return &newRepo
	}

	//find the repo selected by the user
	var target *Repo
	for idx, str := range choices {
		if choice == str {
			target = candidates[idx-1]
		}
	}

	//report the existing remotes, and ask for the name of the new remote
	fmt.Fprintln(os.Stderr, "Existing remotes:")
	for _, remote := range target.Remotes {
		fmt.Fprintf(os.Stderr, "\t(%s) %s\n", remote.Name, remote.URL)
	}
	fmt.Fprintf(os.Stderr, "Enter remote name for %s: ", remoteURL)
	remoteName := util.ReadLine()

	cmd := exec.Command("git", "remote", "add", remoteName, remoteURL)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	cmd.Dir = target.AbsolutePath()
	util.FatalIfError(cmd.Run())

	cmd = exec.Command("git", "remote", "update", remoteName)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	cmd.Dir = target.AbsolutePath()
	util.FatalIfError(cmd.Run())

	target.Remotes = append(target.Remotes, Remote{
		Name: remoteName,
		URL:  remoteURL,
	})
	i.Write()
	return target
}
