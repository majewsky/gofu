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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

func main() {
	if len(os.Args) == 1 {
		usageAndExit()
	}
	switch os.Args[1] {
	case "get":
		panic("unimplemented")
	case "drop":
		panic("unimplemented")
	case "index":
		if len(os.Args) != 2 {
			usageAndExit()
		}
		commandIndex()
	case "repos":
		if len(os.Args) != 2 {
			usageAndExit()
		}
		commandRepos()
	case "remotes":
		if len(os.Args) != 2 {
			usageAndExit()
		}
		commandRemotes()
	case "import":
		panic("unimplemented")
	case "each":
		if len(os.Args) < 3 {
			usageAndExit()
		}
		commandEach(os.Args[2], os.Args[3:])
	default:
		usageAndExit()
	}
}

func usageAndExit() {
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, "  rtree [get|drop] <url>")
	fmt.Fprintln(os.Stderr, "  rtree [index|repos|remotes]")
	fmt.Fprintln(os.Stderr, "  rtree import <path>")
	fmt.Fprintln(os.Stderr, "  rtree each <command>")
	os.Exit(1)
}

func commandIndex() {
	oldIndex := ReadIndex()
	var newIndex Index

	//check if existing index entries are still checked out
	existingRepos := make(map[string]*Repo)
	for _, repo := range oldIndex.Repos {
		gitDirPath := filepath.Join(repo.AbsolutePath(), ".git")
		fi, err := os.Stat(gitDirPath)
		if err == nil {
			if fi.IsDir() {
				//everything okay with this repo
				existingRepos[repo.CheckoutPath] = repo
				newIndex.Repos = append(newIndex.Repos, repo)
				continue
			}
			FatalIfError(fmt.Errorf("%s is not a directory: I'm seriously confused", gitDirPath))
		}
		if !os.IsNotExist(err) {
			FatalIfError(err)
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
			FatalIfError(repo.Checkout())
			newIndex.Repos = append(newIndex.Repos, repo)
		case "d":
			continue
		case "s":
			newIndex.Repos = append(newIndex.Repos, repo)
		}
	}

	//index new repos
	FatalIfError(ForeachPhysicalRepo(func(newRepo Repo) error {
		repo, exists := existingRepos[newRepo.CheckoutPath]
		if exists {
			//update the existing index entry with the new remotes
			repo.Remotes = newRepo.Remotes
		} else {
			newIndex.Repos = append(newIndex.Repos, &newRepo)
		}
		return nil
	}))

	newIndex.Write()
}

func commandRepos() {
	index := ReadIndex()
	var items []string
	for _, repo := range index.Repos {
		items = append(items, repo.CheckoutPath)
	}
	ShowSorted(items)
}

func commandRemotes() {
	index := ReadIndex()
	var items []string
	for _, repo := range index.Repos {
		for _, remote := range repo.Remotes {
			items = append(items, remote.URL)
		}
	}
	ShowSorted(items)
}

func commandEach(command string, args []string) {
	index := ReadIndex()

	var paths []string
	for _, repo := range index.Repos {
		paths = append(paths, repo.AbsolutePath())
	}
	sort.Strings(paths)

	hadErrors := false
	for _, path := range paths {
		fmt.Fprintf(os.Stdout, "\x1B[1;36m>> \x1B[0;36m%s\x1B[0m\n", path)
		cmd := exec.Command(command, args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Dir = path
		err := cmd.Run()
		if err != nil {
			fmt.Fprintf(os.Stderr, "\x1B[1;31m!! \x1B[0;31m%s\x1B[0m\n", err.Error())
			hadErrors = true
		}
	}

	if hadErrors {
		os.Exit(1)
	}
}
