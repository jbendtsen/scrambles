package main

//import "fmt"
import "strconv"
import "strings"

type Animation struct {
    prev uint64
    cur uint64
    animPos int32
    animLen int32
}

type MainMenu struct {
	timeLimitSecs int
	shouldValidateEveryWord bool
}

type Player struct {
    kind int32
    totalScore int32
    turnScore int32
    nTilesHeld int32
	turnLetters [7]int8
	turnPositions [7]uint8
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
    shuffleBuf [14]int8

	bagMap []int32
	bagChars []byte
	wordsList []string
	wordMap map[string]int
	boardTiles []int8

    activeLines []uint16
	scoringWords []string
	scoringCommands []uint16
	wordBuilder strings.Builder

    scoreDisplayStrings []string

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
const ALL_LETTERS = 5
const BONUS = 0x40

const PICK_ORDER = 4
const PLAYER_TURN = 8
const SCORING_TURN = 12

const ROTA_VERT = 0
const ROTA_HORI = 1

const DECK_OPENING = 0
const DECK_REFILL  = 1
const DECK_SHUFFLE = 2

const SHUFFLE_DURATION = 10
const TILE_SCORE_DURATION = 30

const PLAYER_INACTIVE = 0
const PLAYER_REAL = 1
const PLAYER_CPU_EASY = 2
const PLAYER_CPU_HARD = 3

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

    game.scoreDisplayStrings = make([]string, max(BONUS + 4, 51))
    for i := 0; i <= 10; i++ {
        game.scoreDisplayStrings[i] = "+" + strconv.Itoa(i)
    }
    for i := 2; i <= 3; i++ {
        game.scoreDisplayStrings[BONUS + i] = "x" + strconv.Itoa(i)
    }
    game.scoreDisplayStrings[50] = "+50"
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
	    game.players[i].totalScore = 0
	    game.players[i].turnScore = 0
	    for j := 0; j < 7; j++ {
		    game.players[i].turnLetters[j] = 0
		    game.players[i].turnPositions[j] = 0
	    }
	    game.players[i].nTilesHeld = 0
	    game.players[i].turnOffsetsBits.reset()
	    game.players[i].deckTilesBits.reset()
	}

    game.state.prev = 0
    game.state.cur = PLAYER_TURN
    game.state.animPos = 0
    game.state.animLen = 80

    for i := 0; i < 4; i++ {
        if game.players[i].kind == PLAYER_INACTIVE {
            continue
        }
        for j := 0; j < 7; j++ {
            tile := game.takeTileFromBag()
            game.players[i].deckTilesBits.cur <<= 8
            game.players[i].deckTilesBits.cur |= uint64(tile & 0x7f)
        }
    }
}

func (game *Game) updateCursor(inputs *Inputs) {
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
        dx := inputs.cursorX - game.cursorDuringMoveX
        dy := inputs.cursorY - game.cursorDuringMoveY
        distFromCursorSq := dx * dx + dy * dy
        if distFromCursorSq >= 25 {
            game.cursorDuringMoveX = 0
            game.cursorDuringMoveY = 0
            game.turnCursorX = float32(inputs.cursorX)
            game.turnCursorY = float32(inputs.cursorY)
        }
    }
}

func (game *Game) simulate(inputs *Inputs) {
    game.state.step()

    player := int32(game.state.cur) & 3
    mode := int32(game.state.cur) & ^3

    if mode == PLAYER_TURN {
        game.simulatePlayerTurn(inputs, player)
    } else if mode == SCORING_TURN {
        game.simulateScoringTurn(inputs, player)
    }
}

func (game *Game) simulatePlayerTurn(inputs *Inputs, playerIdx int32) {
    game.shuffleTimer--
    if game.shuffleTimer > 0 {
        return
    }
    game.shuffleTimer = 0

    p := &game.players[playerIdx]

    for _, char := range inputs.pressedChars {
        idx := int8(0)
        if char >= 'a' && char <= 'z' {
            idx = int8(char - 0x60)
        } else if char >= 'A' && char <= 'Z' {
            idx = int8(char - 0x40)
        }
        if idx == 0 {
            continue
        }

        deckTiles := p.deckTilesBits.cur
        pos := -1
        for j := 0; j < 7; j++ {
            if ((deckTiles >> ((7-j-1)*8)) & 0x7f) == uint64(idx & 0x7f) {
                pos = j
                break
            }
        }

        isBlank := int8(0)
        if pos < 0 {
            for j := 0; j < 7; j++ {
                if ((deckTiles >> ((7-j-1)*8)) & 0x7f) == uint64(27) {
                    pos = j
                    isBlank = 0x20
                    break
                }
            }
        }

        if pos >= 0 {
            if pos == 0 {
                p.deckTilesBits.cur &= 0xffFFffFFffFF
            } else if pos == 6 {
                p.deckTilesBits.cur >>= 8
            } else {
                topMask := ((uint64(1) << (8*pos)) - 1) << ((7-pos)*8)
                bottomMask := (uint64(1) << ((6-pos)*8)) - 1
                p.deckTilesBits.cur = (p.deckTilesBits.cur & topMask) >> 8 | (p.deckTilesBits.cur & bottomMask)
            }
            p.turnLetters[p.nTilesHeld] = idx | isBlank
            p.turnPositions[p.nTilesHeld] = 0
            p.nTilesHeld++
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
            deckTiles := p.deckTilesBits.cur
            for j := 0; j < 7; j++ {
                if ((deckTiles >> (j*8)) & 0x7f) == 0 {
                    p.nTilesHeld--
                    tile := uint64(p.turnLetters[p.nTilesHeld])
                    if (tile & 0x20) != 0 {
                        tile = 27
                    }
                    p.deckTilesBits.cur |= tile << (j*8)
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
            shouldShuffle = true
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

    game.updateCursor(inputs)

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

            offset := 0
            for i := 0; i < nHeld; i++ {
                offset += int((offsetBits >> ((nHeld-i-1)*4)) & 0xf)
                x = col + xInc * (i + offset)
                y = row + yInc * (i + offset)
                if didPlace {
                    game.boardTiles[x + 15 * y] = p.turnLetters[i]
                    p.turnLetters[i] = 0
                }
                p.turnPositions[i] = uint8((x + 15 * y) + 1) // +1 for sentinel value
            }
        }
    }

    if didPlace {
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
        p.nTilesHeld = 0
        p.turnOffsetsBits.cur = 0

        score := game.findNewWords(p)
        allWordsAreValid := true

        if game.menu.shouldValidateEveryWord {
            for i := 0; i < len(game.scoringWords); i++ {
                _, exists := game.wordMap[game.scoringWords[i]]
                if !exists {
                    // TODO: save information about this word, and the other new invalid words, to display that they're not valid
                    allWordsAreValid = false
                    break
                }
            }
        }

        if allWordsAreValid {
            p.turnScore = int32(score)
            game.state.animPos = 0
            game.state.animLen = max(p.deckTilesBits.animLen, int32(len(game.scoringCommands) * TILE_SCORE_DURATION))
            game.state.cur = uint64(SCORING_TURN | (playerIdx & 3))
            game.simulateScoringTurn(inputs, playerIdx)
        } else {
            game.scoringWords = game.scoringWords[0:0]
            game.scoringCommands = game.scoringCommands[0:0]
        }
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

    p.turnState.step()
    p.turnOffsetsBits.step()
}

func (game *Game) simulateScoringTurn(inputs *Inputs, playerIdx int32) {
    p := &game.players[playerIdx]

    p.deckTilesBits.step()
    if game.state.animLen == 0 && p.deckTilesBits.animLen == 0 {
        p.totalScore += p.turnScore
        p.turnScore = 0

        for i := 0; i < 7; i++ {
            p.turnLetters[i] = 0
            p.turnPositions[i] = 0
        }
        game.scoringWords = game.scoringWords[0:0]
        game.scoringCommands = game.scoringCommands[0:0]

        game.updateCursor(inputs)

        nextPlayer := (playerIdx + 1) % 4
        for i := 0; i < 3; i++ {
            if game.players[nextPlayer].kind != PLAYER_INACTIVE {
                break
            }
            nextPlayer = (nextPlayer + 1) % 4
        }

        game.state.cur = uint64(PLAYER_TURN | nextPlayer)
        game.state.animPos = 0
        game.state.animLen = 80
    }
}

func (game *Game) findNewWords(p *Player) (totalScore int) {
    game.scoringWords = game.scoringWords[0:0]
    game.scoringCommands = game.scoringCommands[0:0]
    game.activeLines = game.activeLines[0:0]

    for i := 0; i < 7; i++ {
        pos := int(p.turnPositions[i]) - 1
        if pos < 0 {
            continue
        }
        x := pos % 15
        y := pos / 15

        for d := 0; d < 4; d++ {
            dx := ((d & 2) - 1) * (d & 1)
            dy := (((d+1) & 2) - 1) * ((d+1) & 1)
            if x + dx < 0 || x + dx >= 15 || y + dy < 0 || y + dy >= 15 {
                continue
            }

            pos2 := (x + dx) + 15 * (y + dy)
            if game.boardTiles[pos2] == 0 {
                continue
            }

            // look in opposite direction for both ends until there is no tile at that position or its off the board
            xx := x - dx
            yy := y - dy
            for xx >= 0 && xx < 15 && yy >= 0 && yy < 15 && game.boardTiles[xx + 15 * yy] != 0 {
                xx -= dx
                yy -= dy
            }
            start := (xx + dx) + 15 * (yy + dy)

            xx = x + dx
            yy = y + dy
            for xx >= 0 && xx < 15 && yy >= 0 && yy < 15 && game.boardTiles[xx + 15 * yy] != 0 {
                xx += dx
                yy += dy
            }
            end := (xx - dx) + 15 * (yy - dy)

            line := uint16(min(start, end) + (15 * 15 * max(start, end)))
            exists := false
            for j := 0; j < len(game.activeLines); j++ {
                if game.activeLines[j] == line {
                    exists = true
                    break
                }
            }

            if !exists {
                game.activeLines = append(game.activeLines, line)
            }
        }
    }

    totalScore = 0

    for i := 0; i < len(game.activeLines); i++ {
        start := int32(game.activeLines[i]) % (15 * 15)
        end   := int32(game.activeLines[i]) / (15 * 15)
        xInc := int32(1)
        yInc := int32(0)
        if start % 15 == end % 15 {
            xInc, yInc = yInc, xInc
        }

        game.wordBuilder.Reset()
        wordScore := 0
        wordMultipliers := uint64(0) // the maximum number of word multipliers per line is 3, so uint64 is fine here
        pos := start

        for pos <= end {
            isNewTile := false
            for j := 0; j < 7; j++ {
                if pos == int32(p.turnPositions[j]) - 1 {
                    isNewTile = true
                    break
                }
            }

            tile := int32(game.boardTiles[pos])
            game.wordBuilder.WriteByte(byte(0x40 + (tile & 0x1f)))

            letterScore := int32(0)
            if tile < 32 {
                letterScore = tiles[tile - 1].points
            }
            game.scoringCommands = append(game.scoringCommands, uint16(pos << 8 | letterScore))

            if isNewTile {
                tt := getTileType(int32(pos % 15), int32(pos / 15))
                cmd := uint16(pos << 8 | BONUS)
                if tt == TRIPLE_LETTER || tt == TRIPLE_WORD {
                    cmd |= 3
                } else {
                    cmd |= 2
                }
                if tt == DOUBLE_LETTER || tt == TRIPLE_LETTER {
                    game.scoringCommands = append(game.scoringCommands, cmd)
                    letterScore *= int32(cmd & 3)
                } else if tt == DOUBLE_WORD || tt == TRIPLE_WORD {
                    wordMultipliers = wordMultipliers << 16 | uint64(cmd)
                }
            }

            wordScore += int(letterScore)
            pos += xInc + 15 * yInc
        }

        game.scoringWords = append(game.scoringWords, game.wordBuilder.String())

        for wordMultipliers != 0 {
            game.scoringCommands = append(game.scoringCommands, uint16(wordMultipliers & 0xffff))
            wordScore *= int(wordMultipliers & 3)
            wordMultipliers >>= 16
        }

        totalScore += wordScore
    }

    usedAllTilesInDeck := true
    for i := 0; i < 7; i++ {
        if p.turnPositions[i] == 0 {
            usedAllTilesInDeck = false
            break
        }
    }

    if usedAllTilesInDeck {
        cmd := uint16((int(p.turnPositions[6]) - 1) << 8 | 50)
        game.scoringCommands = append(game.scoringCommands, cmd)
        totalScore += 50
    }

    return totalScore
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
