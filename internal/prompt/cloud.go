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
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	yaml "gopkg.in/yaml.v2"
)

func getKubernetesField() string {
	_, err := os.Stat("/x/bin/u8s")
	if err == nil {
		return getKubernetesFieldViaU8S()
	}

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

	namespaceBytes, err := os.ReadFile(filepath.Join(os.Getenv("HOME"), ".kubectl-namespace"))
	namespace := strings.TrimSpace(string(namespaceBytes))
	if err != nil {
		if !os.IsNotExist(err) {
			handleError(err)
		}
		namespace = ""
	}

	return buildKubernetesField(context, namespace)
}

func getKubernetesContext(configPath string) string {
	buf, err := os.ReadFile(configPath)
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

func getKubernetesFieldViaU8S() string {
	stdout, err := exec.Command("u8s", "env").Output()
	if err != nil {
		return ""
	}

	var (
		context   string
		namespace string
	)
	for _, line := range strings.Split(string(stdout), "\n") {
		fields := strings.SplitN(strings.TrimSpace(line), "=", 2)
		if len(fields) != 2 {
			continue
		}
		switch fields[0] {
		case "U8S_CONTEXT":
			context = fields[1]
		case "U8S_NAMESPACE":
			namespace = fields[1]
		}
	}
	return buildKubernetesField(context, namespace)
}

func buildKubernetesField(context, namespace string) string {
	if context == "" {
		return ""
	}
	if !strings.Contains(context, "qa") {
		//visual warning when working in a productive region
		context = withColor("1;41", context)
	}
	if namespace != "" {
		context += "/" + namespace
	}
	return withType("kube", context)
}

func getOpenstackField() string {
	cloudName := os.Getenv("CURRENT_OS_CLOUD")
	if cloudName == "" {
		return ""
	}
	return withType("cloud", cloudName)
}
