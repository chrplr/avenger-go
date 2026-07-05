package main

import (
	"math"
	"math/rand"
)

type PlayerTimer int

const (
	TimerHurt PlayerTimer = iota
	TimerFire
	TimerAnim
	TimerExplode
)

type Player struct {
	Sprite
	Controls Controls

	VelocityX float64
	VelocityY float64

	Lives           int
	Shields         int
	ExtraLifeTokens int
	FacingX         float64
	TiltY           float64

	Timers [4]int
	Frame  int

	CarriedHuman *Human
	ThrustSprite Sprite
}

func NewPlayer(controls Controls) *Player {
	p := &Player{
		Sprite:       NewSprite("ship0", Width/2, LevelHeight/2),
		Controls:     controls,
		Lives:        5,
		Shields:      5,
		FacingX:      1,
		ThrustSprite: NewSprite("blank", 0, 0),
	}
	return p
}

func (p *Player) HitTest(x, y float64, audio *Audio) bool {
	if p.Lives == 0 || p.Timers[TimerExplode] > 0 {
		return false
	}
	// Player hit box: 40px wide, 15px high from center
	if math.Abs(x-p.X) < 40 && math.Abs(y-p.Y) < 15 {
		p.Timers[TimerHurt] = 60
		p.Shields--
		audio.PlaySound("player_hit", 1, 1)

		if p.Shields <= 0 {
			p.Lives--
			audio.FadeThrust(false)
			audio.PlaySound("player_explode", 1, 1)
			p.Timers[TimerExplode] = 18 * 4 // EXPLODE_FRAMES

			if p.CarriedHuman != nil {
				// We need the assets to drop the human correctly, but we'll handle this in Game loop
				p.CarriedHuman.Carrier = nil
				p.CarriedHuman.Falling = true // Will re-evaluate in human.update
				p.CarriedHuman.YVelocity = 0
				p.CarriedHuman = nil
			}
		}
		return true
	}
	return false
}

func sign(x float64) float64 {
	if x < 0 {
		return -1
	}
	if x > 0 {
		return 1
	}
	return 0
}

func (p *Player) Update(game *Game) {
	for i := range p.Timers {
		p.Timers[i]--
	}

	p.X = WrapPosition(p.X, p.X, LevelWidth)

	if p.Timers[TimerExplode] > 0 {
		frame := (18*4 - p.Timers[TimerExplode]) / 4
		p.Image = "ship_explode" + itoa(frame)
		p.ThrustSprite.Image = "blank"
		if p.Timers[TimerExplode] == 1 && p.Lives > 0 {
			p.Respawn(game)
		}
	} else if p.Lives == 0 {
		p.Image = "blank"
		p.ThrustSprite.Image = "blank"
	} else {
		xInput := p.Controls.getX()
		yInput := p.Controls.getY()

		p.TiltY = yInput

		if xInput != 0 {
			p.FacingX = sign(xInput)
		}

		moveX, moveY := xInput, yInput

		if p.Frame%8 != 0 || sign(p.FacingX) != sign(moveX) {
			moveX = 0
		}

		// Apply drag and force
		p.VelocityX = p.VelocityX*0.98 + moveX*0.2
		p.VelocityY = p.VelocityY*0.9 + moveY*0.5

		p.X += p.VelocityX
		p.Y += p.VelocityY

		if p.Y < 0 {
			p.Y = 0
		}
		if p.Y > LevelHeight {
			p.Y = LevelHeight
		}

		// Check for picking up human
		if p.CarriedHuman == nil {
			for _, h := range game.humans {
				if h.CanBePickedUpByPlayer() {
					dist := math.Hypot(h.X-p.X, h.Y-p.Y)
					if dist < 40 {
						h.PickedUpByPlayer(p)
						p.CarriedHuman = h
						break
					}
				}
			}
		} else {
			p.CarriedHuman.X = p.X
			p.CarriedHuman.Y = p.Y + 50
			if p.CarriedHuman.TerrainCheck(game.assets) {
				p.CarriedHuman.Dropped(game.assets)
				p.CarriedHuman = nil
				game.audio.PlaySound("rescue_prisoner", 1, 1)
			}
		}

		// Animation and Firing
		target := 0
		if p.FacingX < 0 {
			target = 8
		}

		if p.Frame == target {
			if p.Controls.buttonDown() && p.Timers[TimerFire] <= 0 {
				p.Timers[TimerFire] = 10
				laserVelX := p.VelocityX + 20*p.FacingX
				laserX := p.X + 40*p.FacingX
				laserY := p.Y + float64(p.getLaserFireYOffset())
				game.lasers = append(game.lasers, NewLaser(laserX, laserY, laserVelX, game.audio))
			}
		} else {
			if p.Timers[TimerAnim] <= 0 {
				p.Timers[TimerAnim] = 3
				p.Frame = (p.Frame + 1) % 16
			}
		}

		// Thrust Sound
		if moveX != 0 && p.Frame == target {
			game.audio.FadeThrust(true)
		} else {
			game.audio.FadeThrust(false)
		}

		animType := "ship"
		if p.Timers[TimerHurt] > 0 {
			animType = "hurt"
		}
		tilt := ""
		if p.Frame%8 == 0 && p.TiltY != 0 {
			if p.TiltY < 0 {
				tilt = "u"
			} else {
				tilt = "d"
			}
		}

		p.Image = animType + itoa(p.Frame) + tilt

		if p.Frame%8 != 0 || moveX == 0 {
			p.ThrustSprite.Image = "blank"
		} else {
			direction := 0
			if moveX < 0 {
				direction = 1
			}
			frame := (game.timer / 3) % 2
			p.ThrustSprite.Image = "boost_" + itoa(direction) + "_" + itoa(frame)
			xOffset := 66.0
			yOffset := -3.0
			p.ThrustSprite.X = p.X + xOffset*-moveX
			p.ThrustSprite.Y = p.Y + yOffset
		}
	}
}

func (p *Player) Respawn(game *Game) {
	p.Shields = 5

	bestScore := -1.0
	var bestPos [2]float64

	for i := 0; i < 20; i++ {
		rx := rand.Float64() * LevelWidth
		ry := 150.0 + rand.Float64()*150.0

		if len(game.enemies) == 0 {
			p.X = rx
			p.Y = ry
			break
		}

		minDist := float64(LevelWidth)
		for _, e := range game.enemies {
			dist := math.Abs(rx - e.X)
			if dist > LevelWidth/2 {
				dist = LevelWidth - dist
			}
			if dist < minDist {
				minDist = dist
			}
		}

		if minDist >= bestScore {
			bestScore = minDist
			bestPos[0] = rx
			bestPos[1] = ry
		}
	}
	if len(game.enemies) > 0 {
		p.X = bestPos[0]
		p.Y = bestPos[1]
	}
}

func (p *Player) getLaserFireYOffset() int {
	return []int{-1, 3, 2}[int(p.TiltY)+1]
}

func (p *Player) Draw(assets *Assets, offX, offY float64) {
	if p.TiltY == 1 {
		p.drawFlash(assets, offX, offY)
	}
	p.Sprite.Draw(assets, offX, offY)
	p.ThrustSprite.Draw(assets, offX, offY)
	if p.TiltY != 1 {
		p.drawFlash(assets, offX, offY)
	}
}

func (p *Player) drawFlash(assets *Assets, offX, offY float64) {
	if p.Frame%8 == 0 && p.Timers[TimerFire] > 5 {
		sprite := "flash" + itoa(p.Frame/8)
		x := p.X - offX - 25
		y := p.Y - offY - 13 + float64(p.getLaserFireYOffset())
		assets.Blit(sprite, x, y)
	}
}

func (p *Player) LevelEnded(shieldRestoreAmount int, humansSaved int) {
	p.Shields += shieldRestoreAmount
	if p.Shields > 5 {
		p.Shields = 5
	}

	if humansSaved == 10 {
		p.ExtraLifeTokens++
		if p.ExtraLifeTokens >= 3 {
			p.Lives++
			p.ExtraLifeTokens -= 3
		}
	}
}
