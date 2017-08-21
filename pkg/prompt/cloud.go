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

	yaml "gopkg.in/yaml.v2"
)

func getKubernetesField() string {
	configPaths := filepath.SplitList(os.Getenv("KUBECONFIG"))

	var context string
	for _, configPath := range configPaths {
		context = getKubernetesContext(configPath)
		if context != "" {
			break
		}
	}
	if context == "" {
		return ""
	}

	namespaceBytes, err := ioutil.ReadFile(filepath.Join(os.Getenv("HOME"), ".kubectl-namespace"))
	namespace := strings.TrimSpace(string(namespaceBytes))
	if err != nil {
		if !os.IsNotExist(err) {
			handleError(err)
		}
		namespace = ""
	}
	if namespace != "" {
		context += "/" + namespace
	}

	return withType("kube", context)
}

func getKubernetesContext(configPath string) string {
	buf, err := ioutil.ReadFile(configPath)
	if err != nil {
		//non-existence is acceptable, just make the caller continue with the next configPath
		if !os.IsNotExist(err) {
			handleError(err)
		}
		return ""
	}

	var data struct {
		CurrentContext string `yaml:"current-context"`
	}
	err = yaml.Unmarshal(buf, &data)
	handleError(err)
	return strings.TrimSpace(data.CurrentContext)
}

func getOpenstackField() string {
	cloudName := os.Getenv("CURRENT_OS_CLOUD")
	if cloudName == "" {
		return ""
	}
	return withType("cloud", cloudName)
}
