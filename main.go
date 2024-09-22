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

type Textures struct {
	board rl.Texture2D
	tiles rl.Texture2D
	letterGlyphs []rl.GlyphInfo
	numberGlyphs []rl.GlyphInfo
	fontData []byte
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

func drawMenu(game *Game, isActive bool) {
	
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

func drawGame(game *Game, textures *Textures) {
	
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
    tilesImage := rl.GenImageColor(int(9 * tileDisplaySize), int(3 * tileDisplaySize), color.RGBA{255, 224, 160, 255})

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

    {
        blankNumber := &textures.numberGlyphs[2].Image
        corner := 2 * tileDisplaySize / 3
        dstRect.X = float32((8 * tileDisplaySize) + corner + (tileDisplaySize / 3 - blankNumber.Width) / 2)
        dstRect.Y = float32((2 * tileDisplaySize) + corner + (tileDisplaySize / 3 - blankNumber.Height) / 2)
        dstRect.Width  = float32(blankNumber.Width)
        dstRect.Height = float32(blankNumber.Height)
        srcRect.Width  = dstRect.Width
        srcRect.Height = dstRect.Height
        rl.ImageDraw(tilesImage, blankNumber, srcRect, dstRect, rl.Black)
    }

    if textures.tiles.ID > 0 {
        rl.UnloadTexture(textures.tiles)
    }

    textures.tiles = rl.LoadTextureFromImage(tilesImage)
    rl.UnloadImage(tilesImage)

    return tileSize
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
	game := Game{}
	game.startupTimestamp = time.Now().UnixMilli()

	game.wordsList = loadWords()
	if game.wordsList == nil {
		return
	}

	game.wordMap = make(map[string]int)
	for i := 0; i < len(game.wordsList); i++ {
		game.wordMap[game.wordsList[i]] = i
	}

	ttfData := loadTtfData()
	if ttfData == nil {
		return
	}

	rl.SetConfigFlags(rl.FlagWindowResizable)
	rl.InitWindow(800, 450, "scrambles")
	defer rl.CloseWindow()

	rl.SetTargetFPS(60)

	textures := Textures{}
	textures.fontData = ttfData

    //gameStarted := false
	openingTimer := 0
	const maxOpeningTime = 180

	for !rl.WindowShouldClose() {
	    w := int32(rl.GetRenderWidth())
		h := int32(rl.GetRenderHeight())
		if game.wndWidth != w || game.wndHeight != h {
			game.wndWidth = w
			game.wndHeight = h
			game.tileSize = updateTextures(&textures, w, h, game.tileSize)
		}

		rl.BeginDrawing()
		rl.ClearBackground(color.RGBA{0, 0x68, 0x30, 0xff})

        /*
        if rl.IsMouseButtonPressed(0) {
            gameStarted = true
        }
        */

	    tBoardFall := float32(1.0)
	    if openingTimer < maxOpeningTime {
		    drawMenu(&game, openingTimer == 0)
		    tBoardFall = float32(openingTimer) / float32(maxOpeningTime)
	    }

	    drawBoard(&game, textures.board, tBoardFall)

	    if openingTimer >= maxOpeningTime {
		    drawGame(&game, &textures)
		    openingTimer = maxOpeningTime
	    }

	    openingTimer += 1
        rl.DrawTexture(textures.tiles, 0, 0, rl.White)

		rl.EndDrawing()
		game.frameCounter += 1
	}
}
