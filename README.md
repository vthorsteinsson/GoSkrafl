# GoSkrafl
A concurrent SCRABBLE(tm) engine and robot, written in Go

### About

GoSkrafl is a **fast, concurrent** SCRABBLE(tm) engine and **auto-playing robot**.
It is a package for the Go programming language, licensed under GNU GPLv3.
It has been tested on Linux and Windows, and probably works fine on MacOS too.

Out of the box, GoSkrafl supports **TWL06**, **SOWPODS** and **Icelandic**
SCRABBLE(tm) dictionaries and corresponding tile sets. But as it employs
Unicode and UTF-8 throughout, GoSkrafl can easily be tweaked
to accommodate most natural languages and dictionaries, and any tile bag
configuration. (The only limitation is that there cannot be more different
letters in an alphabet than there are bits in the native **uint** type.)

The GoSkrafl package encompasses the whole game lifecycle, board, rack and bag
management, move validation, scoring, word and cross-word checks, as well as
**robot players**.

The robot players make good use of Go's **goroutines** to discover valid
moves concurrently, employing all available processor cores for
**parallel execution** of multiple worker threads. This, coupled with Go's
compilation to native machine code, and its efficient memory management,
makes GoSkrafl quite fast. (As an order of magnitude, it runs at
over **25 simulated TWL06 games per second** on a quad-core
Intel i7-4400 processor @ 3.4 GHz, or less than 40 milliseconds per game.)

The design and code of GoSkrafl borrow heavily from a battle-hardened
[SCRABBLE(tm) engine in Python](https://github.com/vthorsteinsson/Netskrafl)
by the same author.

### Status

GoSkrafl is currently in Beta. Issues and pull requests are welcome.

### Adding new dictionaries

To add support for a new dictionary, assemble the word list in a UTF-8 text file,
with all words in lower case, one word per line. Use the
[DAWG builder from Netskrafl](https://github.com/vthorsteinsson/Netskrafl/blob/singlepage/dawgbuilder.py)
to build a `.bin.dawg` file.
Copy it to the `/GoSkrafl/dicts/` directory, then add a snippet of
code at the bottom of `dawg.go` to wrap it in an instance of the `Dawg` class. Remember to
add an alphabet string as well, cf. the `IcelandicAlphabet` and `EnglishAlphabet` variables.
The same alphabet string must be used for the encoding in `dawgbuilder.py`. Post an issue if
you need help.

### Example

To enjoy seeing two robots slug it out at the SCRABBLE(tm) board:

```go
package main

import (
    "fmt"
    skrafl "github.com/vthorsteinsson/GoSkrafl"
)

func main() {
    // Set up a game of SOWPODS SCRABBLE(tm)
    game := skrafl.NewSowpodsGame()
    game.SetPlayerNames("Robot A", "Robot B")
    // Create a robot that always selects
    // the highest-scoring valid move
    robot := skrafl.NewHighScoreRobot()
    // Print the initial game board and racks
    fmt.Printf("%v\n", game)
    // Generate moves until the game ends
    for {
        // Extract the game state
        state := game.State()
        // Find the highest-scoring move available
        move := robot.GenerateMove(state)
        // Apply the (implicitly validated) move to the game
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

A fancier **main** program for exercising the GoSkrafl engine can
be [found here](https://github.com/vthorsteinsson/GoSkrafl/blob/master/main/main.go).

### Original Author

_Vilhjálmur Þorsteinsson, Reykjavík, Iceland._

Contact me via GitHub for queries or information regarding GoSkrafl,
for instance if you would like to use GoSkrafl as a basis for your
own game program, server or website but prefer not to do so under the
conditions of the GNU GPL v3 license (see below).

### License

*GoSkrafl - a concurrent SCRABBLE(tm) engine and robot, written in Go*

*Copyright (C) 2019 Vilhjálmur Þorsteinsson*

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
