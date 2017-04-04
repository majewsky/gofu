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
	return &index
}

//Write writes the index file to disk.
func (i *Index) Write() {
	buf, err := yaml.Marshal(i)
	FatalIfError(err)
	path := indexPath()
	FatalIfError(os.MkdirAll(filepath.Dir(path), 0755))
	FatalIfError(ioutil.WriteFile(path, buf, 0644))
}
