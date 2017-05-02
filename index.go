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
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	yaml "gopkg.in/yaml.v2"
)

//Index represents the contents of the index file.
type Index struct {
	Repos []*Repo `yaml:"repos"`
}

func indexPath() string {
	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		FatalIfError(errors.New("$HOME is not set (rtree needs the HOME variable to locate its index file)"))
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
		FatalIfError(err)
	}

	//deserialize YAML
	var index Index
	FatalIfError(yaml.Unmarshal(buf, &index))

	//validate YAML
	valid := true
	for idx, repo := range index.Repos {
		if repo.CheckoutPath == "" {
			ShowError(fmt.Errorf("missing \"repos[%d].path\"", idx))
			valid = false
		}
		if len(repo.Remotes) == 0 {
			ShowError(fmt.Errorf("missing \"repos[%d].remotes\"", idx))
			valid = false
		}
		for idx2, remote := range repo.Remotes {
			switch {
			case remote.Name == "":
				ShowError(fmt.Errorf("missing \"repos[%d].remotes[%d].name\"", idx, idx2))
				valid = false
			case remote.URL == "":
				ShowError(fmt.Errorf("missing \"repos[%d].remotes[%d].url\"", idx, idx2))
				valid = false
			}
		}
	}

	if !valid {
		FatalIfError(errors.New("index file is corrupted; see errors above"))
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
	FatalIfError(err)
	path := indexPath()
	FatalIfError(os.MkdirAll(filepath.Dir(path), 0755))
	FatalIfError(ioutil.WriteFile(path, buf, 0644))
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
			choice = Prompt(
				"no remote to restore from; (d)elete from index or (s)kip?",
				[]string{"d", "s"},
			)
		} else {
			choice = Prompt(
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
