package main

import (
	"fmt"
	"math"
)

const (
	Width                      = 960
	Height                     = 540
	LevelWidth                 = 4096
	LevelHeight                = 640
	TerrainOffsetY             = 160
	WaveCompleteScreenDuration = 320
)

var HumanStartPos = [][2]float64{
	{204, 410}, {489, 209}, {865, 374}, {1262, 405}, {1937, 263},
	{2193, 278}, {2601, 405}, {2846, 347}, {3317, 193}, {3646, 233},
}

type Game struct {
	player *Player

	enemies []*Enemy
	humans  []*Human
	lasers  []*Laser
	bullets []*Bullet

	score     int
	wave      int
	waveTimer int
	timer     int

	playerCameraOffsetX float64

	assets *Assets
	audio  *Audio
}

func NewGame(controls Controls, assets *Assets, audio *Audio) *Game {
	g := &Game{
		player:              NewPlayer(controls),
		playerCameraOffsetX: Width / 3.0,
		assets:              assets,
		audio:               audio,
	}
	g.newWave()
	audio.PlayMusic("ambience")
	return g
}

func (g *Game) newWave() {
	g.wave++
	numLanders := 4 + g.wave
	numPods := -1 + g.wave/2
	numBaiters := 0
	numMutants := 0
	numSwarmers := 0

	if g.wave%5 == 0 {
		numLanders = 0
		numPods = 0
		numBaiters = g.wave
		if g.wave%10 == 0 {
			numSwarmers = g.wave / 2
		} else {
			numMutants = g.wave / 2
		}
	}

	for i := 0; i < numLanders; i++ {
		g.enemies = append(g.enemies, NewEnemy(-i*20, TypeLander, -1, -1, 0, 0))
	}
	for i := 0; i < numPods; i++ {
		g.enemies = append(g.enemies, NewEnemy(-i*50, TypePod, -1, -1, 0, 0))
	}
	for i := 0; i < numBaiters; i++ {
		g.enemies = append(g.enemies, NewEnemy(-i*100, TypeBaiter, -1, -1, 0, 0))
	}
	for i := 0; i < numMutants; i++ {
		g.enemies = append(g.enemies, NewEnemy(-i*10, TypeMutant, -1, -1, 0, 0))
	}
	for i := 0; i < numSwarmers; i++ {
		g.enemies = append(g.enemies, NewEnemy(-i*10, TypeSwarmer, -1, -1, 0, 0))
	}

	g.humans = nil
	for _, pos := range HumanStartPos {
		g.humans = append(g.humans, NewHuman(pos[0], pos[1]+TerrainOffsetY))
	}
	g.audio.PlaySound("new_wave", 1, 1)
}

func (g *Game) Update() {
	g.waveTimer++
	if g.waveTimer == 0 {
		g.newWave()
	}
	g.timer++

	if g.waveTimer > 0 && g.waveTimer%(30*60) == 0 && g.player.Lives > 0 {
		g.enemies = append(g.enemies, NewEnemy(0, TypeBaiter, -1, -1, 0, 0))
	}

	g.player.Update(g)

	var newLasers []*Laser
	for _, l := range g.lasers {
		if !l.Update(g) {
			newLasers = append(newLasers, l)
		}
	}
	g.lasers = newLasers

	var newBullets []*Bullet
	for _, b := range g.bullets {
		if !b.Update(g) {
			newBullets = append(newBullets, b)
		}
	}
	g.bullets = newBullets

	for _, e := range g.enemies {
		e.Update(g)
	}
	for _, h := range g.humans {
		h.Update(g)
	}

	var newHumans []*Human
	for _, h := range g.humans {
		if !h.Dead {
			newHumans = append(newHumans, h)
		}
	}
	g.humans = newHumans

	prevEnemiesCount := len(g.enemies)
	var newEnemies []*Enemy
	for _, e := range g.enemies {
		if e.State != StateDead {
			newEnemies = append(newEnemies, e)
		}
	}
	g.enemies = newEnemies

	diff := prevEnemiesCount - len(g.enemies)
	if diff > 0 {
		g.score += 150 * diff
	}

	anyFalling := false
	for _, h := range g.humans {
		if h.Falling {
			anyFalling = true
			break
		}
	}

	if g.waveTimer > 0 && len(g.enemies) == 0 && !anyFalling && g.player.CarriedHuman == nil {
		g.waveTimer = -WaveCompleteScreenDuration
		g.player.LevelEnded(g.getShieldRestoreAmount(), g.getHumansSaved())
		g.audio.PlaySound("wave_complete", 1, 1)
	}
}

func (g *Game) getShieldRestoreAmount() int {
	res := g.getHumansSaved() / 2
	if res > 5 {
		res = 5
	}
	return res
}

func (g *Game) getHumansSaved() int {
	c := 0
	for _, h := range g.humans {
		if !h.Exploding {
			c++
		}
	}
	return c
}

func (g *Game) Draw() {
	targetCameraOffsetX := float64(Width) / 3.0
	if g.player.FacingX <= 0 {
		targetCameraOffsetX = 2.0 * Width / 3.0
	}
	targetCameraOffsetX -= g.player.VelocityX * 15.0

	delta := (targetCameraOffsetX - g.playerCameraOffsetX) / 20.0
	if delta > 8 {
		delta = 8
	} else if delta < -8 {
		delta = -8
	}
	g.playerCameraOffsetX = math.Floor(g.playerCameraOffsetX + delta)

	left := -float64(int(g.player.X-g.playerCameraOffsetX) % LevelWidth)
	if left > 0 {
		left -= LevelWidth // ensure negative offset
	}
	top := -float64(int(g.player.Y / 4.0))
	if top < -100 {
		top = -100
	}

	bgWidth, _ := g.assets.Size("background")
	for i := 0; i < 5; i++ {
		g.assets.Blit("background", left/2.0+bgWidth*float64(i), top/2.0)
	}

	// Terrain draw - we'll simulate the surface draw via 2 background blits
	g.assets.Blit("terrain", left, top+TerrainOffsetY)
	g.assets.Blit("terrain", left+LevelWidth, top+TerrainOffsetY)

	offsetX := g.player.X - g.playerCameraOffsetX

	for _, b := range g.bullets {
		b.Draw(g.assets, offsetX, -top)
	}
	for _, h := range g.humans {
		h.Draw(g.assets, offsetX, -top)
	}
	for _, e := range g.enemies {
		e.Draw(g.assets, offsetX, -top)
	}

	if g.player.TiltY == 1 {
		for _, l := range g.lasers {
			l.Draw(g.assets, offsetX, -top)
		}
		g.player.Draw(g.assets, offsetX, -top)
	} else {
		g.player.Draw(g.assets, offsetX, -top)
		for _, l := range g.lasers {
			l.Draw(g.assets, offsetX, -top)
		}
	}

	g.drawUI()
}

func radarPos(x, y float64) (float64, float64) {
	// Match Python's modulo, which is always non-negative, so blips stay on the radar
	// even when world coordinates go negative (the player's X is never wrapped).
	xMod := int(x) % LevelWidth
	if xMod < 0 {
		xMod += LevelWidth
	}
	rx := (Width / 2) - 176 + (float64(xMod) / 11.5)
	ry := 4 + (float64(int(y)) / 11.0)
	return rx, ry
}

func (g *Game) drawUI() {
	g.assets.Blit("radar", Width/2-176, 4) // Center anchored in PYGZ, top-left is w/2-w/2

	// Clip the radar dots to the radar panel.
	g.assets.SetClip(int32(Width/2-176), 4, 352, 64)

	for _, e := range g.enemies {
		if e.State == StateAlive {
			rx, ry := radarPos(e.X, e.Y)
			g.assets.Blit("dot-red", rx, ry)
		}
	}
	for _, h := range g.humans {
		rx, ry := radarPos(h.X, h.Y)
		g.assets.Blit("dot-green", rx, ry)
	}
	prx, pry := radarPos(g.player.X, g.player.Y)
	g.assets.Blit("dot-white", prx, pry)

	g.assets.ClearClip()

	for i := 0; i < g.player.Lives; i++ {
		g.assets.Blit("life", float64(20+20*i), 21)
	}
	for i := 0; i < g.player.Shields; i++ {
		g.assets.Blit("armor", float64(20+20*i), 52)
	}
	for i := 0; i < g.player.ExtraLifeTokens; i++ {
		frame := ((g.timer / 6) + i) % 8
		g.assets.Blit("token"+itoa(frame), float64(20+20*i), 83)
	}

	scoreText := itoa(g.score)
	g.assets.DrawText(scoreText, Width-20, 28, true, "font_status")

	if g.waveTimer < 0 {
		y := Height/2.0 - 140.0
		for _, line := range g.getWaveEndText() {
			g.assets.DrawText(line, float64(Width/2), y, true, "font")
			y += 65
		}
	}
}

func (g *Game) getWaveEndText() []string {
	humansSaved := g.getHumansSaved()
	i := (g.waveTimer + WaveCompleteScreenDuration) / (WaveCompleteScreenDuration / 4)
	lines := []string{fmt.Sprintf("WAVE %d COMPLETE", g.wave)}

	if i >= 1 {
		s := "S"
		if humansSaved == 1 {
			s = ""
		}
		lines = append(lines, fmt.Sprintf("%d HUMAN%s SAVED", humansSaved, s))
	}
	if i >= 2 {
		sh := g.getShieldRestoreAmount()
		s := "S"
		if sh == 1 {
			s = ""
		}
		lines = append(lines, fmt.Sprintf("%d SHIELD%s RESTORED", sh, s))
	}
	if i >= 3 && humansSaved == 10 {
		if g.player.ExtraLifeTokens == 0 {
			lines = append(lines, "EXTRA LIFE")
		} else {
			lines = append(lines, "LIFE TOKEN GAINED")
		}
	}
	return lines
}
