# Avenger — Python vs. Go implementation comparison

This document compares the Go port in this folder with the original
`avenger.py`. Avenger is a *Defender*-style side-scroller: a wrapping horizontal
game world, a player ship that rescues falling humans, five enemy types, and a
radar. The Go port is a faithful translation of the game logic; the differences
are almost all mechanical consequences of Go's static typing, the absence of
class inheritance/operator overloading, and the swap from Pygame Zero to
[go-sdl3](https://github.com/Zyko0/go-sdl3).

---

## 1. File organisation

Python is a single 1,446-line module. The Go port splits it by entity:

| Python section | Go file |
|---|---|
| `update()`, `draw()`, state machine, mixer setup | `main.go` |
| `Game` class (waves, update loop, scrolling draw, UI, radar) | `game.go` |
| `Player`, `Radar` | `player.go` |
| `Enemy`, `EnemyState`/`EnemyType` enums | `enemy.go` |
| `Human` | `human.go` |
| `Bullet`, `Laser` | `projectiles.go` |
| `WrapActor`, `Actor` behaviour | `sprite.go` |
| image cache, text, terrain mask | `assets.go` |
| `play_sound`, thrust sound, music | `audio.go` |
| `Controls`, `KeyboardControls` | `input.go` |

---

## 2. Inheritance → struct embedding

Python builds a small class tree on top of Pygame Zero's `Actor`:

```
Actor → WrapActor → { Bullet, Laser, Player, Enemy, Human }
```

`WrapActor` adds "wrap around the edges of the world relative to the player" and
an offset-aware `draw`. Go has no inheritance, so the port uses a `Sprite`
struct (embedded into each entity) plus a free function for the wrap maths:

```go
type Sprite struct {
    X, Y         float64
    Image        string
    AnchorCentre bool
    Ax, Ay       float64
}

func WrapPosition(x, playerX, levelWidth float64) float64 {
    for x-playerX < -levelWidth/2 { x += levelWidth }
    for x-playerX > levelWidth/2  { x -= levelWidth }
    return x
}
```

Each entity (`Player`, `Enemy`, `Human`, `Bullet`, `Laser`) embeds `Sprite` and
calls `WrapPosition` at the top of its own `Update`, exactly where Python called
`super().update()`.

`Player`, `Enemy`, etc. do **not** share a common interface in the Go port,
because — unlike the cars in Leading Edge — they are stored in **separate typed
slices** (`enemies []*Enemy`, `humans []*Human`, `lasers []*Laser`,
`bullets []*Bullet`), so no polymorphic container is needed.

---

## 3. `pygame.Vector2` → scalar field pairs

Avenger's Python uses `Vector2` for position and velocity. Rather than introduce
a vector type, the Go port **decomposes each vector into two `float64` fields**,
which reads naturally for this game's simple axis-aligned motion:

| Python | Go |
|---|---|
| `self.velocity = Vector2(0,0)` | `VelocityX, VelocityY float64` |
| `self.target_pos = Vector2(...)` | `TargetPosX, TargetPosY float64` |
| `(a - b).length()` | `math.Hypot(ax-bx, ay-by)` |
| `vec.normalize()` then scale | manual `dx/dist, dy/dist` |
| `self.velocity.scale_to_length(n)` | `scale := n/vLen; vx*=scale; vy*=scale` |

For example the enemy's "steer toward target" force:

```python
vec = (self.target_pos - self.pos).normalize()
force = vec * self.acceleration
```
```go
dist := math.Hypot(dx, dy)
if dist > 0 { fx = (dx/dist) * e.Acceleration; fy = (dy/dist) * e.Acceleration }
```

---

## 4. Enums → `const … iota`

Python uses `Enum`/`IntEnum`:

```python
class EnemyState(Enum): START=0; ALIVE=1; EXPLODING=2; DEAD=3
class EnemyType(Enum):  LANDER=0; MUTANT=1; BAITER=2; POD=3; SWARMER=4
class Player.Timer(IntEnum): HURT=0; FIRE=1; ANIM=2; EXPLODE=3
```

Go uses typed constant blocks with `iota`:

```go
type EnemyState int
const ( StateStart EnemyState = iota; StateAlive; StateExploding; StateDead )

type EnemyType int
const ( TypeLander EnemyType = iota; TypeMutant; TypeBaiter; TypePod; TypeSwarmer )

type PlayerTimer int
const ( TimerHurt PlayerTimer = iota; TimerFire; TimerAnim; TimerExplode )
```

The player's four timers, a Python `list` indexed by the `Timer` enum, become a
fixed-size Go array `Timers [4]int` indexed by the same constants.

---

## 5. `None` → pointers and booleans

Avenger has several optional references. The port maps them to nil pointers:

| Python | Go |
|---|---|
| `self.carried_human = None / Human` | `CarriedHuman *Human` |
| `self.target_human = None / Human` | `TargetHuman *Human` |
| `human.carrier = None / Player / Enemy` | `Carrier *Player` **and** `EnemyCarrier interface{}` |

The `carrier` field is interesting: in Python one attribute holds *either* the
player or an enemy, distinguished later with `if self.carrier == game.player`.
Go is statically typed, so the port splits it into two fields — a typed
`Carrier *Player` and a separate `EnemyCarrier` — and the "who holds me" checks
become nil checks on the appropriate field.

---

## 6. Dynamic sprite names → string building

Both versions compose sprite names as strings; Python then resolves them with
`getattr(images, name)`, while Go uses them as keys into a texture cache.

```python
self.image = "mutant" + str((self.anim_timer // 6) % 4)
self.image = f"laser_{facing_idx}_{min(1, self.anim_timer // 8)}"
```
```go
e.Image = "mutant" + itoa((e.AnimTimer/6)%4)
l.Image = "laser_" + itoa(facingIdx) + "_" + itoa(frame)
```

Notably, the Go port defines its **own `itoa` helper** (in `projectiles.go`)
rather than importing `strconv`, with a fast path for single digits — a small
stylistic quirk of this particular port.

---

## 7. List comprehensions → explicit slice rebuilds

Python removes expired/dead objects with list comprehensions whose predicate has
side effects (the `update()` returns whether to destroy):

```python
self.lasers  = [l for l in self.lasers  if not l.update()]
self.bullets = [b for b in self.bullets if not b.update()]
self.humans  = [h for h in self.humans  if not h.dead]
self.enemies = [e for e in self.enemies if e.state != EnemyState.DEAD]
```

Go rebuilds the slices explicitly:

```go
var newLasers []*Laser
for _, l := range g.lasers {
    if !l.Update(g) { newLasers = append(newLasers, l) }
}
g.lasers = newLasers
```

The laser/enemy collision uses a Python list comprehension that evaluates
`laser_hit_test` for *every* enemy and human (killing each overlapping one),
then `sum(collisions) > 0`. The Go port breaks on the first hit per category — a
minor behavioural simplification that only matters if a single laser point
overlaps two enemies in one frame.

---

## 8. The camera-offset pitfall (a fixed porting bug)

The scrolling draw in Python computes `offset_x = -(player.x - camera_offset)`
and each `WrapActor.draw` adds it: screen `x = world_x + offset_x - anchor`. The
Go `Sprite.Draw` *subtracts* its offset (`X - offX - anchor`), so the Go offset
must be `player.X - cameraOffsetX` (no leading negation). The original port had
the sign inverted, which placed the ship at `2*playerX - camera` — fine at the
start position but off-screen after the first respawn at a random X. This was
found and corrected during review; it is the class of bug most likely to appear
when translating pygame's "add offset to position, then blit at anchor"
convention into a "blit at position minus camera" convention.

---

## 9. Framework specifics

| Concern | Python | Go (go-sdl3) |
|---|---|---|
| Terrain collision | `pygame.mask.from_surface` + `get_at` | decode `terrain.png` to `image.Image`, test alpha in `CheckTerrain` |
| Radar clipping | `screen.surface.set_clip(rect)` | `renderer.SetClipRect(&rect)` |
| Blit anchor | `Actor` centre/anchor | `Sprite.anchorOffset` (centre or explicit `Ax,Ay`) |
| Text | `getattr(images, font+"0"+ord)` | same technique in `Assets.DrawText` |
| Explosion anchor swap | reset `self.anchor` mid-animation | set `AnchorCentre=false; Ax,Ay=…` |

The terrain mask is the standout: Python asks Pygame for a bitmask; the Go port
loads the PNG with the standard library `image/png` decoder and checks the alpha
channel of a pixel directly.

---

## 10. Audio

Both play random one-shot variants and a looping thrust sound that fades in/out.
Python calls methods on `Sound` objects and even creates a fresh `Sound`
instance to play an effect at a custom volume (for distance-attenuated enemy
lasers). The go-sdl3 port:

- preloads every `.ogg` into a `map[string]*mixer.Audio`;
- plays one-shots with `mixer.PlayAudio`;
- models the looping thrust as a persistent `*mixer.Track` toggled by
  `FadeThrust(active bool)`.

**Intentional simplification:** the distance-based per-shot volume of
`enemy_laser` is approximated with a constant, because the current mixer wrapper
doesn't expose per-play gain. This affects only effect loudness, not gameplay.

---

## 11. Input

Python's `Controls` is an abstract base with keyboard and joystick subclasses
that track button edges in an `update()` method. The Go port keeps a per-frame
keyboard snapshot (`keys`/`prevKeys`) and derives held vs. just-pressed from it,
implementing keyboard only. It also accepts a few extra fire keys (space, Z, X,
Ctrl) for convenience. Held state (fire) and the two movement axes behave
identically to the original.

---

## 12. Numeric semantics matched exactly

- **Floor vs. truncation for wrapping/radar.** Python's `%` is always
  non-negative; Go's `%` can be negative. The radar blip mapping and terrain
  lookup use an explicit "add `LevelWidth` if negative" to reproduce Python's
  modulo (this was also part of the review fixes).
- **Integer division** in animation frames (`anim_timer // 6`) → Go `/` on ints.
- **`math.Floor`** for the camera offset (`math.floor(... )`) → `math.Floor`.
- **`min`/`max`/`abs`** → `math.Min`/`math.Max`/`math.Abs` (or the builtins).

---

## 13. What is intentionally identical

- Wave generation (`new_wave`): lander/pod/baiter/mutant/swarmer counts per wave,
  the every-5th/10th-wave special waves, and human spawn positions.
- Enemy AI per type: target selection, human abduction → mutant conversion,
  baiter firing pattern, pod → swarmer split on death.
- Player physics: drag/force per axis, facing-flip animation gating of thrust and
  firing, shield/life/extra-life-token logic, respawn position scoring.
- Human behaviour: falling, being carried by player vs. enemy, terrain landing,
  explode/die.
- Scrolling camera, parallax background, radar, and all UI/score/wave-end text.

---

## 14. Summary of differences

| Category | Difference | Reason |
|---|---|---|
| Inheritance | `Actor`/`WrapActor` tree → embedded `Sprite` + `WrapPosition` | no classes |
| Vectors | `Vector2` → scalar `X/Y`, `VelocityX/Y`, etc. | no operator overloading |
| Enums | `Enum`/`IntEnum` → `const … iota`, `[4]int` timers | static typing |
| Optionals | `None` → nil pointers; split `carrier` into two fields | no `None`, static types |
| Containers | polymorphic `list` + comprehensions → typed slices + loops | static typing |
| Terrain | `pygame.mask` → `image/png` alpha test | library difference |
| Audio | `Sound` methods → `mixer.Track`; constant laser volume | go-sdl3 model / simplification |
| Input | keyboard+joystick classes → keyboard snapshot | scope |
| Fixed bug | camera offset sign; negative modulo on radar | pygame→SDL draw convention |

The control flow, constants, wave design, and AI are line-by-line equivalent to
`avenger.py`; the differences are the standard consequences of moving from a
dynamically typed, inheritance-based Pygame Zero program to statically typed Go
on go-sdl3.
