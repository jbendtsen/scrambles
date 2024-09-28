package main

type MainMenu struct {
	nPlayers int
	timeLimitSecs int
}

type Player struct {
	deckTiles []int8
}

type Inputs struct {
    pressedKeys []int32
    pressedChars []int32
    buttons []int32
    cursorX int32
    cursorY int32
}

type Game struct {
	menu MainMenu
	players [4]Player

	bagMap []int32
	bagChars []byte
	wordsList []string
	wordMap map[string]int
	boardTiles []int8

	startupTimestamp int64
	prevHash64 uint64
	frameCounter int64

    prevMode int32
    curMode int32
    modeAnimPos int32
    modeAnimLen int32

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

func (game *Game) takeTileFromBag() Tile {
	if len(game.bagMap) == 0 {
		return Tile{}
	}

	idx := int(game.getRandom(int64(len(game.bagMap))))
	selected := game.bagMap[idx]
	game.bagMap[idx] = game.bagMap[len(game.bagMap)-1]
	//game.bagMap[len(game.bagMap)-1] = selected

	game.bagMap = game.bagMap[:len(game.bagMap)-1]
	ch := game.bagChars[selected]
	if ch == ' ' {
		return tiles[26]
	}
	return tiles[ch - 0x41]
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

	for i := 0; i < 4; i++ {
		game.players[i].deckTiles = make([]int8, 7)
	}
}

func (game *Game) start() {
    game.bagChars, game.bagMap = generateTileBag()
	for i := 0; i < 15 * 15; i++ {
		game.boardTiles[i] = 0
	}
	for i := 0; i < 4 * 7; i++ {
		game.players[i / 7].deckTiles[i % 7] = 0
	}
}

func (game *Game) getRandom(endExclusive int64) int64 {
	upper, lower := generateNext128(game.startupTimestamp, game.frameCounter, game.prevHash64)
	game.prevHash64 = upper ^ lower

	value := int64(lower & ^(uint64(1) << 63))
	if endExclusive > 0 {
		value = value % endExclusive
	}
	return value
}

func makeInputs() Inputs {
    inputs := Inputs{}
    inputs.pressedKeys  = make([]int32, 0, 16)
    inputs.pressedChars = make([]int32, 0, 16)
    inputs.buttons = make([]int32, 2)
    return inputs
}
