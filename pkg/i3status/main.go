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
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"
)

//Exec executes the i3status applet and returns an exit code (0 for
//success, >0 for error).
func Exec(args []string) int {
	//start protocol (we handle errors here for once because if stdout is
	//working, we have nothing left to do)
	_, err := os.Stdout.Write([]byte("{\"version\":1}\n[\n"))
	if err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
		return 1
	}

	//main loop
	currentBlocks := make(map[string][]Block)
	for {
		//prepare clock (this is inline instead of split into a separate function
		//because the wallclock also drives the main loop's clock, see below)
		now := time.Now()
		currentBlocks["clock"] = section(
			Block{
				Name:                "clock",
				Instance:            "date",
				Position:            PositionClock,
				FullText:            now.Format("2006-01-02"),
				ShortText:           " ",
				Color:               "#AAAAAA",
				SeparatorBlockWidth: 6,
			},
			Block{
				Name:     "clock",
				Instance: "time",
				Position: PositionClock,
				FullText: now.Format("15:04:05"),
			},
		)

		//prepare other blocks
		currentBlocks["battery"] = getBatteryStatus()

		//put blocks in rendering order
		var allBlocks []Block
		for _, blocks := range currentBlocks {
			allBlocks = append(allBlocks, blocks...)
		}
		sort.Sort(byPositionAndInstance(allBlocks))

		//write output
		buf, err := json.Marshal(allBlocks)
		if err == nil {
			os.Stdout.Write(append(buf, ',', '\n'))
		} else {
			//this should not happen, but if it does, fall back to just encoding the
			//error string (which always works, otherwise wtf)
			buf, _ = json.Marshal(err.Error())
			fmt.Printf(
				`[{"name":"error","urgent":true,"color":"#FF0000","full_text":%s}],`,
				string(buf),
			)
		}

		//sleep until the next second starts, so that the clock is always on time
		nsec := 1000000000 - now.Nanosecond()
		time.Sleep(time.Duration(nsec) * time.Nanosecond)
	}

	return 0
}

//Block is a block of text as used in the i3status protocol.
//(Not all attributes are represented.)
type Block struct {
	Position            Position  `json:"-"`
	Name                string    `json:"name"` //REQUIRED
	Instance            string    `json:"instance,omitempty"`
	FullText            string    `json:"full_text"` //REQUIRED
	ShortText           string    `json:"short_text"`
	Color               string    `json:"color,omitempty"` //CSS hex syntax, e.g. #123456
	BackgroundColor     string    `json:"background,omitempty"`
	MinWidth            uint      `json:"min_width,omitempty"`
	Alignment           Alignment `json:"align,omitempty"` //only plausible with MinWidth
	Urgent              bool      `json:"urgent,omitempty"`
	Separator           bool      `json:"separator"`
	SeparatorBlockWidth int       `json:"separator_block_width,omitempty"`
}

type byPositionAndInstance []Block

func (b byPositionAndInstance) Len() int      { return len(b) }
func (b byPositionAndInstance) Swap(i, j int) { b[i], b[j] = b[j], b[i] }
func (b byPositionAndInstance) Less(i, j int) bool {
	if b[i].Position == b[j].Position {
		return b[i].Instance < b[j].Instance
	}
	return b[i].Position < b[j].Position
}

//Alignment is the alignment of a Block.
type Alignment string

//Acceptable values for Alignment.
const (
	AlignmentLeft   Alignment = "left"
	AlignmentCenter Alignment = "center"
	AlignmentRight  Alignment = "right"
)

//Position defines how blocks are ordered.
type Position int

//Acceptable values for Position, from left to right.
const (
	PositionNone Position = iota
	PositionBattery
	PositionClock
)

//Order the given blocks byPositionAndInstance, then add a separator to the last one.
func section(blocks ...Block) []Block {
	if len(blocks) == 0 {
		return nil
	}
	sort.Sort(byPositionAndInstance(blocks))
	last := len(blocks) - 1
	blocks[last].Separator = true
	blocks[last].SeparatorBlockWidth = 15
	return blocks
}
