# GoSkrafl
A concurrent crossword game engine and robot, written in Go

### About

GoSkrafl is a **fast, concurrent** crossword game engine and **auto-playing robot**.
It is a package for the Go programming language, licensed under CC-BY-NC 4.0.
It has been tested on Linux and Windows, and probably works fine on MacOS too.

Out of the box, GoSkrafl supports **OTCWL2014**, **SOWPODS**, **OSPS37**,
**Norwegian** and **Icelandic** dictionaries and corresponding tile sets.
But as it employs Unicode and UTF-8 throughout, GoSkrafl can easily be tweaked
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
over **25 simulated OTCWL2014 games per second** on a quad-core
Intel i7-4400 processor @ 3.4 GHz, or less than 40 milliseconds per game.)

The design and code of GoSkrafl borrow heavily from a battle-hardened
[crossword game engine in Python](https://github.com/vthorsteinsson/Netskrafl)
by the same author.

### Status

GoSkrafl is well tested and in production. Issues and pull requests are welcome.

### Adding new dictionaries

To add support for a new dictionary, assemble the word list in a UTF-8 text file,
with all words in lower case, one word per line. Use the
[DAWG builder from Netskrafl](https://github.com/vthorsteinsson/Netskrafl/blob/master/src/dawgbuilder.py)
to build a `.bin.dawg` file.
Copy it to the `/GoSkrafl/dicts/` directory, then add a snippet of
code at the bottom of `dawg.go` to wrap it in an instance of the `Dawg` class. Remember to
add an alphabet string as well, cf. the `IcelandicAlphabet`,
`EnglishAlphabet` and `PolishAlphabet` variables.
The same alphabet string must be used for the encoding in `dawgbuilder.py`.
Post an issue if you need help.

### Example

To enjoy seeing two robots slug it out in a game:

```go
package main

import (
    "fmt"
    skrafl "github.com/vthorsteinsson/GoSkrafl"
)

func main() {
    // Set up a game using the SOWPODS dictionary
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
conditions of the CC-BY-NC 4.0 license (see below).

### License

**GoSkrafl - a concurrent crossword game engine and robot, written in Go**

*Copyright (C) 2024 Vilhjálmur Þorsteinsson / Miðeind ehf.*

This set of programs is licensed under the **Creative Commons
Attribution-NonCommercial 4.0 International Public License (CC-BY-NC 4.0).**

The full text of the license is available
here: https://creativecommons.org/licenses/by-nc/4.0/ - as well as in the
LICENSE file in this repository.

### Trademarks

*SCRABBLE is a registered trademark. This software or its author are in no way*
*affiliated with or endorsed by the owners or licensees of the SCRABBLE trademark.*
