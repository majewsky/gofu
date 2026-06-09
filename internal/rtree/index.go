// SPDX-FileCopyrightText: 2017 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: GPL-3.0-only

package rtree

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"git.xyrillian.de/gofu/internal/cli"
)

// Index represents the contents of the index.json file.
type Index struct {
	Repos []*Repo `json:"repos"`
}

// ReadIndex reads the index file.
func ReadIndex() (*Index, []error) {
	//read contents of index file
	buf, err := os.ReadFile(IndexPath)
	if err != nil {
		if os.IsNotExist(err) {
			_, err := os.Stat(OldIndexPath)
			if !os.IsNotExist(err) {
				err = fmt.Errorf(
					"old index format detected: upgrade to the new index format with this command:\n\t"+
						`yq -o json < %s | jq --sort-keys '{ repos: .repos | map({ path, remotes: .remotes | map({ key: .name, value: { urls: [.url] }}) | from_entries }) }' > %s`,
					OldIndexPath, IndexPath,
				)
				return nil, []error{err}
			}
			return &Index{Repos: nil}, nil
		}
		return nil, []error{err}
	}

	//deserialize JSON
	var index Index
	err = json.Unmarshal(buf, &index)
	if err != nil {
		return nil, []error{err}
	}
	//validate JSON
	var errs []error
	missing := func(key string, args ...any) {
		errs = append(errs, fmt.Errorf("read %s: missing \"%s\"",
			IndexPath, fmt.Sprintf(key, args...),
		))
	}
	for idx, repo := range index.Repos {
		if repo.CheckoutPath == "" {
			missing("repos[%d].path", idx)
		}
		if len(repo.Remotes) == 0 {
			missing("repos[%d].remotes", idx)
		}
		for remoteName, remote := range repo.Remotes {
			switch {
			case remoteName == "":
				errs = append(errs, fmt.Errorf("read %s: empty key in \"repos[%d].remotes\"",
					IndexPath, idx,
				))
			case len(remote.URLs) == 0:
				missing("repos[%d].remotes[%q].urls", idx, remoteName)
			}
		}
	}

	sort.Sort(reposByAbsPath(index.Repos))
	return &index, errs
}

type reposByAbsPath []*Repo

func (r reposByAbsPath) Len() int           { return len(r) }
func (r reposByAbsPath) Less(i, j int) bool { return r[i].AbsolutePath() < r[j].AbsolutePath() }
func (r reposByAbsPath) Swap(i, j int)      { r[i], r[j] = r[j], r[i] }

// Write writes the index file to disk.
func (i *Index) Write() error {
	sort.Sort(reposByAbsPath(i.Repos))
	buf, err := json.MarshalIndent(i, "", "  ")
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Dir(IndexPath), 0755)
	if err != nil {
		return err
	}
	err = os.WriteFile(IndexPath, buf, 0644)
	if err != nil {
		return err
	}

	//perform sanity check (TODO: do this instead when rebuilding the index)
	seen := make(map[string]bool)
	warned := make(map[string]bool)
	for _, repo := range i.Repos {
		if seen[repo.CheckoutPath] && !warned[repo.CheckoutPath] {
			cli.Interface.ShowWarning(
				fmt.Sprintf("repo %s appears multiple times in the index file!", repo.AbsolutePath()),
			)
			warned[repo.CheckoutPath] = true
		}
		seen[repo.CheckoutPath] = true
	}

	return nil
}

// Rebuild implements the `rtree index` subcommand.
func (i *Index) Rebuild() error {
	//check if existing index entries are still checked out
	var newRepos []*Repo
	for _, repo := range i.Repos {
		fi, err := os.Stat(repo.GitDirPath())
		switch {
		case err == nil:
			// in a normal repo .git is a directory but when the repo is a submodule of another repo
			// and the .git dir is absorbed then it is a file which contains the path to the real .git directory
			if fi.IsDir() || fi.Mode().IsRegular() {
				//everything okay with this repo
				newRepos = append(newRepos, repo)
				continue
			}
			return fmt.Errorf("expected repository at %s, but is not a directory or file", repo.GitDirPath())
		case !os.IsNotExist(err):
			return err
		}

		//repo has been deleted - ask what to do
		var remoteURLs []string
		if origin, ok := repo.Remotes["origin"]; ok {
			remoteURLs = origin.CompactURLs()
		} else {
			for _, remote := range repo.Remotes {
				remoteURLs = append(remoteURLs, remote.CompactURLs()...)
			}
		}

		repoPath := filepath.Join(RootPath, repo.CheckoutPath)
		var selection string
		if len(remoteURLs) == 0 {
			selection, err = cli.Interface.Query(
				fmt.Sprintf("repository %s has been deleted; no remote to restore from", repoPath),
				cli.Choice{Return: "d", Shortcut: 'd', Text: "delete from index"},
				cli.Choice{Return: "s", Shortcut: 's', Text: "skip"},
			)
		} else {
			selection, err = cli.Interface.Query(
				fmt.Sprintf("repository %s has been deleted", repoPath),
				cli.Choice{Return: "r", Shortcut: 'r', Text: "restore from " + strings.Join(remoteURLs, " and ")},
				cli.Choice{Return: "d", Shortcut: 'd', Text: "delete from index"},
				cli.Choice{Return: "s", Shortcut: 's', Text: "skip"},
			)
		}
		if err != nil {
			return err
		}

		switch selection {
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

	existingRepos := make(map[string]*Repo)
	for _, repo := range newRepos {
		existingRepos[repo.CheckoutPath] = repo
	}

	//index new repos
	err := ForeachPhysicalRepo(func(newRepo Repo) error {
		repo, exists := existingRepos[newRepo.CheckoutPath]

		// if a repo has no remotes, repo is nil which rtree cannot parse back and doesn't make sense to add anyway
		if repo == nil || repo.Remotes == nil {
			fmt.Printf("repository %s has no remotes; skipping", filepath.Join(RootPath, newRepo.CheckoutPath))
			return nil
		}

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

// FindRepo locates the repo with the given remote if it exists on disk or (if
// allowClone is set) clones it and adds it to the index. This is the meat of
// `rtree get`, and is also used by `rtree drop`.
func (i *Index) FindRepo(rawRemoteURL string, allowClone bool) (*Repo, error) {
	//make sure that stdout is not used for prompts
	cli.Interface.StdoutProtected = true

	remoteURL := ParseRemoteURL(rawRemoteURL)
	basename := path.Base(remoteURL.CanonicalURL())

	//is this remote already checked out directly? also look for repos with the
	//same basename that could be forks
	var candidates []*Repo
	for _, repo := range i.Repos {
		isCandidate := false
		for _, remote := range repo.Remotes {
			for _, url := range remote.URLs {
				// be flexible about .git ending in remote
				if remoteURL == url || remoteURL+".git" == url || remoteURL == url+".git" {
					return repo, nil
				}
				if basename == path.Base(url.CanonicalURL()) {
					isCandidate = true
				}
			}
		}
		if isCandidate {
			candidates = append(candidates, repo)
		}
	}

	//double-check if the repo is already checked out, but we didn't notice it yet
	newRepo, err := NewRepoFromRemoteURL(remoteURL)
	if err != nil {
		return nil, err
	}
	_, err = os.Stat(newRepo.AbsolutePath())
	switch {
	case err == nil:
		return nil, fmt.Errorf(
			"%s already exists (if there is a repo there, try `rtree index`)",
			newRepo.AbsolutePath(),
		)
	case !os.IsNotExist(err):
		return nil, err
	}

	if !allowClone {
		return nil, errors.New("no such remote in index (you can validate the index with `rtree index`)")
	}

	//if no fork candidates found, clone as new repo
	if len(candidates) == 0 {
		err := newRepo.Checkout()
		if err != nil {
			return nil, err
		}
		i.Repos = append(i.Repos, &newRepo)
		err = i.Write()
		return &newRepo, err
	}

	//if we found fork candidates, ask the user to match the repo with a fork
	//candidate (or confirm that the repo shall be cloned fresh)
	if len(candidates) > 10 {
		candidates = candidates[:10]
	}
	choices := make([]cli.Choice, len(candidates)+1)
	for idx, repo := range candidates {
		choices[idx] = cli.Choice{Text: "add as remote to " + repo.AbsolutePath(), Return: repo.CheckoutPath}
	}
	choices[len(candidates)] = cli.Choice{
		Return:   "clone",
		Shortcut: 'n',
		Text:     "clone to " + newRepo.AbsolutePath(),
	}
	selection, err := cli.Interface.Query("Found possible fork candidates. What to do?", choices...)
	if err != nil {
		return nil, err
	}

	if selection == "clone" {
		err := newRepo.Checkout()
		if err != nil {
			return nil, err
		}
		i.Repos = append(i.Repos, &newRepo)
		err = i.Write()
		return &newRepo, err
	}

	//find the repo selected by the user
	var target *Repo
	for _, repo := range candidates {
		if repo.CheckoutPath == selection {
			target = repo
			break
		}
	}

	//report the existing remotes, and ask for the name of the new remote
	var prompt strings.Builder
	prompt.WriteString("Existing remotes:\n")
	for remoteName, remote := range target.Remotes {
		fmt.Fprintf(&prompt, "\t(%s) %s\n", remoteName, strings.Join(remote.CompactURLs(), " "))
	}
	fmt.Fprintf(&prompt, "Enter remote name for %s:", remoteURL)
	remoteName, err := cli.Interface.ReadLine(prompt.String())
	if err != nil {
		return nil, err
	}

	err = cli.Interface.Run(cli.Command{
		Program: []string{"git", "remote", "add", remoteName, remoteURL.CompactURL()},
		WorkDir: target.AbsolutePath(),
	})
	if err != nil {
		return nil, err
	}

	err = cli.Interface.Run(cli.Command{
		Program: []string{"git", "remote", "update", remoteName},
		WorkDir: target.AbsolutePath(),
	})
	if err != nil {
		return nil, err
	}

	target.Remotes[remoteName] = Remote{
		URLs: []RemoteURL{remoteURL},
	}
	err = i.Write()
	return target, err
}

// ImportRepo moves the given repo into the rtree and adds it to the index.
func (i *Index) ImportRepo(dirPath string) error {
	//need to make dirPath absolute first
	dirPath, err := filepath.Abs(dirPath)
	if err != nil {
		return err
	}
	repo, err := NewRepoFromAbsolutePath(dirPath, true)
	if err != nil {
		return err
	}

	//repo must be outside $GOPATH/src
	if !strings.HasPrefix(repo.CheckoutPath, "../") {
		return fmt.Errorf("%s is already inside GOPATH", dirPath)
	}

	//select the remote which determines the checkout path
	choices := make([]cli.Choice, 0, len(repo.Remotes))
	var checkoutPath string
	for remoteName, remote := range repo.Remotes {
		// NOTE: This uses URLs[0] only because git fetches only from the first URL (the others are only for pushing).
		thisPath, err := remote.URLs[0].CheckoutPath()
		if err != nil {
			return err
		}
		if remoteName == "origin" {
			//prefer "origin" over everything else
			checkoutPath = thisPath
			break
		}
		choices = append(choices, cli.Choice{Return: thisPath, Text: thisPath})
	}

	//cannot decide myself -> let the user select
	if checkoutPath == "" {
		if len(choices) == 0 {
			return errors.New("repo has no remotes")
		}

		question := fmt.Sprintf("Repo has multiple remotes. Where to put below %s?", RootPath)
		checkoutPath, err = cli.Interface.Query(question, choices...)
		if err != nil {
			return err
		}
	}

	//double-check that there is no such repo in the rtree yet
	for _, other := range i.Repos {
		if other.CheckoutPath == checkoutPath {
			return errors.New("will not overwrite existing checkout at " + other.AbsolutePath())
		}
	}

	//do the move
	err = repo.Move(checkoutPath, true)
	if err != nil {
		return err
	}
	i.Repos = append(i.Repos, &repo)
	return nil
}

// DropRepo deletes the given repo from the rtree and removes it from the index.
func (i *Index) DropRepo(repo *Repo) error {
	err := repo.Exec("git", "status")
	if err != nil {
		return err
	}
	ok, err := cli.Interface.Confirm(">> Drop this repo?")
	if !ok || err != nil {
		return err
	}

	err = os.RemoveAll(repo.AbsolutePath())
	if err != nil {
		return err
	}

	reposNew := make([]*Repo, 0, len(i.Repos)-1)
	for _, r := range i.Repos {
		if r.CheckoutPath != repo.CheckoutPath {
			reposNew = append(reposNew, r)
		}
	}
	i.Repos = reposNew
	return i.Write()
}
