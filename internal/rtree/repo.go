// SPDX-FileCopyrightText: 2017 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: GPL-3.0-only

package rtree

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/majewsky/gofu/internal/cli"
)

// Repo describes the entry for a repository in the index file.
type Repo struct {
	//CheckoutPath shall be relative to the RootPath.
	CheckoutPath string `json:"path"`
	//Remotes maps remote names (as noted in the .git/config of the repo) to
	//remote URLs (as they appear in the .git/config of the repo, i.e. possibly
	//abbreviated).
	Remotes map[string]Remote `json:"remotes"`
}

// Remote describes a remote that is configured in a Repo.
type Remote struct {
	URLs []RemoteURL `json:"urls"`
}

// AbsolutePath returns the absolute CheckoutPath of this repo.
func (r Repo) AbsolutePath() string {
	return filepath.Join(RootPath, r.CheckoutPath)
}

// GitDirPath returns the path of the .git directory of this repo.
func (r Repo) GitDirPath() string {
	return filepath.Join(r.AbsolutePath(), ".git")
}

// CompactURLs returns all URLs for this remote in their compact form.
func (r Remote) CompactURLs() []string {
	result := make([]string, len(r.URLs))
	for idx, url := range r.URLs {
		result[idx] = url.CompactURL()
	}
	return result
}

// NewRepoFromAbsolutePath initializes a Repo instance by scanning the existing
// checkout at the given path.
func NewRepoFromAbsolutePath(path string) (repo Repo, err error) {
	repo.CheckoutPath, err = filepath.Rel(RootPath, path)
	if err != nil {
		return
	}

	//list remotes
	out, err := cli.Interface.CaptureStdout(cli.Command{
		Program: []string{"git", "config", "-l"},
		WorkDir: path,
	})
	if err != nil {
		return
	}

	repo.Remotes = make(map[string]Remote)
	for line := range strings.SplitSeq(out, "\n") {
		match := remoteConfigRx.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		name, url := match[1], ParseRemoteURL(match[2])
		if remote, ok := repo.Remotes[name]; ok {
			remote.URLs = append(remote.URLs, url)
			repo.Remotes[name] = remote
		} else {
			repo.Remotes[name] = Remote{URLs: []RemoteURL{url}}
		}
	}
	return
}

// NewRepoFromRemoteURL initializes a Repo instance for checking out a remote
// for the first time. The checkout does not happen until Checkout() is called.
func NewRepoFromRemoteURL(remoteURL RemoteURL) (Repo, error) {
	checkoutPath, err := remoteURL.CheckoutPath()
	return Repo{
		CheckoutPath: checkoutPath,
		Remotes: map[string]Remote{
			"origin": {
				URLs: []RemoteURL{remoteURL},
			},
		},
	}, err
}

var remoteConfigRx = regexp.MustCompile(`remote\.([^=]+)\.url=(.+)`)

// ForeachPhysicalRepo walks over the repository tree, executing the action
// function once for every repo encountered (but *not* for repos contained
// within other repos, e.g. submodules).
func ForeachPhysicalRepo(action func(repo Repo) error) error {
	return filepath.Walk(RootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		//look for repos, i.e. directories containing a .git directory
		if !info.IsDir() {
			return nil
		}
		_, err = os.Stat(filepath.Join(path, ".git"))
		if err != nil {
			return nil
		}

		//appears to be a repo
		repo, err := NewRepoFromAbsolutePath(path)
		if err == nil {
			err = action(repo)
		}
		if err != nil {
			return err
		}
		//do not traverse further down into submodules etc.
		return filepath.SkipDir
	})
}

// Checkout creates the repo in the given path with the given remotes. The
// working copy will only be initialized if there is an "origin" remote.
func (r Repo) Checkout() error {
	//check if we have an "origin" remote to clone from
	var originURL RemoteURL
	for remoteName, remote := range r.Remotes {
		if remoteName == "origin" {
			originURL = remote.URLs[0]
			break
		}
	}

	if originURL == "" {
		err := cli.Interface.Run(cli.Command{
			Program: []string{"git", "init", r.AbsolutePath()},
		})
		if err != nil {
			return err
		}
		cli.Interface.ShowWarning(`will not checkout anything since there is no remote named "origin"`)
	} else {
		err := cli.Interface.Run(cli.Command{
			Program: []string{"git", "clone", originURL.CanonicalURL(), r.AbsolutePath()},
		})
		if err != nil {
			return err
		}
	}

	remotesAdded := false
	for remoteName, remote := range r.Remotes {
		for idx, url := range remote.URLs {
			if idx == 0 && remoteName != "origin" {
				err := cli.Interface.Run(cli.Command{
					Program: []string{"git", "remote", "add", remoteName, url.CanonicalURL()},
					WorkDir: r.AbsolutePath(),
				})
				if err != nil {
					return err
				}
				remotesAdded = true
			} else if idx > 0 {
				err := cli.Interface.Run(cli.Command{
					Program: []string{"git", "remote", "set-url", "--add", remoteName, url.CanonicalURL()},
					WorkDir: r.AbsolutePath(),
				})
				if err != nil {
					return err
				}
				remotesAdded = true
			}
		}
	}
	if remotesAdded {
		return cli.Interface.Run(cli.Command{
			Program: []string{"git", "remote", "update"},
			WorkDir: r.AbsolutePath(),
		})
	}

	return nil
}

// Exec implements the meat of the `rtree exec` command. It returns
// true iff the command exited successfully.
func (r Repo) Exec(cmdline ...string) error {
	cli.Interface.ShowProgress(r.AbsolutePath())
	return cli.Interface.Run(cli.Command{
		Program: cmdline,
		WorkDir: r.AbsolutePath(),
	})
}

// Move sets the CheckoutPath to the given value and moves the existing repo
// from the old to the new checkoutPath. If makeSymlink is given, a symlink will
// be created from the old to the new location.
func (r *Repo) Move(checkoutPath string, makeSymlink bool) error {
	sourcePath := filepath.Join(RootPath, r.CheckoutPath)
	targetPath := filepath.Join(RootPath, checkoutPath)

	//ensure that target does not exist
	_, err := os.Lstat(targetPath)
	if err == nil {
		return fmt.Errorf("cannot move %s to %s: target exists in filesystem", sourcePath, targetPath)
	}
	if !os.IsNotExist(err) {
		return err
	}

	//prepare directory to move repo into
	err = os.MkdirAll(filepath.Dir(targetPath), 0755)
	if err != nil {
		return err
	}

	//move directory
	err = os.Rename(sourcePath, targetPath)
	if err != nil {
		return err
	}
	r.CheckoutPath = checkoutPath

	//if requested, make compatibility symlink
	if makeSymlink {
		return os.Symlink(targetPath, sourcePath)
	}
	return nil
}

// ReformatRemoteURLs rewrites the remote URLs in this repo's .git/config into
// their canonical forms.
func (r Repo) ReformatRemoteURLs() error {
	// NOTE: This is a bit convoluted because the specific case of updating URLs
	// for a remote with multiple URLs requires multiple steps. First, we clear
	// out all non-primary URLs, and then re-add them after updating the primary URL.
	actualRepo, err := NewRepoFromAbsolutePath(r.AbsolutePath())
	if err != nil {
		return err
	}

	for remoteName, remote := range r.Remotes {
		actualRemote := actualRepo.Remotes[remoteName]

		if len(actualRemote.URLs) > 1 {
			for _, url := range actualRemote.URLs[1:] {
				err := cli.Interface.Run(cli.Command{
					Program: []string{"git", "remote", "set-url", "--delete", remoteName, url.CanonicalURL()},
					WorkDir: r.AbsolutePath(),
				})
				if err != nil {
					return err
				}
			}
		}

		err := cli.Interface.Run(cli.Command{
			Program: []string{"git", "remote", "set-url", remoteName, remote.URLs[0].CanonicalURL()},
			WorkDir: r.AbsolutePath(),
		})
		if err != nil {
			return err
		}

		if len(remote.URLs) > 1 {
			for _, url := range remote.URLs[1:] {
				err := cli.Interface.Run(cli.Command{
					Program: []string{"git", "remote", "set-url", "--add", remoteName, url.CanonicalURL()},
					WorkDir: r.AbsolutePath(),
				})
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
