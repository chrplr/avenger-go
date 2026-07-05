package main

// Sprite is a positioned image. By default, the anchor is the image centre;
// a sprite may instead pin an explicit pixel offset within the image to its position.
type Sprite struct {
	X, Y         float64
	Image        string
	AnchorCentre bool
	Ax, Ay       float64
}

func NewSprite(image string, x, y float64) Sprite {
	return Sprite{X: x, Y: y, Image: image, AnchorCentre: true}
}

// anchorOffset returns the pixel offset of the anchor from the image top-left.
func (s *Sprite) anchorOffset(a *Assets) (float64, float64) {
	if s.AnchorCentre {
		w, h := a.Size(s.Image)
		return w / 2, h / 2
	}
	return s.Ax, s.Ay
}

// Draw blits the sprite at its world position minus the camera offset (offX, offY).
func (s *Sprite) Draw(a *Assets, offX, offY float64) {
	ax, ay := s.anchorOffset(a)
	a.Blit(s.Image, s.X-offX-ax, s.Y-offY-ay)
}

// CollidePoint reports whether (px, py) lies within the sprite's current image rectangle.
func (s *Sprite) CollidePoint(a *Assets, px, py float64) bool {
	ax, ay := s.anchorOffset(a)
	w, h := a.Size(s.Image)
	left, top := s.X-ax, s.Y-ay
	return px >= left && px < left+w && py >= top && py < top+h
}

// WrapActor is a helper function to enforce wrapping around the level boundaries
// relative to the player's position.
func WrapPosition(x, playerX, levelWidth float64) float64 {
	for x-playerX < -levelWidth/2 {
		x += levelWidth
	}
	for x-playerX > levelWidth/2 {
		x -= levelWidth
	}
	return x
}
