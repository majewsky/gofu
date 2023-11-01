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
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/majewsky/gofu/internal/cli"

	yaml "gopkg.in/yaml.v2"
)

// Index represents the contents of the index file.
type Index struct {
	Repos []*Repo `yaml:"repos"`
}

// ReadIndex reads the index file.
func ReadIndex() (*Index, []error) {
	//read contents of index file
	buf, err := os.ReadFile(IndexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &Index{Repos: nil}, nil
		}
		return nil, []error{err}
	}

	//deserialize YAML
	var index Index
	err = yaml.Unmarshal(buf, &index)
	if err != nil {
		return nil, []error{err}
	}

	//validate YAML
	var errs []error
	missing := func(key string, args ...interface{}) {
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
		for idx2, remote := range repo.Remotes {
			switch {
			case remote.Name == "":
				missing("repos[%d].remotes[%d].name", idx, idx2)
			case remote.URL == "":
				missing("repos[%d].remotes[%d].url", idx, idx2)
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
	buf, err := yaml.Marshal(i)
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
				repo, err := handleUpdateRemotes(*repo)
				if err != nil {
					return err
				}

				//everything okay with this repo
				newRepos = append(newRepos, &repo)
				continue
			}
			return fmt.Errorf("expected repository at %s, but is not a directory or file", repo.GitDirPath())
		case !os.IsNotExist(err):
			return err
		}

		// repo has been deleted - ask what to do
		repo, err := handleDeleteRepo(*repo)
		if err != nil {
			return err
		}
		// repo got deleted
		if repo == nil {
			continue
		}

		newRepos = append(newRepos, repo)
	}

	existingRepos := make(map[string]*Repo)
	for _, repo := range newRepos {
		existingRepos[repo.CheckoutPath] = repo
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

func handleUpdateRemotes(repo Repo) (Repo, error) {
	currentRemotes, err := collectRemotesFromAbsolutePath(repo.AbsolutePath())
	if err != nil {
		return Repo{}, err
	}

	var (
		newRemotes []Remote
		toUpdate   []string
	)

configRemotes:
	for _, remote := range repo.Remotes {
		// remote in index exists in repo
		for _, currentRemote := range currentRemotes {
			if currentRemote.URL.CompactURL() == remote.URL.CompactURL() {
				// add URL matching remote as the name might have changed
				// e.g. when forking from upstream and renaming the old origin to upstream
				newRemotes = append(newRemotes, currentRemote)
				continue configRemotes
			}
		}

		// ask the user what to do with remotes not existing in checked out repo
		selection, err := cli.Interface.Query(
			fmt.Sprintf(`repository "%s" remote "%s" has been deleted`, repo.AbsolutePath(), remote.Name),
			cli.Choice{Return: "r", Shortcut: 'r', Text: "restore from " + remote.URL.CompactURL()},
			cli.Choice{Return: "d", Shortcut: 'd', Text: "delete from index"},
			cli.Choice{Return: "s", Shortcut: 's', Text: "skip"},
		)
		if err != nil {
			return Repo{}, err
		}

		switch selection {
		case "r":
			err := repo.RunGitCommand("remote", "add", remote.Name, remote.URL.CompactURL())
			if err != nil {
				return Repo{}, err
			}
			toUpdate = append(toUpdate, remote.Name)
			newRemotes = append(newRemotes, remote)
		case "d":
		case "s":
		}
	}

	if len(toUpdate) > 0 {
		err = repo.RunGitCommand(append([]string{"remote", "update"}, toUpdate...)...)
		if err != nil {
			return Repo{}, err
		}
	}

	// add all remotes in checkout to index
currentRemotes:
	for _, currentRemote := range currentRemotes {
		for _, newRemote := range newRemotes {
			if newRemote == currentRemote {
				continue currentRemotes
			}
		}

		newRemotes = append(newRemotes, currentRemote)
	}

	repo.Remotes = newRemotes
	return repo, nil
}

func handleDeleteRepo(repo Repo) (*Repo, error) {
	var remoteURLs []string
	for _, remote := range repo.Remotes {
		if remote.Name == "origin" {
			remoteURLs = []string{remote.URL.CompactURL()}
			break
		}
		remoteURLs = append(remoteURLs, remote.URL.CompactURL())
	}

	var (
		err       error
		selection string
	)
	if len(remoteURLs) == 0 {
		selection, err = cli.Interface.Query(
			fmt.Sprintf(`repository "%s" has been deleted; no remote to restore from`, repo.AbsolutePath()),
			cli.Choice{Return: "d", Shortcut: 'd', Text: "delete from index"},
			cli.Choice{Return: "s", Shortcut: 's', Text: "skip"},
		)
	} else {
		selection, err = cli.Interface.Query(
			fmt.Sprintf(`repository "%s" has been deleted`, repo.AbsolutePath()),
			cli.Choice{Return: "r", Shortcut: 'r', Text: "restore from " + strings.Join(remoteURLs, " and ")},
			cli.Choice{Return: "d", Shortcut: 'd', Text: "delete from index"},
			cli.Choice{Return: "s", Shortcut: 's', Text: "skip"},
		)
	}
	if err != nil {
		return nil, err
	}

	switch selection {
	case "r":
		err := repo.Checkout()
		if err != nil {
			return nil, err
		}
	case "d":
		return nil, nil
	case "s":
	}

	return &repo, nil
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
			// be flexible about .git ending in remote
			if remoteURL == remote.URL || remoteURL+".git" == remote.URL || remoteURL == remote.URL+".git" {
				return repo, nil
			}
			if basename == path.Base(remote.URL.CanonicalURL()) {
				isCandidate = true
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
	prompt := "Existing remotes:\n"
	for _, remote := range target.Remotes {
		prompt += fmt.Sprintf("\t(%s) %s\n", remote.Name, remote.URL.CompactURL())
	}
	prompt += fmt.Sprintf("Enter remote name for %s:", remoteURL)
	remoteName, err := cli.Interface.ReadLine(prompt)
	if err != nil {
		return nil, err
	}

	err = target.RunGitCommand("remote", "add", remoteName, remoteURL.CompactURL())
	if err != nil {
		return nil, err
	}
	err = target.RunGitCommand("remote", "update", remoteName)
	if err != nil {
		return nil, err
	}

	target.Remotes = append(target.Remotes, Remote{
		Name: remoteName,
		URL:  remoteURL,
	})
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
	repo, err := NewRepoFromAbsolutePath(dirPath)
	if err != nil {
		return err
	}

	//repo must be outside $GOPATH/src
	if !strings.HasPrefix(repo.CheckoutPath, "../") {
		return fmt.Errorf("%s is already inside GOPATH", dirPath)
	}

	//select the remote which determines the checkout path
	choices := make([]cli.Choice, len(repo.Remotes))
	var checkoutPath string
	for idx, remote := range repo.Remotes {
		thisPath, err := remote.URL.CheckoutPath()
		if err != nil {
			return err
		}
		if remote.Name == "origin" {
			//prefer "origin" over everything else
			checkoutPath = thisPath
			break
		}
		choices[idx] = cli.Choice{Return: thisPath, Text: thisPath}
	}

	//cannot decide myself -> let the user select
	if checkoutPath == "" {
		if len(choices) == 0 {
			return errors.New("repo has no remotes")
		}

		question := fmt.Sprintf(`Repo has multiple remotes. Where to put below "%s"?`, RootPath)
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
