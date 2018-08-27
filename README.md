# GoSkrafl
A fast SCRABBLE(tm) engine written in Go

### About

This project aims to create a very fast SCRABBLE(tm) engine. It will
accommodate most languages (using Unicode and UTF-8 throughout) as well as
any tile bag configuration. It supports the whole game lifecycle, board,
rack and bag management, full move validation, scoring, word and
cross-word checks, and robot players.

The robot players use Go's goroutines to line up the highest-scoring
moves in parallel, employing all available processor cores for true
concurrency.

The same author has already written
a [SCRABBLE(tm) engine in Python](https://github.com/vthorsteinsson/Netskrafl)
which is up-and-running and thoroughly tested.

### Status

This software is under development and has not yet reached Alpha status.

### Original Author

_Vilhjálmur Þorsteinsson, Reykjavík, Iceland._

Contact me via GitHub for queries or information regarding GoSkrafl.

Please contact me if you would like to use GoSkrafl as a basis for your
own game program, server or website but prefer not to do so under the
conditions of the GNU GPL v3 license (see below).

### License

*GoSkrafl - a fast SCRABBLE(tm) engine written in Go*

*Copyright (C) 2018 Vilhjálmur Þorsteinsson*

This set of programs is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This set of programs is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

The full text of the GNU General Public License is available here:
[http://www.gnu.org/licenses/gpl.html](http://www.gnu.org/licenses/gpl.html).

### Trademarks

*SCRABBLE is a registered trademark. This software or its author are in no way*
*affiliated with or endorsed by the owners or licensees of the SCRABBLE trademark.*
