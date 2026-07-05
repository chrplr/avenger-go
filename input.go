package main

import "github.com/Zyko0/go-sdl3/sdl"

// Keyboard/gamepad snapshotting lives in the pgzgo harness; keyDown,
// keyJustPressed and the pad* helpers are thin wrappers over it (see harness.go).

// Controls models the player's input axes and fire button
type Controls struct{}

// gamepadDeadZone matches the Python JoystickControls dead-zone.
const gamepadDeadZone = 0.6

// The axis getters return a digital -1/0/1, reading the keyboard or, failing
// that, the gamepad (d-pad first, then the left stick past the dead-zone).
func (c Controls) getX() float64 {
	if keyDown(sdl.SCANCODE_LEFT) {
		return -1
	} else if keyDown(sdl.SCANCODE_RIGHT) {
		return 1
	}
	if padLeft() {
		return -1
	} else if padRight() {
		return 1
	}
	if ax := padAxisX(); ax <= -gamepadDeadZone {
		return -1
	} else if ax >= gamepadDeadZone {
		return 1
	}
	return 0
}

func (c Controls) getY() float64 {
	if keyDown(sdl.SCANCODE_UP) {
		return -1
	} else if keyDown(sdl.SCANCODE_DOWN) {
		return 1
	}
	if padUp() {
		return -1
	} else if padDown() {
		return 1
	}
	if ay := padAxisY(); ay <= -gamepadDeadZone {
		return -1
	} else if ay >= gamepadDeadZone {
		return 1
	}
	return 0
}

// Fire is any keyboard fire key or either gamepad face button (A/B).
func (c Controls) buttonDown() bool {
	return keyDown(sdl.SCANCODE_SPACE) || keyDown(sdl.SCANCODE_RETURN) || keyDown(sdl.SCANCODE_KP_ENTER) ||
		keyDown(sdl.SCANCODE_Z) || keyDown(sdl.SCANCODE_X) || keyDown(sdl.SCANCODE_LCTRL) ||
		padButton0() || padButton1()
}

func (c Controls) buttonPressed() bool {
	return keyJustPressed(sdl.SCANCODE_SPACE) || keyJustPressed(sdl.SCANCODE_RETURN) || keyJustPressed(sdl.SCANCODE_KP_ENTER) ||
		keyJustPressed(sdl.SCANCODE_Z) || keyJustPressed(sdl.SCANCODE_X) || keyJustPressed(sdl.SCANCODE_LCTRL) ||
		padButton0Pressed() || padButton1Pressed()
}
