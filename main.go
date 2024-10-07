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
const KEY_RETURN = rl.KeyEnter
const KEY_LSHIFT = rl.KeyLeftShift
const KEY_RSHIFT = rl.KeyRightShift
const KEY_LCTRL = rl.KeyLeftControl
const KEY_RCTRL = rl.KeyRightControl
const KEY_TAB = rl.KeyTab
const KEY_UP = rl.KeyUp
const KEY_DOWN = rl.KeyDown
const KEY_LEFT = rl.KeyLeft
const KEY_RIGHT = rl.KeyRight

const ARROW_UP = 0
const ARROW_DOWN = 1
const ARROW_LEFT = 2
const ARROW_RIGHT = 3

type Textures struct {
	board rl.Texture2D
	tilesSmall rl.Texture2D
	tilesLarge rl.Texture2D
	tileHl rl.Texture2D
	tileCursor rl.Texture2D
	letterGlyphsSmall []rl.GlyphInfo
	letterGlyphsLarge []rl.GlyphInfo
	numberGlyphsSmall []rl.GlyphInfo
	numberGlyphsLarge []rl.GlyphInfo
	fontData []byte
	smallTileSize int
	largeTileSize int
	tileHlBorderSize int
}

var boardTileColorsRgba = [...]uint32 {
    0x00902cff,
    0xffc020ff,
    0xe00000ff,
    0x80d0ffff,
    0x00a0e0ff,
}

var playerDeckColors = [...]color.RGBA {
    {128, 0, 0, 255},
    {0, 0, 192, 255},
    {128, 128, 0, 255},
    {0, 64, 16, 255},
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
	if (inputs.mouseButtons[0] & 1) == 1 {
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

    mode := int32(game.state.cur) & ^3
    player := int32(game.state.cur) & 3

    tileW := float32(textures.smallTileSize)
    rect := rl.Rectangle{0, 0, tileW, tileW}
    pos := rl.Vector2{}

    tileSize := int(game.tileSize)
    boardLen := tileSize * 15
    xBoardOff := (int(game.wndWidth) - boardLen) / 2
    yBoardOff := (int(game.wndHeight) - boardLen - 2 * tileSize) / 2
    tileOff := (tileSize - textures.smallTileSize) / 2

    if mode == PLAYER_TURN {
        boardCurX := int(game.turnCursorX) - xBoardOff
        boardCurY := int(game.turnCursorY) - yBoardOff
        col := boardCurX / tileSize
        row := boardCurY / tileSize
        if boardCurX >= 0 && boardCurY >= 0 && col >= 0 && col < 15 && row >= 0 && row < 15 {
            xHl := int32(xBoardOff + (col * tileSize) - textures.tileHlBorderSize)
            yHl := int32(yBoardOff + (row * tileSize) - textures.tileHlBorderSize)
            rl.DrawTexture(textures.tileHl, xHl, yHl, rl.White)
        }

        if game.shuffleTimer > 0 {
            t := float32(game.shuffleTimer) / float32(SHUFFLE_DURATION)
            drawDeck(game, textures, player, t, DECK_SHUFFLE)
        } else {
            t := float32(1.0)
            if game.state.animLen > 0 {
                t = float32(game.state.animPos) / float32(game.state.animLen)
                prev := int32(game.state.prev) & ^3
                if prev >= PLAYER_TURN {
                    drawDeck(game, textures, int32(game.state.prev) & 3, -(1.0 + t), DECK_OPENING)
                }
            }
            drawDeck(game, textures, player, t, DECK_OPENING)
        }
    }

    for i := 0; i < 15 * 15; i++ {
        tileIndex := int(game.boardTiles[i]) - 1
        if tileIndex < 0 {
            continue
        }
        x := i % 15
        y := i / 15
        pos.X = float32(xBoardOff + tileOff + (x * tileSize))
        pos.Y = float32(yBoardOff + tileOff + (y * tileSize))
        rect.X = float32((tileIndex % 9) * textures.smallTileSize)
        rect.Y = float32((tileIndex / 9) * textures.smallTileSize)
        rl.DrawTextureRec(textures.tilesSmall, rect, pos, rl.White)
    }

    if mode == PICK_ORDER {
        // TODO
    } else if mode == PLAYER_TURN {
        if game.players[player].nTilesHeld == 0 {
            rl.DrawTexture(textures.tileCursor, int32(game.turnCursorX - tileW * 0.5), int32(game.turnCursorY - tileW * 0.5), rl.White)
        }
        drawTurn(game, textures, inputs, player, rect)
    } else if mode == SCORING_TURN {
        //tRefill := game.players[player].deckTilesBits.getPosition()
        drawDeck(game, textures, player, 1.0, DECK_REFILL)

        tScoring := game.state.getPositionOr(0.0)
        drawScoring(game, textures, player, tScoring, rect)
    }

	isGameOver = false
	return isGameOver
}

func drawDeck(game *Game, textures *Textures, playerIdx int32, t float32, animMode int32) {
    tileRect := rl.Rectangle{0, 0, float32(textures.largeTileSize), float32(textures.largeTileSize)}
    dstRect := tileRect

    it2 := float32(0.0)
    if animMode == DECK_OPENING {
        if t < 0.0 {
            it2 = (1.0 + t) * (1.0 + t)
            it2 = -it2
        } else {
            it2 = (1.0 - t) * (1.0 - t)
        }
    }

	deckPadding := tileRect.Width * 0.25
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
    rl.DrawRectangle(deckX, deckY, deckW, deckH, playerDeckColors[playerIdx])

    p := &game.players[playerIdx]
    origin := rl.Vector2{}

    if animMode == DECK_SHUFFLE {
        for i := 0; i < 7; i++ {
            prevTile := int((p.deckTilesBits.prev >> ((7-i-1)*8)) & 0x7f) - 1
            if prevTile < 0 {
                continue
            }

            //xOff := float32(0.0)
            //curTile := int((p.deckTilesBits.cur >> ((7-i-1)*8)) & 0x7f) - 1
            //if prevTile != curTile {
            dstIdx := 6 - game.shuffleBuf[i]
            xOff := float32(i) * t + float32(dstIdx) * (1.0 - t)

            tileRect.X = float32((prevTile % 9) * textures.largeTileSize)
            tileRect.Y = float32((prevTile / 9) * textures.largeTileSize)
            dstRect.X = float32(deckX + deckSidePad) + xOff * (dstRect.Width + deckPadding)
            dstRect.Y = float32(deckY)
            rl.DrawTexturePro(textures.tilesLarge, tileRect, dstRect, origin, 0.0, rl.White)
        }
    } else {
        for i := 0; i < 7; i++ {
            t = 0.0
            tileIndex := int((p.deckTilesBits.prev >> ((7-i-1)*8)) & 0x7f) - 1
		    if tileIndex < 0 {
		        tileIndex = int((p.deckTilesBits.cur >> ((7-i-1)*8)) & 0x7f) - 1
		        if tileIndex < 0 {
			        continue
		        }
		        if p.deckTilesBits.animLen > 0 {
		            t = min(float32(p.deckTilesBits.animPos - int32(i*2)) / float32(p.deckTilesBits.animLen - 14), 1.0)
		            t = (1.0 - t) * (1.0 - t)
	            }
		    }

            tileRect.X = float32((tileIndex % 9) * textures.largeTileSize)
            tileRect.Y = float32((tileIndex / 9) * textures.largeTileSize)
            dstRect.X = float32(deckX + deckSidePad) + float32(i) * (dstRect.Width + deckPadding)
            dstRect.Y = float32(deckY) + (t * dstRect.Height * 5.0)
            rl.DrawTexturePro(textures.tilesLarge, tileRect, dstRect, origin, 0.0, rl.White)
        }
    }
}

func drawTurn(game *Game, textures *Textures, inputs *Inputs, playerIdx int32, tileRect rl.Rectangle) {
    origin := rl.Vector2{}

    p := &game.players[playerIdx]
    tTurnRot := float64(p.turnState.getPositionOr(0.0))

    dstRect := tileRect
    xDir := float32(math.Abs(math.Cos(0.5 * math.Pi * tTurnRot)))
    yDir := float32(math.Abs(math.Sin(0.5 * math.Pi * tTurnRot)))

    if p.turnState.prev == ROTA_VERT {
        xDir, yDir = yDir, xDir
    }

    offsetCur := float32(0)
    offsetPrev := float32(0)
    nHeld := p.nTilesHeld

    for i := int32(0); i < nHeld; i++ {
        tileIndex := int(p.turnTiles[i]) - 1
		if tileIndex < 0 {
			break
		}
		offsetCur  += float32((p.turnOffsetsBits.cur >> ((nHeld-i-1)*4)) & 0xf)
		offsetPrev += float32((p.turnOffsetsBits.prev >> ((nHeld-i-1)*4)) & 0xf)
	    pos := float32(i) + float32(tTurnRot) * offsetCur + float32(1.0 - tTurnRot) * offsetPrev

		tileRect.X = float32((tileIndex % 9) * textures.smallTileSize)
        tileRect.Y = float32((tileIndex / 9) * textures.smallTileSize)
        dstRect.X = game.turnCursorX - float32(game.tileSize / 2) + xDir * pos * float32(game.tileSize)
        dstRect.Y = game.turnCursorY - float32(game.tileSize / 2) + yDir * pos * float32(game.tileSize)
        rl.DrawTexturePro(textures.tilesSmall, tileRect, dstRect, origin, 0.0, rl.White)
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

    if textures.letterGlyphsSmall != nil {
        rl.UnloadFontData(textures.letterGlyphsSmall)
    }
    if textures.letterGlyphsLarge != nil {
        rl.UnloadFontData(textures.letterGlyphsLarge)
    }
    if textures.numberGlyphsSmall != nil {
        rl.UnloadFontData(textures.numberGlyphsSmall)
    }
    if textures.numberGlyphsLarge != nil {
        rl.UnloadFontData(textures.numberGlyphsLarge)
    }

	letterCodePoints := make([]int32, 26)
	numberCodePoints := make([]int32, 10)
	for i := 0; i < 26; i++ {
		letterCodePoints[i] = int32(0x41 + i)
	}
	for i := 0; i < 10; i++ {
		numberCodePoints[i] = int32(0x30 + i)
	}

	textures.letterGlyphsSmall = rl.LoadFontData(
		textures.fontData,
		int32(float64(tileSize) * 0.9),
		letterCodePoints,
		rl.FontDefault,
	)
	textures.letterGlyphsLarge = rl.LoadFontData(
		textures.fontData,
		int32(float64(tileSize) * 0.9 * 1.4),
		letterCodePoints,
		rl.FontDefault,
	)
	textures.numberGlyphsSmall = rl.LoadFontData(
		textures.fontData,
		int32(float64(tileSize) * 0.33),
		numberCodePoints,
		rl.FontDefault,
	)
	textures.numberGlyphsLarge = rl.LoadFontData(
		textures.fontData,
		int32(float64(tileSize) * 0.33 * 1.4),
		numberCodePoints,
		rl.FontDefault,
	)

    for i := 0; i < 26; i++ {
        img := &textures.letterGlyphsSmall[i].Image
        rl.ImageFormat(img, rl.UncompressedR8g8b8a8)
        setAlphaToBrightness(img.Data, img.Width, img.Height)

        img = &textures.letterGlyphsLarge[i].Image
        rl.ImageFormat(img, rl.UncompressedR8g8b8a8)
        setAlphaToBrightness(img.Data, img.Width, img.Height)
    }
    for i := 0; i < 10; i++ {
        img := &textures.numberGlyphsSmall[i].Image
        rl.ImageFormat(img, rl.UncompressedR8g8b8a8)
        setAlphaToBrightness(img.Data, img.Width, img.Height)

        img = &textures.numberGlyphsLarge[i].Image
        rl.ImageFormat(img, rl.UncompressedR8g8b8a8)
        setAlphaToBrightness(img.Data, img.Width, img.Height)
    }

    smallTileSize := min(tileSize - 1, int32(float64(tileSize) * 0.95))
    largeTileSize := int32(1.4 * float64(smallTileSize))
    tilesImageSmall := rl.GenImageColor(int(9 * smallTileSize), int(3 * smallTileSize), rl.White)
    tilesImageLarge := rl.GenImageColor(int(9 * largeTileSize), int(3 * largeTileSize), rl.White)

    textures.smallTileSize = int(smallTileSize)
    textures.largeTileSize = int(largeTileSize)
    textures.tileHlBorderSize = int(1 + (smallTileSize / 5))
    hlSize := textures.tileHlBorderSize * 2 + textures.smallTileSize

    tileHlImage := rl.GenImageColor(hlSize, hlSize, rl.White)
    makeTileHighlight(tileHlImage.Data, int32(hlSize), int32(hlSize), smallTileSize, smallTileSize, 0xfff0a0ff)
    if textures.tileHl.ID > 0 {
        rl.UnloadTexture(textures.tileHl)
    }
    textures.tileHl = rl.LoadTextureFromImage(tileHlImage)
    rl.UnloadImage(tileHlImage)

    tileCursorImage := rl.GenImageColor(textures.smallTileSize, textures.smallTileSize, rl.White)
    makeTileCursor(tileCursorImage.Data, smallTileSize, smallTileSize, 0xc060ffff)
    if textures.tileCursor.ID > 0 {
        rl.UnloadTexture(textures.tileCursor)
    }
    textures.tileCursor = rl.LoadTextureFromImage(tileCursorImage)
    rl.UnloadImage(tileCursorImage)

    scores := getLetterScores()
    for i := int32(0); i < 26; i++ {
        renderTile(tilesImageSmall, textures.letterGlyphsSmall, textures.numberGlyphsSmall, i, int32(scores[i]), smallTileSize)
        renderTile(tilesImageLarge, textures.letterGlyphsLarge, textures.numberGlyphsLarge, i, int32(scores[i]), largeTileSize)
    }

    tdsIntSmall := int(smallTileSize)
    tdsIntLarge := int(largeTileSize)
    roundTileEdges(tilesImageSmall.Data, 9, 3, tdsIntSmall, tdsIntSmall, int(float64(tdsIntSmall) * 0.15))
    roundTileEdges(tilesImageLarge.Data, 9, 3, tdsIntLarge, tdsIntLarge, int(float64(tdsIntLarge) * 0.15))

    if textures.tilesSmall.ID > 0 {
        rl.UnloadTexture(textures.tilesSmall)
    }
    textures.tilesSmall = rl.LoadTextureFromImage(tilesImageSmall)
    rl.UnloadImage(tilesImageSmall)

    if textures.tilesLarge.ID > 0 {
        rl.UnloadTexture(textures.tilesLarge)
    }
    textures.tilesLarge = rl.LoadTextureFromImage(tilesImageLarge)
    rl.UnloadImage(tilesImageLarge)

    return tileSize
}

func renderTile(tilesImage *rl.Image, letterGlyphs, numberGlyphs []rl.GlyphInfo, idx, points, tileSize int32) {
    srcRect := rl.Rectangle{}
    dstRect := rl.Rectangle{}
    letter := &letterGlyphs[idx].Image
    number := &numberGlyphs[points % 10].Image
    lift := -float32(letter.Height) * 0.15

    dstRect.X = float32(((idx % 9) * tileSize) + (tileSize - letter.Width) / 2)
    dstRect.Y = lift + float32(((idx / 9) * tileSize) + (tileSize - letter.Height) / 2)
    dstRect.Width  = float32(letter.Width)
    dstRect.Height = float32(letter.Height)
    srcRect.Width  = dstRect.Width
    srcRect.Height = dstRect.Height
    rl.ImageDraw(tilesImage, letter, srcRect, dstRect, rl.Black)

    corner := 2 * tileSize / 3
    dstRect.X = float32(((idx % 9) * tileSize) + corner + (tileSize / 3 - number.Width) / 2)
    dstRect.Y = float32(((idx / 9) * tileSize) + corner + (tileSize / 3 - number.Height) / 2)
    dstRect.Width  = float32(number.Width)
    dstRect.Height = float32(number.Height)
    srcRect.Width  = dstRect.Width
    srcRect.Height = dstRect.Height
    rl.ImageDraw(tilesImage, number, srcRect, dstRect, rl.Black)

    if points >= 10 {
        dstRect.X -= float32(number.Height)

        number = &numberGlyphs[(points / 10) % 10].Image
        dstRect.Width  = float32(number.Width)
        dstRect.Height = float32(number.Height)
        srcRect.Width  = dstRect.Width
        srcRect.Height = dstRect.Height
        rl.ImageDraw(tilesImage, number, srcRect, dstRect, rl.Black)
    }
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

    vel := rl.GetMouseDelta()
    inputs.cursorVelX = vel.X
    inputs.cursorVelY = vel.Y

    for i := 0; i < len(inputs.mouseButtons); i++ {
        flags := int32(0)
        // pointless. wrapping a single int in a structure just so that int can be taken out again and passed down in isolation.
        // raylib just takes the int directly, but the Go binding just has to be "clever" and make people do work that it will undo anyway.
        button := rl.MouseButton(i)
        if rl.IsMouseButtonPressed(button) {
            flags |= 1
        }
        if rl.IsMouseButtonDown(button) {
            flags |= 2
        }
        if rl.IsMouseButtonReleased(button) {
            flags |= 4
        }
        inputs.mouseButtons[i] = flags
    }

    if rl.IsKeyDown(KEY_UP) {
        inputs.arrowTimers[ARROW_UP]++
    } else {
        inputs.arrowTimers[ARROW_UP] = 0
    }
    if rl.IsKeyDown(KEY_DOWN) {
        inputs.arrowTimers[ARROW_DOWN]++
    } else {
        inputs.arrowTimers[ARROW_DOWN] = 0
    }
    if rl.IsKeyDown(KEY_LEFT) {
        inputs.arrowTimers[ARROW_LEFT]++
    } else {
        inputs.arrowTimers[ARROW_LEFT] = 0
    }
    if rl.IsKeyDown(KEY_RIGHT) {
        inputs.arrowTimers[ARROW_RIGHT]++
    } else {
        inputs.arrowTimers[ARROW_RIGHT] = 0
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

    rl.SetExitKey(0)
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
