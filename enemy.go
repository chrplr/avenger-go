package main

import (
	"math"
	"math/rand"
)

type EnemyType int

const (
	TypeLander EnemyType = iota
	TypeMutant
	TypeBaiter
	TypePod
	TypeSwarmer
)

type EnemyState int

const (
	StateStart EnemyState = iota
	StateAlive
	StateExploding
	StateDead
)

type Enemy struct {
	Sprite
	Type EnemyType

	State      EnemyState
	StateTimer int
	AnimTimer  int

	MaxSpeed     float64
	Acceleration float64

	TargetPosX float64
	TargetPosY float64

	UpdateTargetTimer int
	VelocityX         float64
	VelocityY         float64

	TargetHuman *Human
	Carrying    bool

	BulletTimer int
	FireAngle   float64
}

func NewEnemy(startTimer int, t EnemyType, x, y float64, startVelX, startVelY float64) *Enemy {
	e := &Enemy{
		Sprite:     NewSprite("blank", x, y),
		Type:       t,
		StateTimer: startTimer,
		VelocityX:  startVelX,
		VelocityY:  startVelY,
	}

	if x == -1 && y == -1 {
		e.X = rand.Float64() * LevelWidth
		e.Y = 32.0 + rand.Float64()*(LevelHeight-64.0)
	}

	switch t {
	case TypeLander:
		e.MaxSpeed, e.Acceleration = 5.0, 0.1
	case TypeMutant:
		e.MaxSpeed, e.Acceleration = 9.0, 0.5
	case TypeBaiter:
		e.MaxSpeed, e.Acceleration = 9.0, 0.01
	case TypePod:
		e.MaxSpeed, e.Acceleration = 10.0, 0.03
	case TypeSwarmer:
		e.MaxSpeed, e.Acceleration = 8.0, 1.0
	}

	e.TargetPosX = e.X + (rand.Float64()*200.0 - 100.0)
	e.TargetPosY = e.Y + (rand.Float64()*200.0 - 100.0)

	if t == TypeSwarmer {
		e.State = StateAlive
	} else {
		e.State = StateStart
	}

	e.BulletTimer = rand.Intn(61) + 30
	e.AnimTimer = rand.Intn(48)

	return e
}

func (e *Enemy) LaserHitTest(px, py float64, game *Game) bool {
	if e.State == StateAlive && e.CollidePoint(game.assets, px, py) {
		e.State = StateExploding
		e.StateTimer = 0
		e.AnimTimer = 0

		if e.TargetHuman != nil {
			if e.Carrying {
				e.TargetHuman.Dropped(game.assets)
			}
			e.TargetHuman = nil
			e.Carrying = false
		}
		game.audio.PlaySound("enemy_explode", 6, 1)

		if e.Type == TypePod {
			for i := 0; i < 3; i++ {
				vx := rand.Float64()*50.0 - 25.0
				vy := rand.Float64()*50.0 - 25.0
				game.enemies = append(game.enemies, NewEnemy(0, TypeSwarmer, e.X, e.Y, vx, vy))
			}
		}
		return true
	}
	return false
}

func (e *Enemy) Update(game *Game) {
	e.X = WrapPosition(e.X, game.player.X, LevelWidth)
	e.TargetPosX = WrapPosition(e.TargetPosX, game.player.X, LevelWidth)

	if e.State == StateStart {
		e.StateTimer++
		if e.StateTimer == 1 {
			if e.Type == TypeMutant {
				game.audio.PlaySound("enemy_appear_mutant", 1, 1)
			} else if e.Type == TypeLander {
				game.audio.PlaySound("enemy_appear_normal", 1, 1)
			} else if e.Type == TypeBaiter {
				game.audio.PlaySound("enemy_appear_ufo", 1, 1)
			}
		}
		if e.StateTimer == 33 {
			e.State = StateAlive
		} else if e.StateTimer >= 0 {
			e.Image = "appear" + itoa(e.StateTimer/3)
		}
	} else if e.State == StateAlive {
		maxSpeed := e.MaxSpeed

		if e.TargetHuman != nil && e.TargetHuman.Dead {
			e.TargetHuman = nil
			e.Carrying = false
		}

		if e.TargetHuman == nil && e.Type == TypeLander && rand.Float64() < 0.001 {
			var available []*Human
			for _, h := range game.humans {
				if h.CanBePickedUpByEnemy() {
					targeted := false
					for _, otherE := range game.enemies {
						if otherE.TargetHuman == h {
							targeted = true
							break
						}
					}
					if !targeted {
						available = append(available, h)
					}
				}
			}
			if len(available) > 0 {
				bestDist := math.MaxFloat64
				var best *Human
				for _, h := range available {
					dx := h.X - e.X
					dy := h.Y - e.Y
					d2 := dx*dx + dy*dy
					if d2 < bestDist {
						bestDist = d2
						best = h
					}
				}
				e.TargetHuman = best
			}
		}

		if e.TargetHuman != nil {
			if e.Carrying {
				e.TargetPosX = e.X
				e.TargetPosY = 64
				maxSpeed = 0.5
				if math.Abs(e.Y-e.TargetPosY) < 10 {
					game.enemies = append(game.enemies, NewEnemy(0, TypeMutant, e.TargetHuman.X, e.TargetHuman.Y, 0, 0))
					e.TargetHuman.Die(game.audio)
					e.TargetHuman = nil
					e.Carrying = false
				}
			} else {
				xDist := math.Abs(e.X - e.TargetHuman.X)
				if xDist < 80 {
					maxSpeed = 1
				}
				if xDist > 100 {
					e.TargetPosX = e.TargetHuman.X
					e.TargetPosY = e.TargetHuman.Y - 200
				} else {
					e.TargetPosX = e.TargetHuman.X
					e.TargetPosY = e.TargetHuman.Y
					dist := math.Hypot(e.X-e.TargetPosX, e.Y-e.TargetPosY)
					if dist < 55 {
						e.Carrying = true
						e.TargetHuman.EnemyCarrier = e
						e.TargetHuman.Falling = false
					}
				}
			}
		} else {
			e.UpdateTargetTimer--
			if e.UpdateTargetTimer <= 0 {
				e.UpdateTargetTimer = 60
				maxPlayerDist := float64(LevelWidth)
				if e.Type == TypeLander {
					maxPlayerDist = 500.0
				}
				if math.Hypot(e.X-game.player.X, e.Y-game.player.Y) < maxPlayerDist {
					e.TargetPosX = game.player.X
					e.TargetPosY = game.player.Y
				}
				xRange, yRange := 100.0, 100.0
				if e.Type == TypeBaiter {
					xRange, yRange = 800.0, 300.0
				}
				e.TargetPosX += rand.Float64()*(xRange*2) - xRange
				e.TargetPosY += rand.Float64()*(yRange*2) - yRange
			}
		}

		dx := e.TargetPosX - e.X
		dy := e.TargetPosY - e.Y
		dist := math.Hypot(dx, dy)
		fx, fy := 0.0, 0.0
		if dist > 0 {
			fx = (dx / dist) * e.Acceleration
			fy = (dy / dist) * e.Acceleration
		}
		if e.Y < 64 {
			fy += 0.2
		}
		if e.Y > LevelHeight-64 {
			fy -= 0.2
		}

		e.VelocityX += fx
		e.VelocityY += fy

		vLen := math.Hypot(e.VelocityX, e.VelocityY)
		if vLen > maxSpeed {
			scale := math.Max(vLen*0.9, maxSpeed) / vLen
			e.VelocityX *= scale
			e.VelocityY *= scale
		}

		e.X += e.VelocityX
		e.Y += e.VelocityY

		if e.Carrying {
			e.TargetHuman.X = e.X
			e.TargetHuman.Y = e.Y + 50
		}

		e.BulletTimer--
		if e.BulletTimer <= 0 {
			if e.Type == TypeBaiter {
				vx := math.Cos(e.FireAngle) * 3
				vy := math.Sin(e.FireAngle) * 3
				game.bullets = append(game.bullets, NewBullet(e.X, e.Y, vx, vy, game.audio))
				e.BulletTimer = 8
				e.FireAngle += 0.3
			} else if game.player.Lives > 0 {
				pdx := game.player.X - e.X
				pdy := game.player.Y - e.Y
				pdist := math.Hypot(pdx, pdy)
				if pdist > 100 && pdist < 300 {
					pdx /= pdist
					pdy /= pdist
					vx := (pdx + rand.Float64() - 0.5) * 6
					vy := (pdy + rand.Float64() - 0.5) * 6
					game.bullets = append(game.bullets, NewBullet(e.X, e.Y, vx, vy, game.audio))
					upperLimit := 90
					if e.Type == TypeMutant {
						upperLimit = 30
					}
					e.BulletTimer = rand.Intn(upperLimit-20) + 20
				}
			}
		}

		// Animations
		if e.Type == TypeLander {
			frame := 0
			if e.TargetHuman != nil {
				if e.Carrying {
					frame = 2
				} else if math.Hypot(e.X-e.TargetHuman.X, e.Y-e.TargetHuman.Y) < 90 {
					frame = 1
				}
			}
			e.Image = "lander" + itoa(frame)
		} else if e.Type == TypeMutant {
			e.AnimTimer++
			e.Image = "mutant" + itoa((e.AnimTimer/6)%4)
		} else if e.Type == TypeBaiter {
			e.AnimTimer++
			e.Image = "baiter" + itoa((e.AnimTimer/3)%8)
		} else if e.Type == TypePod {
			e.AnimTimer++
			frame := forwardBackwardAnimationFrame(e.AnimTimer/6, 3)
			if e.VelocityX > 0 {
				frame += 3
			}
			e.Image = "pod" + itoa(frame)
		} else if e.Type == TypeSwarmer {
			e.AnimTimer++
			e.Image = "swarmer" + itoa((e.AnimTimer/6)%8)
		}

	} else if e.State == StateExploding {
		e.AnimTimer++
		frame := e.AnimTimer / 2
		imgFrame := frame
		if imgFrame > 9 {
			imgFrame = 9
		}
		e.Image = "enemy_explode" + itoa(imgFrame)
		if frame == 10 {
			e.State = StateDead
		}
	}
}
