package main

import (
	"io"
	"os"
	"fmt"
	"strings"
	"image/color"
	rl "github.com/gen2brain/raylib-go/raylib"
)

type MainMenu struct {
	nPlayers int
	timeLimit int
}

type Game struct {
	menu MainMenu
	wndWidth int32
	wndHeight int32
	tileSize int32
}

const NORMAL = 0
const DOUBLE_WORD = 1
const TRIPLE_WORD = 2
const DOUBLE_LETTER = 3
const TRIPLE_LETTER = 4

var boardTileTypeLookup = [...]int32 {
    2, 0, 0, 3, 0, 0, 0, 2,
    0, 1, 0, 0, 0, 4, 0, 0,
    0, 0, 1, 0, 0, 0, 3, 0,
    3, 0, 0, 1, 0, 0, 0, 3,
    0, 0, 0, 0, 1, 0, 0, 0,
    0, 4, 0, 0, 0, 4, 0, 0,
    0, 0, 3, 0, 0, 0, 3, 0,
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

func getTileType(x, y int32) int32 {
    if x < 0 || y < 0 || x >= 15 || y >= 15 {
        return NORMAL
    }
    if x == 7 && y == 7 {
        return DOUBLE_WORD
    }

	ox := x
	oy := y

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

func drawMenu(game *Game, isActive bool) {
	
}

func drawFallingBoard(t float32) {
	
}

func drawGame(game *Game, boardTex rl.Texture2D) {
    boardLen := game.tileSize * 15
    var xOff int32 = int32(game.wndWidth - boardLen) / 2
    var yOff int32 = int32(game.wndHeight - boardLen) / 2
	rl.DrawTexture(boardTex, xOff, yOff, rl.White)
}

func maybeRecreateBoard(tex *rl.Texture2D, wndWidth, wndHeight, oldTileSize int32) (tileSize int32) {
	tileSize = int32(min(wndWidth / 16, wndHeight / 18))
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
    lineWidth := tileSize / 10
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

	rl.SetConfigFlags(rl.FlagWindowResizable)
	rl.InitWindow(800, 450, "scrambles")
	defer rl.CloseWindow()

	rl.SetTargetFPS(60)

	game := Game{}
	boardTex := rl.Texture2D{}

	//openingTimer := 0
	const maxOpeningTime = 240

	for !rl.WindowShouldClose() {
	    w := int32(rl.GetRenderWidth())
		h := int32(rl.GetRenderHeight())
		if game.wndWidth != w || game.wndHeight != h {
			game.wndWidth = w
			game.wndHeight = h
			game.tileSize = maybeRecreateBoard(&boardTex, w, h, game.tileSize)
		}

		rl.BeginDrawing()
		rl.ClearBackground(color.RGBA{0, 0x68, 0x30, 0xff})

        /*
		if openingTimer < maxOpeningTime {
			drawMenu(&game, openingTimer == 0)
			if openingTimer > 0 {
				drawFallingBoard(float32(openingTimer) / float32(maxOpeningTime))
			}
		} else {
			drawGame(&game)
		}
		*/
		drawGame(&game, boardTex)

		rl.EndDrawing()
	}
}
