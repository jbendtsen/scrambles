package main

import (
	"io"
	"os"
	"fmt"
	"math"
	"time"
	"strings"
	"image/color"
	rl "github.com/gen2brain/raylib-go/raylib"
)

const fps = 60

const KEY_BACKSPACE = rl.KeyBackspace
const KEY_UP = rl.KeyUp
const KEY_DOWN = rl.KeyDown
const KEY_LEFT = rl.KeyLeft
const KEY_RIGHT = rl.KeyRight

type Textures struct {
	board rl.Texture2D
	tiles rl.Texture2D
	letterGlyphs []rl.GlyphInfo
	numberGlyphs []rl.GlyphInfo
	fontData []byte
	tileDisplaySize int
}

var boardTileColorsRgba = [...]uint32 {
    0x00902cff,
    0xffc020ff,
    0xe00000ff,
    0x80d0ffff,
    0x00a0e0ff,
}

func updateColor(color *color.RGBA, rgba uint32) {
	color.R = uint8((rgba >> 24) & 0xff)
	color.G = uint8((rgba >> 16) & 0xff)
	color.B = uint8((rgba >> 8) & 0xff)
	color.A = uint8(rgba & 0xff)
}

func secsToFrames(seconds int) int {
    return seconds * fps
}

func drawMenu(game *Game, inputs *Inputs, isMenuActive bool) (shouldStartGame bool) {
    // menu drawing goes here

	if !isMenuActive {
		return false
	}

    game.menu.nPlayers = 2
    game.menu.timeLimitSecs = 120

	shouldStartGame = false
	if (inputs.buttons[0] & 1) == 1 {
		shouldStartGame = true
	}

	return shouldStartGame
}

func drawBoard(game *Game, boardTex rl.Texture2D, tBoardFall float32) {
	if tBoardFall == 0.0 {
		return
	}

    boardLen := game.tileSize * 15
    var xOff int32 = int32(game.wndWidth - boardLen) / 2
    var yOff int32 = int32(game.wndHeight - boardLen - 2 * game.tileSize) / 2

	it := (1.0 - tBoardFall)
	it2 := it * it
	t2 := tBoardFall * tBoardFall
	t4 := t2 * t2

	scale := float32(1.0 + 2.0 * it2 * float32(math.Abs(math.Cos(float64(t4 * 2.0 * math.Pi)))))
	dScaled := float32(boardLen) * scale

	origin := rl.Vector2{dScaled * 0.5, dScaled * 0.5}
	srcRect := rl.Rectangle{0, 0, float32(boardLen), float32(boardLen)}
	dstRect := rl.Rectangle{float32(xOff) + float32(boardLen) * 0.5, float32(yOff) + float32(boardLen) * 0.5, dScaled, dScaled}

	boardRotation := it2 * 30.0

	boardTint := rl.White
	boardTint.A = uint8(int(t2 * 255.0) & 0xff)

	rl.DrawTexturePro(boardTex, srcRect, dstRect, origin, boardRotation, boardTint)
}

func drawGame(game *Game, textures *Textures, inputs *Inputs) (isGameOver bool) {
    game.simulate(inputs)

    tileW := float32(textures.tileDisplaySize)
    rect := rl.Rectangle{0, 0, tileW, tileW}
    pos := rl.Vector2{}

    tileSize := int(game.tileSize)
    boardLen := tileSize * 15
    xBoardOff := (int(game.wndWidth) - boardLen) / 2
    yBoardOff := (int(game.wndHeight) - boardLen - 2 * tileSize) / 2
    tileOff := (tileSize - textures.tileDisplaySize) / 2

    for i := 0; i < 15 * 15; i++ {
        tileIndex := int(game.boardTiles[i]) - 1
        if tileIndex < 0 {
            continue
        }
        x := i % 15
        y := i / 15
        pos.X = float32(xBoardOff + tileOff + (x * tileSize))
        pos.Y = float32(yBoardOff + tileOff + (y * tileSize))
        rect.X = float32((tileIndex % 9) * textures.tileDisplaySize)
        rect.Y = float32((tileIndex / 9) * textures.tileDisplaySize)
        rl.DrawTextureRec(textures.tiles, rect, pos, rl.White)
    }

    mode := game.state.cur & ^3
    player := game.state.cur & 3
    if mode == PICK_ORDER {
        // TODO
    } else if mode == PLAYER_TURN {
        t := float32(1.0)
        if game.state.animLen > 0 {
            t = float32(game.state.animPos) / float32(game.state.animLen)
            prev := game.state.prev & ^3
            if prev >= PLAYER_TURN {
                drawDeck(game, textures, game.state.prev & 3, 1.0 + t, rect)
            }
        }
        drawDeck(game, textures, player, t, rect)
        drawTurn(game, textures, inputs, player, rect)
    } else if mode == TURN_SCORING {
        drawDeck(game, textures, player, 1.0, rect)
        t := float32(1.0)
        if game.state.animLen > 0 {
            t = float32(game.state.animPos) / float32(game.state.animLen)
        }
        drawScoring(game, textures, player, t, rect)
    }

	isGameOver = (inputs.buttons[0] & 1) == 1
	return isGameOver
}

func drawDeck(game *Game, textures *Textures, playerIdx int32, tOpening float32, tileRect rl.Rectangle) {
    dstRect := tileRect
    dstRect.Width *= 1.4
    dstRect.Height *= 1.4

    it2 := (1.0 - tOpening) * (1.0 - tOpening)

	deckPadding := tileRect.Width * 0.4
	tilesSpan := int32(7.0 * dstRect.Width + 6.0 * deckPadding)
	deckSidePad := int32(float64(tilesSpan) * 0.05)

    boardLen := int(game.tileSize) * 15
    //xBoardOff := (int(game.wndWidth) - boardLen) / 2
    yBoardOff := (int(game.wndHeight) - boardLen - 2 * int(game.tileSize)) / 2

    leftoverH := int32(int(game.wndHeight) - yBoardOff - boardLen)
	deckW := 2 * deckSidePad + tilesSpan
	deckH := int32(dstRect.Height + deckPadding)
    deckX := int32((1.0 - it2) * float32(game.wndWidth)) - ((game.wndWidth + deckW) / 2)
    deckY := int32(yBoardOff + boardLen) + (leftoverH / 2) - (deckH / 2)
    rl.DrawRectangle(deckX, deckY, deckW, deckH, color.RGBA{0, 64, 16, 255})

    origin := rl.Vector2{}

    for i := 0; i < 7; i++ {
        tileIndex := int(game.players[playerIdx].deckTiles[i]) - 1
		if tileIndex < 0 {
			continue
		}
        tileRect.X = float32((tileIndex % 9) * textures.tileDisplaySize)
        tileRect.Y = float32((tileIndex / 9) * textures.tileDisplaySize)
        dstRect.X = float32(deckX + deckSidePad) + float32(i) * (dstRect.Width + deckPadding)
        dstRect.Y = float32(deckY) - deckPadding
        rl.DrawTexturePro(textures.tiles, tileRect, dstRect, origin, 0.0, rl.White)
    }
}

func drawTurn(game *Game, textures *Textures, inputs *Inputs, playerIdx int32, tileRect rl.Rectangle) {
    origin := rl.Vector2{}

    tTurnRot := float64(0.0)
    if game.turnRotation.animLen > 0 {
        tTurnRot = float64(game.turnRotation.animPos) / float64(game.turnRotation.animLen)
    }
    if game.turnRotation.prev == 0 {
        tTurnRot = 1.0 - tTurnRot
    }

    dstRect := tileRect
    xDir := float32(math.Abs(math.Cos(0.5 * math.Pi * tTurnRot)))
    yDir := float32(math.Abs(math.Sin(0.5 * math.Pi * tTurnRot)))

    holdPos := int32(0)
    for i := int32(0); i < game.players[playerIdx].nTilesHeld; i++ {
        tileIndex := int(game.players[playerIdx].turnTiles[i]) - 1
		if tileIndex < 0 {
			break
		}
		tileRect.X = float32((tileIndex % 9) * textures.tileDisplaySize)
        tileRect.Y = float32((tileIndex / 9) * textures.tileDisplaySize)
        dstRect.X = float32(inputs.cursorX - game.tileSize / 2) + xDir * float32(holdPos * game.tileSize)
        dstRect.Y = float32(inputs.cursorY - game.tileSize / 2) + yDir * float32(holdPos * game.tileSize)
        rl.DrawTexturePro(textures.tiles, tileRect, dstRect, origin, 0.0, rl.White)
        holdPos += 1
    }
}

func drawScoring(game *Game, textures *Textures, playerIdx int32, tOpening float32, tileRect rl.Rectangle) {
    // TODO
}

func maybeRecreateBoard(tex *rl.Texture2D, wndWidth, wndHeight, oldTileSize int32) (tileSize int32) {
	tileSize = int32(min(wndWidth / 16, wndHeight / 20))
    if tileSize == oldTileSize {
        return tileSize
    }

	c := color.RGBA{}
	updateColor(&c, boardTileColorsRgba[0])

	boardLen := int(tileSize * 15)
	canvas := rl.GenImageColor(boardLen, boardLen, c)

    topTri    := Triangle{tileSize / 2, -tileSize / 6, 2 * tileSize / 3, 0, tileSize / 3, 0}
    leftTri   := Triangle{-tileSize / 6, tileSize / 2, 0, tileSize / 3, 0, 2 * tileSize / 3}
    rightTri  := Triangle{tileSize, tileSize / 3, 7 * tileSize / 6, tileSize / 2, tileSize, 2 * tileSize / 3}
    bottomTri := Triangle{tileSize / 3, tileSize, 2 * tileSize / 3, tileSize, tileSize / 2, 7 * tileSize / 6}

    // draw tiles
    for y := int32(0); y < 15; y++ {
        for x := int32(0); x < 15; x++ {
            tt := getTileType(x, y)
            if tt != 0 {
                rgba := boardTileColorsRgba[tt]
                updateColor(&c, rgba)
                rl.ImageDrawRectangle(canvas, x * tileSize, y * tileSize, tileSize, tileSize, c)
                renderTriangle(canvas.Data, tileSize * 15, tileSize * 15, x * tileSize, y * tileSize, &topTri, rgba)
                renderTriangle(canvas.Data, tileSize * 15, tileSize * 15, x * tileSize, y * tileSize, &leftTri, rgba)
                renderTriangle(canvas.Data, tileSize * 15, tileSize * 15, x * tileSize, y * tileSize, &rightTri, rgba)
                renderTriangle(canvas.Data, tileSize * 15, tileSize * 15, x * tileSize, y * tileSize, &bottomTri, rgba)
            }
        }
    }

    // draw gridlines
    lineColor := color.RGBA{192, 240, 255, 255}
    lineWidth := tileSize / 12
    lineOff := tileSize - (lineWidth / 2)
    for i := int32(0); i < 14; i++ {
        rl.ImageDrawRectangle(canvas, 0, i * tileSize + lineOff, tileSize * 15, lineWidth, lineColor)
        rl.ImageDrawRectangle(canvas, i * tileSize + lineOff, 0, lineWidth, tileSize * 15, lineColor)
    }

	if tex.ID != 0 {
		rl.UnloadTexture(*tex)
	}

    *tex = rl.LoadTextureFromImage(canvas)
    rl.UnloadImage(canvas)

    return tileSize
}

func updateTextures(textures *Textures, wndWidth, wndHeight, oldTileSize int32) (tileSize int32) {
	tileSize = maybeRecreateBoard(&textures.board, wndWidth, wndHeight, oldTileSize)
	if tileSize == oldTileSize {
		return tileSize
	}

    if textures.letterGlyphs != nil {
        rl.UnloadFontData(textures.letterGlyphs)
    }
    if textures.numberGlyphs != nil {
        rl.UnloadFontData(textures.numberGlyphs)
    }

	letterCodePoints := make([]int32, 26)
	numberCodePoints := make([]int32, 10)
	for i := 0; i < 26; i++ {
		letterCodePoints[i] = int32(0x41 + i)
	}
	for i := 0; i < 10; i++ {
		numberCodePoints[i] = int32(0x30 + i)
	}

	textures.letterGlyphs = rl.LoadFontData(
		textures.fontData,
		int32(float64(tileSize) * 0.9),
		letterCodePoints,
		rl.FontDefault,
	)
	textures.numberGlyphs = rl.LoadFontData(
		textures.fontData,
		int32(float64(tileSize) * 0.33),
		numberCodePoints,
		rl.FontDefault,
	)

    for i := 0; i < 26; i++ {
        img := &textures.letterGlyphs[i].Image
        rl.ImageFormat(img, rl.UncompressedR8g8b8a8)
        setAlphaToBrightness(img.Data, img.Width, img.Height)
    }
    for i := 0; i < 10; i++ {
        img := &textures.numberGlyphs[i].Image
        rl.ImageFormat(img, rl.UncompressedR8g8b8a8)
        setAlphaToBrightness(img.Data, img.Width, img.Height)
    }

    tileDisplaySize := min(tileSize - 1, int32(float64(tileSize) * 0.95))
    tilesImage := rl.GenImageColor(int(9 * tileDisplaySize), int(3 * tileDisplaySize), rl.White)

    textures.tileDisplaySize = int(tileDisplaySize)

    scores := getLetterScores()
    srcRect := rl.Rectangle{}
    dstRect := rl.Rectangle{}
    for i := int32(0); i < 26; i++ {
        letter := &textures.letterGlyphs[i].Image
        lift := -float32(letter.Height) * 0.15
        dstRect.X = float32(((i % 9) * tileDisplaySize) + (tileDisplaySize - letter.Width) / 2)
        dstRect.Y = lift + float32(((i / 9) * tileDisplaySize) + (tileDisplaySize - letter.Height) / 2)
        dstRect.Width  = float32(letter.Width)
        dstRect.Height = float32(letter.Height)
        srcRect.Width  = dstRect.Width
        srcRect.Height = dstRect.Height
        rl.ImageDraw(tilesImage, letter, srcRect, dstRect, rl.Black)

        points := scores[i]
        number := &textures.numberGlyphs[points % 10].Image
        corner := 2 * tileDisplaySize / 3
        dstRect.X = float32(((i % 9) * tileDisplaySize) + corner + (tileDisplaySize / 3 - number.Width) / 2)
        dstRect.Y = float32(((i / 9) * tileDisplaySize) + corner + (tileDisplaySize / 3 - number.Height) / 2)
        dstRect.Width  = float32(number.Width)
        dstRect.Height = float32(number.Height)
        srcRect.Width  = dstRect.Width
        srcRect.Height = dstRect.Height
        rl.ImageDraw(tilesImage, number, srcRect, dstRect, rl.Black)

        if points >= 10 {
            dstRect.X -= float32(number.Height)

            number = &textures.numberGlyphs[(points / 10) % 10].Image
            dstRect.Width  = float32(number.Width)
            dstRect.Height = float32(number.Height)
            srcRect.Width  = dstRect.Width
            srcRect.Height = dstRect.Height
            rl.ImageDraw(tilesImage, number, srcRect, dstRect, rl.Black)
        }
    }

    tdsInt := int(tileDisplaySize)
    roundTileEdges(tilesImage.Data, 9, 3, tdsInt, tdsInt, int(float64(tdsInt) * 0.15))

    if textures.tiles.ID > 0 {
        rl.UnloadTexture(textures.tiles)
    }

    textures.tiles = rl.LoadTextureFromImage(tilesImage)
    rl.UnloadImage(tilesImage)

    return tileSize
}

func updateInputs(inputs *Inputs) {
    inputs.pressedKeys = inputs.pressedKeys[0:0]
    inputs.pressedChars = inputs.pressedChars[0:0]
    for true {
        key := rl.GetKeyPressed()
        if key == 0 {
            break
        }
        inputs.pressedKeys = append(inputs.pressedKeys, key)
    }
    for true {
        char := rl.GetCharPressed()
        if char == 0 {
            break
        }
        inputs.pressedChars = append(inputs.pressedChars, char)
    }

    inputs.cursorX = rl.GetMouseX()
    inputs.cursorY = rl.GetMouseY()
    for i := 0; i < 2; i++ {
        flags := int32(0)
        if rl.IsMouseButtonPressed(0) {
            flags |= 1
        }
        if rl.IsMouseButtonDown(0) {
            flags |= 2
        }
        if rl.IsMouseButtonReleased(0) {
            flags |= 4
        }
        inputs.buttons[i] = flags
    }
}

func loadTtfData() []byte {
	entries, err := os.ReadDir("assets/")
	if err != nil {
		fmt.Println("Could not open assets folder")
		return nil
	}

	var ttfName string
	for i := 0; i < len(entries); i++ {
		fname := entries[i].Name()
		if strings.HasSuffix(fname, ".ttf") {
			ttfName = fname
			break
		}
	}

	if ttfName == "" {
		fmt.Println("Could not a ttf file in the assets folder")
		return nil
	}

	f, err := os.Open("assets/" + ttfName)
	if err != nil {
		fmt.Println("Could not open assets/" + ttfName)
		return nil
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		fmt.Println("Could not read font from assets/" + ttfName)
		return nil
	}

	return data
}

func loadWords() []string {
	wordsFile, err := os.Open("assets/all-words.txt")
	if err != nil {
		fmt.Println("Could not open assets/all-words.txt")
		return nil
	}
	defer wordsFile.Close()

	wordsBytes, err := io.ReadAll(wordsFile)
	if err != nil {
		fmt.Println("Could not read word list from assets/all-words.txt")
		return nil
	}

	return strings.Split(string(wordsBytes), "\n")
}

func main() {
	wordsList := loadWords()
	if wordsList == nil {
		return
	}

	game := Game{}
	game.init(wordsList, time.Now().UnixMilli())

	ttfData := loadTtfData()
	if ttfData == nil {
		return
	}

	rl.SetConfigFlags(rl.FlagWindowResizable)
	rl.InitWindow(800, 450, "scrambles")
	defer rl.CloseWindow()

	rl.SetTargetFPS(fps)

	textures := Textures{}
	textures.fontData = ttfData

    inputs := makeInputs()

    gameStarted := false
	isGameOver := false
	openingTimer := 0
	const maxOpeningTime = 120

	for !rl.WindowShouldClose() {
	    w := int32(rl.GetRenderWidth())
		h := int32(rl.GetRenderHeight())
		if game.wndWidth != w || game.wndHeight != h {
			game.wndWidth = w
			game.wndHeight = h
			game.tileSize = updateTextures(&textures, w, h, game.tileSize)
		}

        updateInputs(&inputs)

		rl.BeginDrawing()
		rl.ClearBackground(color.RGBA{0, 0x68, 0x30, 0xff})

	    tBoardFall := float32(1.0)
	    if !gameStarted || openingTimer < maxOpeningTime {
		    if drawMenu(&game, &inputs, openingTimer == 0) {
				game.start()
				isGameOver = false
				gameStarted = true
			}
			tBoardFall = float32(openingTimer) / float32(maxOpeningTime)
	    }

		if gameStarted {
			drawBoard(&game, textures.board, tBoardFall)

			if !isGameOver {
				openingTimer += 1
			}
			if openingTimer >= maxOpeningTime {
				isGameOver = drawGame(&game, &textures, &inputs)
				openingTimer = maxOpeningTime
			}
		}

		if isGameOver {
			openingTimer -= 2
			if openingTimer <= 0 {
				openingTimer = 0
				gameStarted = false
			}
		}

		rl.EndDrawing()
		game.frameCounter += 1
	}
}
