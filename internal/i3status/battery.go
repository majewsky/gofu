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
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
)

const (
	batteryFullPath = "/sys/class/power_supply/BAT0/energy_full"
	batteryNowPath  = "/sys/class/power_supply/BAT0/energy_now"
	powerOnlinePath = "/sys/class/power_supply/AC0/online"
)

func getBatteryStatus() []Block {
	//TODO: What's with /sys/class/power_supply/BAT0/uevent? Can this be used to
	//remove all the polling overhead here?

	energyFull, err := readNumberFromFile(batteryFullPath)
	if err != nil {
		return nil
	}
	energyNow, err := readNumberFromFile(batteryNowPath)
	if err != nil {
		return nil
	}
	powerOnline, err := readNumberFromFile(powerOnlinePath)
	if err != nil {
		return nil
	}

	energyPerc := energyNow * 100 / energyFull
	charging := powerOnline > 0
	color := "#AAAA00"
	if charging {
		color = "#00AA00"
	} else if energyPerc < 10 {
		color = "#AA0000"
	}

	return section("bat", Block{
		Name:     "battery",
		Position: PositionBattery,
		FullText: fmt.Sprintf("%d%%", energyPerc),
		Urgent:   energyPerc < 10 && !charging,
		Color:    color,
	})
}

func readNumberFromFile(path string) (int64, error) {
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(strings.TrimSpace(string(buf)), 0, 64)
}
