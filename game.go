package main

//import "fmt"
import "math"

type Animation struct {
    prev uint64
    cur uint64
    animPos int32
    animLen int32
}

type MainMenu struct {
	nPlayers int
	timeLimitSecs int
}

type Player struct {
    waitingToPlace bool
	turnTiles [7]int8
	nTilesHeld int32
	turnState Animation
	turnOffsetsBits Animation
	deckTilesBits Animation
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

type Game struct {
	menu MainMenu
	players [4]Player
	state Animation

    turnCursorX float32
    turnCursorY float32
    cursorDuringMoveX int32
    cursorDuringMoveY int32

    shuffleTimer int32

	bagMap []int32
	bagChars []byte
	wordsList []string
	wordMap map[string]int
	boardTiles []int8
	shuffleBuf [14]int8

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

const ROTA_VERT = 0
const ROTA_HORI = 1

const DECK_OPENING = 0
const DECK_REFILL  = 1
const DECK_SHUFFLE = 2

const SHUFFLE_DURATION = 10

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

func (game *Game) updateShuffleBuffer() {
    for i := 0; i < 7; i++ {
        game.shuffleBuf[7+i] = int8(i)
    }
    for i := 0; i < 7; i++ {
        idx := int(game.getRandom(int64(7-i)))
        game.shuffleBuf[i] = game.shuffleBuf[7+idx]
        game.shuffleBuf[7+idx] = game.shuffleBuf[13-i]
    }
    for i := 0; i < 7; i++ {
        game.shuffleBuf[7 + game.shuffleBuf[i]] = int8(i)
    }
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
		    game.players[i].turnTiles[j] = 0
	    }
	    game.players[i].nTilesHeld = 0
	    game.players[i].turnOffsetsBits.reset()
	    game.players[i].deckTilesBits.reset()
	}

    game.state.prev = 0
    game.state.cur = PLAYER_TURN
    game.state.animPos = 0
    game.state.animLen = 80

    for i := 0; i < game.menu.nPlayers; i++ {
        for j := 0; j < 7; j++ {
            tile := game.takeTileFromBag()
            game.players[i].deckTilesBits.cur <<= 8
            game.players[i].deckTilesBits.cur |= uint64(tile & 0x7f)
        }
    }
}

func (game *Game) simulate(inputs *Inputs) {
    player := int32(game.state.cur) & 3
    mode := int32(game.state.cur) & ^3
    if mode == PLAYER_TURN {
        game.simulatePlayerTurn(inputs, player)
    } else if mode == SCORING_TURN {
        game.simulateScoringTurn(inputs, player)
    }

    game.state.step()
}

func (game *Game) simulatePlayerTurn(inputs *Inputs, playerIdx int32) {
    p := &game.players[playerIdx]

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
        tiles := p.deckTilesBits.cur
        for j := 0; j < 7; j++ {
            if ((tiles >> ((7-j-1)*8)) & 0x7f) == uint64(idx & 0x7f) {
                p.deckTilesBits.cur &= ^(uint64(0xff) << ((7-j-1)*8))
                p.turnTiles[p.nTilesHeld] = idx
                p.nTilesHeld++
                break
            }
        }
    }

    shouldRotate := false
    shouldPlace := false
    shouldShuffle := false

    for _, code := range inputs.pressedKeys {
        if code == KEY_BACKSPACE {
            if p.nTilesHeld <= 0 {
                p.nTilesHeld = 0
                continue
            }
            tiles := p.deckTilesBits.cur
            for j := 0; j < 7; j++ {
                if ((tiles >> ((7-j-1)*8)) & 0x7f) == 0 {
                    p.nTilesHeld--
                    idx := p.turnTiles[p.nTilesHeld]
                    p.deckTilesBits.cur |= uint64(idx & 0x7f) << ((7-j-1)*8)
                    break
                }
            }
        } else if code == KEY_LSHIFT || code == KEY_RSHIFT {
            shouldRotate = p.turnState.cur == ROTA_VERT
        } else if code == KEY_LCTRL || code == KEY_RCTRL {
            shouldRotate = p.turnState.cur == ROTA_HORI
        } else if code == KEY_RETURN {
            shouldPlace = true
        } else if code == KEY_TAB {
            shouldShuffle = game.state.animLen == 0
        }
    }

    if (inputs.mouseButtons[1] & 1) == 1 {
        shouldRotate = true
    }
    if (inputs.mouseButtons[0] & 1) == 1 {
        shouldPlace = true
    }

    if shouldRotate {
        const rotateLen = 20
        p.turnState.prev = p.turnState.cur
        p.turnState.cur ^= 1
        p.turnState.animLen = rotateLen
        if (p.turnState.animPos > 0) {
            p.turnState.animPos = rotateLen - p.turnState.animPos
        } else {
            p.turnState.animPos = 0
        }

        p.turnOffsetsBits.prev = p.turnOffsetsBits.cur
        p.turnOffsetsBits.cur = 0
        p.turnOffsetsBits.animPos = p.turnState.animPos
        p.turnOffsetsBits.animLen = p.turnState.animLen
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

    didPlace := false

    nHeld := int(p.nTilesHeld)
    if nHeld > 0 {
        tileSize := int(game.tileSize)
        boardLen := tileSize * 15
        xBoardOff := (int(game.wndWidth) - boardLen) / 2
        yBoardOff := (int(game.wndHeight) - boardLen - 2 * tileSize) / 2

        boardCurX := int(game.turnCursorX) - xBoardOff
        boardCurY := int(game.turnCursorY) - yBoardOff
        col := boardCurX / tileSize
        row := boardCurY / tileSize

        if boardCurX >= 0 && boardCurY >= 0 && col >= 0 && col < 15 && row >= 0 && row < 15 {
            xInc := 1
            yInc := 0
            if p.turnState.cur == ROTA_VERT {
                xInc = 0
                yInc = 1
            }

            offsetBits := uint64(0)
            totalOffset := 0
            x := col
            y := row
            for i := 0; i < nHeld; i++ {
                offsetBits <<= 4
                offset := 0
                for true {
                    x = col + xInc * (i + totalOffset)
                    y = row + yInc * (i + totalOffset)
                    if x >= 15 || y >= 15 {
                        shouldPlace = false
                        break
                    } else if game.boardTiles[x + 15 * y] != 0 {
                        totalOffset++
                        offset++
                    } else {
                        break
                    }
                }
                offsetBits |= uint64(offset & 0xf)
            }

            p.turnOffsetsBits.cur = offsetBits

            didPlace = shouldPlace
            if didPlace {
                offset := 0
                for i := 0; i < nHeld; i++ {
                    offset += int((offsetBits >> ((nHeld-i-1)*4)) & 0xf)
                    x = col + xInc * (i + offset)
                    y = row + yInc * (i + offset)
                    game.boardTiles[x + 15 * y] = p.turnTiles[i]
                    p.turnTiles[i] = 0
                }
            }
        }
    }

    if game.shuffleTimer > 0 {
        game.shuffleTimer--
        if didPlace {
            p.waitingToPlace = true
        }
    }
    if game.shuffleTimer == 0 {
        if didPlace || p.waitingToPlace {
            p.waitingToPlace = false

            p.deckTilesBits.prev = p.deckTilesBits.cur
            p.deckTilesBits.animPos = 0
            p.deckTilesBits.animLen = 60

            for i := 0; i < 7; i++ {
                shift := (7-i-1)*8
                if ((p.deckTilesBits.cur >> shift) & 0x7f) == 0 {
                    tile := game.takeTileFromBag()
                    if tile == 0 {
                        break
                    }
                    p.deckTilesBits.cur |= uint64(tile & 0x7f) << shift
                }
            }
            game.state.cur = uint64(SCORING_TURN | (playerIdx & 3))
            p.nTilesHeld = 0
            p.turnOffsetsBits.cur = 0
        } else if shouldShuffle {
            game.updateShuffleBuffer()
            oldDeck := p.deckTilesBits.cur
            newDeck := uint64(0)
            for i := 0; i < 7; i++ {
                n := (oldDeck >> ((7-i-1) * 8)) & 0x7f
                newDeck |= n << (game.shuffleBuf[i] * 8)
            }
            p.deckTilesBits.prev = oldDeck
            p.deckTilesBits.cur = newDeck
            game.shuffleTimer = SHUFFLE_DURATION
        } else {
            // this allows to draw the previous deck value, and the current new tiles if part of the picking animation
            p.deckTilesBits.prev = p.deckTilesBits.cur
        }
    }

    p.turnState.step()
    p.turnOffsetsBits.step()
}

func (game *Game) simulateScoringTurn(inputs *Inputs, playerIdx int32) {
    if game.players[playerIdx].deckTilesBits.step() {
        game.state.cur = uint64(PLAYER_TURN | ((playerIdx + 1) % int32(game.menu.nPlayers)))
        game.state.animPos = 0
        game.state.animLen = 80
    }
}

func (a *Animation) step() (justCompleted bool) {
    justCompleted = false
    if a.animPos < a.animLen {
        a.animPos += 1
    }
    if a.animPos >= a.animLen {
        justCompleted = a.animLen > 0
        a.animPos = 0
        a.animLen = 0
        a.prev = a.cur
    }
    return justCompleted
}

func (a *Animation) getPositionOr(t float32) float32 {
    if a.animLen > 0 {
        t = min(max(float32(a.animPos) / float32(a.animLen), 0.0), 1.0)
    }
    return t
}

func (a *Animation) reset() {
    a.prev = 0
    a.cur = 0
    a.animPos = 0
    a.animLen = 0
}

func makeInputs() Inputs {
    inputs := Inputs{}
    inputs.pressedKeys  = make([]int32, 0, 16)
    inputs.pressedChars = make([]int32, 0, 16)
    return inputs
}
