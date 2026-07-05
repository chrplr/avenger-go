package main

// Bullet is fired by enemies.
type Bullet struct {
	Sprite
	VelX, VelY float64
	AnimTimer  int
}

func NewBullet(x, y, velX, velY float64, audio *Audio) *Bullet {
	// Sound volume calculation is simplified for now.
	audio.PlaySound("enemy_laser", 1, 0.5)
	return &Bullet{
		Sprite: NewSprite("bullet0", x, y),
		VelX:   velX,
		VelY:   velY,
	}
}

// Update returns true if the bullet should be destroyed.
func (b *Bullet) Update(game *Game) bool {
	b.X += b.VelX
	b.Y += b.VelY

	b.X = WrapPosition(b.X, game.player.X, LevelWidth)

	b.AnimTimer = game.timer
	frame := (b.AnimTimer / 4) % 2
	b.Image = "bullet" + itoa(frame)

	tooFar := b.X < game.player.X-Width || b.X > game.player.X+Width
	hitPlayer := game.player.HitTest(b.X, b.Y, game.audio)
	return tooFar || hitPlayer
}

// Laser is fired by the player.
type Laser struct {
	Sprite
	VelX      float64
	AnimTimer int
}

func NewLaser(x, y, velX float64, audio *Audio) *Laser {
	facingIdx := 0
	if velX <= 0 {
		facingIdx = 1
	}
	image := "laser_" + itoa(facingIdx) + "_0"
	audio.PlaySound("player_shoot", 1, 1)
	return &Laser{
		Sprite: NewSprite(image, x+velX, y),
		VelX:   velX,
	}
}

// Update returns true if the laser should be destroyed.
func (l *Laser) Update(game *Game) bool {
	l.X += l.VelX
	l.X = WrapPosition(l.X, game.player.X, LevelWidth)

	l.AnimTimer++
	facingIdx := 0
	if l.VelX <= 0 {
		facingIdx = 1
	}
	frame := l.AnimTimer / 8
	if frame > 1 {
		frame = 1
	}
	l.Image = "laser_" + itoa(facingIdx) + "_" + itoa(frame)

	tooFar := l.X < game.player.X-800 || l.X > game.player.X+800

	// Check enemy and human collisions
	hit := false
	for _, e := range game.enemies {
		if e.LaserHitTest(l.X, l.Y, game) {
			hit = true
			break
		}
	}
	if !hit {
		for _, h := range game.humans {
			if h.LaserHitTest(l.X, l.Y, game) {
				hit = true
				break
			}
		}
	}

	return tooFar || hit
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	if i == 1 {
		return "1"
	}
	if i == 2 {
		return "2"
	}
	if i == 3 {
		return "3"
	}
	if i == 4 {
		return "4"
	}
	if i == 5 {
		return "5"
	}
	if i == 6 {
		return "6"
	}
	if i == 7 {
		return "7"
	}
	if i == 8 {
		return "8"
	}
	if i == 9 {
		return "9"
	}
	// Fallback
	b := make([]byte, 0, 20)
	return string(appendInt(b, int64(i), 10))
}

func appendInt(b []byte, i int64, base int) []byte {
	if i < 0 {
		b = append(b, '-')
		i = -i
	}
	var a [64]byte
	n := len(a)
	for i >= int64(base) {
		n--
		a[n] = "0123456789abcdef"[i%int64(base)]
		i /= int64(base)
	}
	n--
	a[n] = "0123456789abcdef"[i]
	return append(b, a[n:]...)
}
