# GoSkrafl
A concurrent SCRABBLE(tm) engine and robot, written in Go

### About

GoSkrafl is a **fast, concurrent** SCRABBLE(tm) engine and **auto-playing robot**.
It is a package for the Go programming language, licensed under GNU GPLv3.
It has been tested on Linux and Windows, and probably works fine on MacOS too.

Out of the box, GoSkrafl supports Icelandic (the author's native language).
But as it uses Unicode and UTF-8 throughout, GoSkrafl can easily be tweaked
to accommodate most natural languages and dictionaries, and any tile bag
configuration. It supports the whole game lifecycle, board, rack and bag
management, move validation, scoring, word and cross-word checks, and
**robot players**.

The robot players make good use of Go's **goroutines** to evaluate all valid
moves concurrently, employing all available processor cores for
**parallel execution** of multiple worker threads. This, coupled with Go's
compilation to native machine code, and its efficient memory management,
makes GoSkrafl quite fast. (As an order of magnitude, it runs at
approximately 10 simulated games per second on a quad-core
Intel i7-4400 processor @ 3.4 GHz, with the Icelandic dictionary
of 2.4 million word forms.)

The design and code of GoSkrafl borrow heavily from a battle-hardened
[SCRABBLE(tm) engine in Python](https://github.com/vthorsteinsson/Netskrafl)
by the same author.

### Status

GoSkrafl is currently in Alpha but moving close to Beta.

### Example

To enjoy seeing two robots slug it out at the SCRABBLE(tm) board:

```go
package main

import (
    "fmt"
    skrafl "github.com/vthorsteinsson/GoSkrafl"
)

func main() {
    // Set up a game of Icelandic SCRABBLE(tm)
    game := skrafl.NewIcelandicGame()
    game.SetPlayerNames("Robot A", "Robot B")
    // Create a robot that always selects
    // the highest-scoring valid move
    robot := skrafl.NewHighScoreRobot()
    // Print the initial game board and racks
    fmt.Printf("%v\n", game)
    // Generate moves until the game ends
    for {
        state := game.State()
        move := robot.GenerateMove(state)
        game.ApplyValid(move)
        // Print the new game state after the move
        fmt.Printf("%v\n", game)
        if game.IsOver() {
            fmt.Printf("Game over!\n")
            break
        }
    }
}
```

### Original Author

_Vilhjálmur Þorsteinsson, Reykjavík, Iceland._

Contact me via GitHub for queries or information regarding GoSkrafl,
for instance if you would like to use GoSkrafl as a basis for your
own game program, server or website but prefer not to do so under the
conditions of the GNU GPL v3 license (see below).

### License

*GoSkrafl - a concurrent SCRABBLE(tm) engine and robot, written in Go*

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
