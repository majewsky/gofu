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

//Package earlyerrors can be used to cache errors that occurred during func
//init(). These errors can then be displayed by func main().
package earlyerrors

import "fmt"

var errs []string

//Put submits an error message.
func Put(err string) {
	if err != "" {
		errs = append(errs, err)
	}
}

//Putf builds and submits an error message.
func Putf(err string, args ...interface{}) {
	Put(fmt.Sprintf(err, args...))
}

//Get returns all collected errors, or an empty slice if there were none.
func Get() []string {
	return errs
}
