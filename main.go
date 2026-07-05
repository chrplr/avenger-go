package main

import (
	"flag"

	"github.com/chrplr/pgzgo"
)

type State int

const (
	StateTitle State = iota
	StatePlay
	StateGameOver
)

var (
	state      State
	stateTimer int
	game       *Game
	assets     *Assets
	audio      *Audio
)

func update() {
	controls := Controls{}

	stateTimer++

	switch state {
	case StateTitle:
		if controls.buttonPressed() {
			state = StatePlay
			stateTimer = 0
			game = NewGame(controls, assets, audio)
		}
	case StatePlay:
		if game.player.Lives <= 0 {
			state = StateGameOver
			stateTimer = 0
		} else {
			game.Update()
		}
	case StateGameOver:
		game.Update()
		if stateTimer > 60 && controls.buttonPressed() {
			state = StateTitle
			stateTimer = 0
			game = nil
			audio.PlayMusic("menu_theme")
		}
	}
}

func draw() {
	switch state {
	case StateTitle:
		assets.Blit("title", 0, 0)
		frame := (stateTimer / 4) % 14
		assets.Blit("start"+itoa(frame), float64(Width)/2-175, 450)
	case StatePlay:
		game.Draw()
	case StateGameOver:
		game.Draw()
		assets.DrawText("GAME OVER", float64(Width)/2, float64(Height)/2-100, true, "font")
	}
}

func main() {
	flag.Parse()

	a, err := pgzgo.New(pgzgo.Config{
		Title:  "Avenger",
		Width:  Width,
		Height: Height,
		Images: imagesFS,
		// Audio is nil: avenger keeps its own mixer for the thrust-fade behaviour.
	})
	if err != nil {
		panic(err)
	}
	defer a.Close()

	app = a
	assets = NewAssets(a.Screen)
	audio = NewAudio()
	defer audio.Destroy()

	state = StateTitle
	audio.PlayMusic("menu_theme")

	a.Loop(
		func(*pgzgo.App) { update() },
		func(*pgzgo.App) { draw() },
	)
}
