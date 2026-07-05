package main

import (
	"bytes"
	"fmt"
	"image"
	"image/png"

	"github.com/chrplr/pgzgo"
)

// Assets embeds the pgzgo Screen (texture cache, Blit, Size, Destroy) and adds
// the two game-specific pieces: the sprite-font text helper and the terrain
// collision mask decoded from the embedded terrain.png.
type Assets struct {
	*pgzgo.Screen
	terrain image.Image
}

// NewAssets wraps the harness screen and decodes the terrain mask for
// pixel-perfect collision from the embedded PNG.
func NewAssets(screen *pgzgo.Screen) *Assets {
	a := &Assets{Screen: screen}
	if data, err := imagesFS.ReadFile("images/terrain.png"); err == nil {
		if terrain, err := png.Decode(bytes.NewReader(data)); err == nil {
			a.terrain = terrain
		}
	}
	return a
}

// avengerFont builds a sprite font whose glyphs are named "<font>0<codepoint>"
// with a 22px space, matching the original text rendering.
func avengerFont(font string) pgzgo.Font {
	return pgzgo.Font{
		Space: 22,
		Name:  func(r rune) string { return fmt.Sprintf("%s0%d", font, r) },
	}
}

// DrawText draws a string using a sprite font, optionally centred on x.
func (a *Assets) DrawText(text string, x, y float64, centre bool, font string) {
	align := pgzgo.AlignLeft
	if centre {
		align = pgzgo.AlignCentre
	}
	a.Screen.DrawText(text, x, y, align, avengerFont(font))
}

// CheckTerrain returns true if the pixel at (x, y) on the terrain image is opaque.
func (a *Assets) CheckTerrain(x, y int) bool {
	if a.terrain == nil {
		return false
	}
	bounds := a.terrain.Bounds()
	if x >= bounds.Min.X && x < bounds.Max.X && y >= bounds.Min.Y && y < bounds.Max.Y {
		_, _, _, alpha := a.terrain.At(x, y).RGBA()
		return alpha > 0
	}
	// If below the terrain, treat as opaque so things don't fall off the world.
	return y >= bounds.Max.Y
}
