package main

//import "fmt"
import "math"

type MainMenu struct {
	nPlayers int
	timeLimitSecs int
}

type Player struct {
	deckTiles [7]int8
	turnTiles [7]int8
	nTilesHeld int32
}

type Inputs struct {
    pressedKeys []int32
    pressedChars []int32
    mouseButtons [2]int32
    arrowTimers [4]uint32
    cursorX int32
    cursorY int32
    cursorVelX float32
    cursorVelY float32
}

type Animation struct {
    prev int32
    cur int32
    animPos int32
    animLen int32
}

type Game struct {
	menu MainMenu
	players [4]Player
	state Animation
	turnState Animation

    turnCursorX float32
    turnCursorY float32
    cursorDuringMoveX int32
    cursorDuringMoveY int32

	bagMap []int32
	bagChars []byte
	wordsList []string
	wordMap map[string]int
	boardTiles []int8

	startupTimestamp int64
	prevHash64 uint64
	frameCounter int64
	callsToRng int32

	wndWidth int32
	wndHeight int32
	tileSize int32
}

type Tile struct {
	letter int32
	points int32
	count int32
}

const NORMAL = 0
const DOUBLE_WORD = 1
const TRIPLE_WORD = 2
const DOUBLE_LETTER = 3
const TRIPLE_LETTER = 4

const PICK_ORDER = 4
const PLAYER_TURN = 8
const SCORING_TURN = 12

var boardTileTypeLookup = [...]int32 {
    2, 0, 0, 3, 0, 0, 0, 2,
    0, 1, 0, 0, 0, 4, 0, 0,
    0, 0, 1, 0, 0, 0, 3, 0,
    3, 0, 0, 1, 0, 0, 0, 3,
    0, 0, 0, 0, 1, 0, 0, 0,
    0, 4, 0, 0, 0, 4, 0, 0,
    0, 0, 3, 0, 0, 0, 3, 0,
}

var tiles = [...]Tile {
	{'A', 1, 9},
	{'B', 3, 2},
	{'C', 3, 2},
	{'D', 2, 4},
	{'E', 1, 12},
	{'F', 4, 2},
	{'G', 2, 3},
	{'H', 4, 2},
	{'I', 1, 9},
	{'J', 8, 1},
	{'K', 5, 1},
	{'L', 1, 4},
	{'M', 3, 2},
	{'N', 1, 6},
	{'O', 1, 8},
	{'P', 3, 2},
	{'Q', 10, 1},
	{'R', 1, 6},
	{'S', 1, 4},
	{'T', 1, 6},
	{'U', 1, 4},
	{'V', 4, 2},
	{'W', 4, 2},
	{'X', 8, 1},
	{'Y', 4, 2},
	{'Z', 10, 1},
	{' ', 0, 2},
}

func getTileType(x, y int32) int32 {
    if x < 0 || y < 0 || x >= 15 || y >= 15 {
        return NORMAL
    }
    if x == 7 && y == 7 {
        return DOUBLE_WORD
    }

    if x >= 7 && y >= 8 {
        x = 14 - x
        y = 14 - y
    } else if x >= 8 && y <= 7 {
        temp := x
        x = y
        y = 14 - temp
    } else if x <= 7 && y >= 7 {
        temp := x
        x = 14 - y
        y = temp
    }

    return boardTileTypeLookup[x + 8 * y]
}

func getLetterScores() (scores []int32) {
    scores = make([]int32, 27)
    for i := 0; i < 27; i++ {
        scores[i] = tiles[i].points
    }
    return scores
}

func generateTileBag() ([]byte, []int32) {
	var chars []byte
	idx := 0
	for i := 0; i < len(tiles); i++ {
		n := int(tiles[i].count)
		if idx + n > cap(chars) {
			chars = append(make([]byte, 0, max(idx + n, cap(chars))), chars...)
		}
		chars = chars[:idx+n]

		for j := 0; j < n; j++ {
			chars[idx+j] = byte(tiles[i].letter & 0xff)
		}
		idx += n
	}

	bag := make([]int32, idx)
	for i := 0; i < idx; i++ {
		bag[i] = int32(i)
	}

	return chars, bag
}

func (game *Game) takeTileFromBag() int8 {
	if len(game.bagMap) == 0 {
		return 0
	}

	idx := int(game.getRandom(int64(len(game.bagMap))))
	selected := game.bagMap[idx]
	game.bagMap[idx] = game.bagMap[len(game.bagMap)-1]
	game.bagMap = game.bagMap[:len(game.bagMap)-1]

	ch := game.bagChars[selected]
	if ch == ' ' {
		return 27
	}
	return int8(ch) - 0x40
}

func (game *Game) getBoardTile(index int) Tile {
    if index < 0 || index >= 15 * 15 || game.boardTiles[index] == 0 {
        return Tile{}
    }
    return tiles[game.boardTiles[index] - 1]
}

func (game *Game) init(wordsList []string, timestamp int64) {
	game.wordsList = wordsList
	game.startupTimestamp = timestamp
	game.boardTiles = make([]int8, 15 * 15)

	game.wordMap = make(map[string]int)
	for i := 0; i < len(game.wordsList); i++ {
		game.wordMap[game.wordsList[i]] = i
	}
}

func (game *Game) getRandom(endExclusive int64) int64 {
	upper, lower := generateNext128(game.startupTimestamp, (game.frameCounter << 8) | int64(game.callsToRng & 0xff), game.prevHash64)
	game.callsToRng += 1
	game.prevHash64 = upper

	value := int64(lower & ^(uint64(1) << 63))
	if endExclusive > 0 {
		value = value % endExclusive
	}
	return value
}

func (game *Game) start() {
    game.bagChars, game.bagMap = generateTileBag()
	for i := 0; i < 15 * 15; i++ {
		game.boardTiles[i] = 0
	}
	for i := 0; i < 4; i++ {
	    for j := 0; j < 7; j++ {
		    game.players[i].deckTiles[j] = 0
		    game.players[i].turnTiles[j] = 0
	    }
	    game.players[i].nTilesHeld = 0
	}

    game.state.prev = 0
    game.state.cur = PLAYER_TURN
    game.state.animPos = 0
    game.state.animLen = 80

    if false {
        for i := 0; i < game.menu.nPlayers * 7; i++ {
            game.players[i / 7].deckTiles[i % 7] = game.takeTileFromBag()
        }
    } else {
        for i := 0; i < 7; i++ {
            game.players[0].deckTiles[i] = game.takeTileFromBag()
        }
    }
}

func (game *Game) simulate(inputs *Inputs) {
    player := game.state.cur & 3
    mode := game.state.cur & ^3
    if mode == PLAYER_TURN {
        game.simulatePlayerTurn(inputs, player)
    }

    game.state.step()
}

func (game *Game) simulatePlayerTurn(inputs *Inputs, playerIdx int32) {
    for _, char := range inputs.pressedChars {
        idx := int8(0)
        if char >= 'a' && char <= 'z' {
            idx = int8(char - 0x60)
        } else if char >= 'A' && char <= 'Z' {
            idx = int8(char - 0x40)
        } else if char == ' ' {
            idx = 27
        }
        if idx <= 0 {
            continue
        }
        for j := 0; j < 7; j++ {
            if game.players[playerIdx].deckTiles[j] == idx {
                game.players[playerIdx].deckTiles[j] = 0
                game.players[playerIdx].turnTiles[game.players[playerIdx].nTilesHeld] = idx
                game.players[playerIdx].nTilesHeld++
                break
            }
        }
    }

    shouldRotate := false

    for _, code := range inputs.pressedKeys {
        if code == KEY_BACKSPACE {
            if game.players[playerIdx].nTilesHeld <= 0 {
                game.players[playerIdx].nTilesHeld = 0
                continue
            }
            for j := 0; j < 7; j++ {
                if (game.players[playerIdx].deckTiles[j] == 0) {
                    game.players[playerIdx].nTilesHeld--
                    game.players[playerIdx].deckTiles[j] = game.players[playerIdx].turnTiles[game.players[playerIdx].nTilesHeld]
                    break
                }
            }
        }
        if code == KEY_LSHIFT || code == KEY_RSHIFT {
            shouldRotate = (game.turnState.cur & 1) == 0
        }
        if code == KEY_LCTRL || code == KEY_RCTRL {
            shouldRotate = (game.turnState.cur & 1) == 1
        }
    }

    if (inputs.mouseButtons[1] & 1) == 1 {
        shouldRotate = true
    }

    if shouldRotate {
        const rotateLen = 20
        game.turnState.prev = game.turnState.cur
        game.turnState.cur ^= 1
        game.turnState.animLen = rotateLen
        if (game.turnState.animPos > 0) {
            game.turnState.animPos = rotateLen - game.turnState.animPos
        } else {
            game.turnState.animPos = 0
        }
    }

    if inputs.arrowTimers[ARROW_UP] != 0 || inputs.arrowTimers[ARROW_DOWN] != 0 ||
        inputs.arrowTimers[ARROW_LEFT] != 0 || inputs.arrowTimers[ARROW_RIGHT] != 0 {

        game.cursorDuringMoveX = inputs.cursorX
        game.cursorDuringMoveY = inputs.cursorY

        const tttv = uint32(20)
        f := float32(game.wndWidth) / float32(150 * tttv)
        game.turnCursorY -= f * float32(min(inputs.arrowTimers[ARROW_UP], tttv))
        game.turnCursorY += f * float32(min(inputs.arrowTimers[ARROW_DOWN], tttv))
        game.turnCursorX -= f * float32(min(inputs.arrowTimers[ARROW_LEFT], tttv))
        game.turnCursorX += f * float32(min(inputs.arrowTimers[ARROW_RIGHT], tttv))
    } else {
        dx := float64(inputs.cursorX - game.cursorDuringMoveX)
        dy := float64(inputs.cursorY - game.cursorDuringMoveY)
        distFromCursor := math.Sqrt(dx * dx + dy * dy)
        if distFromCursor >= 5 {
            game.cursorDuringMoveX = 0
            game.cursorDuringMoveY = 0
            game.turnCursorX = float32(inputs.cursorX)
            game.turnCursorY = float32(inputs.cursorY)
        }
    }

    game.turnState.step()
}

func (a *Animation) step() {
    if a.animPos < a.animLen {
        a.animPos += 1
    }
    if a.animPos >= a.animLen {
        a.animPos = 0
        a.animLen = 0
        a.prev = a.cur
    }
}

func makeInputs() Inputs {
    inputs := Inputs{}
    inputs.pressedKeys  = make([]int32, 0, 16)
    inputs.pressedChars = make([]int32, 0, 16)
    return inputs
}
