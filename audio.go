package main

import (
	"embed"
	"math/rand"
	"path"
	"runtime"
	"strconv"

	"github.com/Zyko0/go-sdl3/mixer"
	"github.com/Zyko0/go-sdl3/sdl"
)

// audioFS embeds the sound effects and music into the binary.
//
//go:embed sounds music
var audioFS embed.FS

// Audio wraps SDL3_mixer. All operations are best-effort.
type Audio struct {
	mixer  *mixer.Mixer
	sounds map[string]*mixer.Audio

	menuMusic     *mixer.Track
	ambientMusic  *mixer.Track
	thrustTrack   *mixer.Track
	thrustPlaying bool
}

func NewAudio() *Audio {
	a := &Audio{sounds: make(map[string]*mixer.Audio)}

	// SDL3_mixer has no js/wasm bindings yet: in the browser, skip audio init
	// entirely and run silently (every play method no-ops on a nil mixer).
	if runtime.GOOS == "js" {
		return a
	}

	if err := mixer.Init(); err != nil {
		return a
	}
	// Request more channels for intense bullet hell situations.
	// Since go-sdl3's mixer device creation is simple, we rely on SDL's defaults which usually handle 8+ streams.
	m, err := mixer.CreateMixerDevice(sdl.AUDIO_DEVICE_DEFAULT_PLAYBACK, nil)
	if err != nil {
		return a
	}
	a.mixer = m

	// Preload every embedded .ogg sound effect.
	entries, _ := audioFS.ReadDir("sounds")
	for _, e := range entries {
		fname := e.Name()
		if path.Ext(fname) != ".ogg" {
			continue
		}
		if snd := loadAudioFromFS(m, "sounds/"+fname); snd != nil {
			a.sounds[fname[:len(fname)-len(".ogg")]] = snd
		}
	}

	a.menuMusic = a.loopingTrack(m, "music/menu_theme.ogg", 0.5)
	a.ambientMusic = a.loopingTrack(m, "music/ambience.ogg", 0.5)

	// Set up the thrust sound as a track so it can be smoothly faded and looped.
	if snd, ok := a.sounds["thrust0"]; ok {
		if track, err := m.CreateTrack(); err == nil {
			track.SetAudio(snd)
			track.SetLoops(-1)
			track.SetGain(0.3)
			a.thrustTrack = track
		}
	}

	return a
}

// loadAudioFromFS decodes an embedded audio file into an in-memory Audio via an
// SDL IOStream (predecoded, so no stream stays open afterwards).
func loadAudioFromFS(m *mixer.Mixer, p string) *mixer.Audio {
	data, err := audioFS.ReadFile(p)
	if err != nil {
		return nil
	}
	stream, err := sdl.IOFromConstMem(data)
	if err != nil {
		return nil
	}
	snd, err := m.LoadAudio_IO(stream, true, true) // predecode + closeio
	if err != nil {
		return nil
	}
	return snd
}

func (a *Audio) loopingTrack(m *mixer.Mixer, p string, gain float32) *mixer.Track {
	audio := loadAudioFromFS(m, p)
	if audio == nil {
		return nil
	}
	t, err := m.CreateTrack()
	if err != nil {
		return nil
	}
	t.SetAudio(audio)
	t.SetLoops(-1)
	t.SetGain(gain)
	return t
}

// PlaySound plays one of a family of sound variants based on count.
func (a *Audio) PlaySound(name string, count int, volume float32) {
	if a.mixer == nil || volume <= 0 {
		return
	}
	variant := name + "0"
	if count > 1 {
		variant = name + strconv.Itoa(rand.Intn(count))
	}
	if snd, ok := a.sounds[variant]; ok {
		a.mixer.PlayAudio(snd) // go-sdl3 mixer doesn't currently expose per-play gain overrides, so it plays at full volume.
	}
}

// Play plays a single sound by exact name.
func (a *Audio) Play(name string) { a.PlaySound(name, 1, 1.0) }

func (a *Audio) PlayMusic(name string) {
	if a.menuMusic != nil {
		a.menuMusic.Stop(0)
	}
	if a.ambientMusic != nil {
		a.ambientMusic.Stop(0)
	}

	if name == "menu_theme" && a.menuMusic != nil {
		a.menuMusic.Play(0)
	} else if name == "ambience" && a.ambientMusic != nil {
		a.ambientMusic.Play(0)
	}
}

// FadeThrust starts or stops the continuous thrust sound with a fade.
func (a *Audio) FadeThrust(active bool) {
	if a.thrustTrack == nil {
		return
	}
	if active && !a.thrustPlaying {
		a.thrustTrack.Play(200) // 200ms fade in
		a.thrustPlaying = true
	} else if !active && a.thrustPlaying {
		a.thrustTrack.Stop(200) // 200ms fade out
		a.thrustPlaying = false
	}
}

func (a *Audio) Destroy() {
	if a.mixer != nil {
		a.mixer.Destroy()
	}
}
