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

package i3status

import (
	"net"
	"regexp"
	"strings"
)

var networkPortRx = regexp.MustCompile(`:\d+$`)
var networkMaskRx = regexp.MustCompile(`/\d+$`)

func getNetworkStatus() []Block {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil
	}
	addrStrs := make([]string, 0, len(addrs))
	for _, addr := range addrs {
		str := addr.String()
		//remove uninteresting parts
		str = networkPortRx.ReplaceAllString(str, "")
		str = networkMaskRx.ReplaceAllString(str, "")
		//ignore IPv6 for now
		if strings.ContainsRune(str, ':') {
			continue
		}
		//ignore uninteresting addrs
		if strings.HasPrefix(str, "127.") {
			continue
		}
		addrStrs = append(addrStrs, str)
	}

	if len(addrStrs) == 0 {
		return nil
	}
	return section("ip", Block{
		Name:     "network",
		Position: PositionNetwork,
		FullText: strings.Join(addrStrs, " "),
		Color:    "#00AAAA",
	})
}
