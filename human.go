package main

import "math/rand"

// Human represents a collectible human character.
type Human struct {
	Sprite
	YVelocity    float64
	AnimTimer    int
	Waving       bool
	Dead         bool
	Exploding    bool
	Carrier      *Player     // Simplified: point to Player. In a full implementation, it might point to an interface.
	EnemyCarrier interface{} // We will use this in enemy.go
	Falling      bool
}

func NewHuman(x, y float64) *Human {
	return &Human{
		Sprite: NewSprite("human_stand0", x, y),
	}
}

func (h *Human) LaserHitTest(px, py float64, game *Game) bool {
	if !h.Exploding && h.CollidePoint(game.assets, px, py) {
		h.Die(game.audio)
		return true
	}
	return false
}

func (h *Human) Update(game *Game) {
	h.X = WrapPosition(h.X, game.player.X, LevelWidth)
	h.AnimTimer++

	if h.Exploding {
		frame := h.AnimTimer / 2
		if frame >= 10 {
			h.Dead = true
		} else {
			h.AnchorCentre = false
			h.Ax = 175
			h.Ay = 172
			h.Image = "human_explode" + itoa(frame)
		}
		return
	}

	if h.Carrier == nil && h.EnemyCarrier == nil {
		h.Falling = !h.TerrainCheck(game.assets)
		if !h.Falling && h.YVelocity > 3 {
			h.Die(game.audio)
		}

		if h.Falling {
			h.YVelocity += 0.05
			if h.YVelocity > 4 {
				h.YVelocity = 4
			}
			h.Y += h.YVelocity
		}
	}

	frame := h.AnimTimer / 7
	numFrames := 4
	spriteName := "stand"

	if h.Carrier != nil {
		spriteName = "saved"
		numFrames = 1
	} else if h.EnemyCarrier != nil {
		spriteName = "abducted"
	} else if h.Falling {
		spriteName = "fall"
		numFrames = 2
	} else if h.Waving {
		spriteName = "wave"
		numFrames = 3
		if h.AnimTimer > 100 {
			h.Waving = false
		}
	} else {
		spriteName = "stand"
		numFrames = 1
		if rand.Intn(200) == 0 {
			h.Waving = true
			h.AnimTimer = 0
		}
	}

	h.Image = "human_" + spriteName + itoa(forwardBackwardAnimationFrame(frame, numFrames))
}

func forwardBackwardAnimationFrame(frame, numFrames int) int {
	if numFrames < 2 {
		return 0
	}
	frame %= (numFrames*2 - 2)
	if frame >= numFrames {
		frame = (numFrames-1)*2 - frame
	}
	return frame
}

func (h *Human) CanBePickedUpByPlayer() bool {
	return h.Carrier == nil && h.EnemyCarrier == nil && h.Falling && !h.Dead
}

func (h *Human) CanBePickedUpByEnemy() bool {
	return h.Carrier == nil && h.EnemyCarrier == nil && !h.Falling && !h.Dead
}

func (h *Human) PickedUpByPlayer(p *Player) {
	h.Carrier = p
	h.Falling = false
}

func (h *Human) Dropped(assets *Assets) {
	h.Carrier = nil
	h.EnemyCarrier = nil
	h.Falling = !h.TerrainCheck(assets)
	h.YVelocity = 0
}

func (h *Human) TerrainCheck(assets *Assets) bool {
	// Convert world pos to pixel pos on terrain image
	x := int(h.X) % int(LevelWidth)
	if x < 0 {
		x += int(LevelWidth)
	}
	y := int(h.Y - TerrainOffsetY)
	return assets.CheckTerrain(x, y)
}

func (h *Human) Die(audio *Audio) {
	h.Exploding = true
	h.AnimTimer = 0
	audio.PlaySound("prisoner_die", 1, 1)
}
