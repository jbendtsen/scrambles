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
}

func drawMenu(menu *MainMenu, isActive bool) {
	
}

func drawFallingBoard(t float32) {
	
}

func drawGame(game *Game) {
	
}

func createBoard(tex *rl.Texture2D, canvas *rl.Image, wndWidth, wndHeight, oldTileSize int) (tileSize int) {
	tileSize = wndWidth / 17
	canvas = rl.CreateImage

	rl.UnloadImage(canvas)

	if tex.id != 0 {
		rl.UnloadTexture(tex)
	}
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

	rl.InitWindow(800, 450, "scrambles")
	defer rl.CloseWindow()

	wndWidth := 0
	wndHeight := 0

	rl.SetTargetFPS(60)

	game := Game{}
	boardTex := rl.Texture2D{}
	boardImage := rl.Image{}

	openingTimer := 0
	const maxOpeningTime = 240

	for !rl.WindowShouldClose() {
		rl.BeginDrawing()
		rl.ClearBackground(color.RGBA{0, 0x68, 0x3a, 0xff})

		w := rl.GetRenderWidth()
		h := rl.GetRenderHeight()
		if wndWidth != w || wndHeight != h {
			wndWidth = w
			wndHeight = h
			createBoard(&boardTex, &boardImage, w, h)
		}

		if openingTimer < maxOpeningTime {
			drawMenu(&game.menu, openingTimer == 0)
			if openingTimer > 0 {
				drawFallingBoard(float32(openingTimer) / float32(maxOpeningTime))
			}
		} else {
			drawGame(&game)
		}

		rl.EndDrawing()
	}
}