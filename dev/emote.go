package main

import (
	"time"
)

const (
	angle    = "angle"
	sick     = "sick"
	itsPoker = "itsPoker"

	EmoteDuration = 5000 * time.Millisecond
	EmoteCooldown = 120 * time.Second
)

func isValidEmote(t string) bool {
	switch t {
	case angle:
		return true
	case sick:
		return true
	case itsPoker:
		return true
	default:
		return false
	}
}

func emoteDisplay(t string) string {
	switch t {
	case angle:
		return "What an 📐"
	case sick:
		return "So Sick"
	case itsPoker:
		return "It's Poker"
	default:
		return t // fallback
	}
}

func emoteAudio(t string) string {
	switch t {
	case angle:
		return "/static/sounds/angle.mp3"
	case sick:
		return "/static/sounds/so_sick.mp3"
	case itsPoker:
		return "/static/sounds/its_poker.mp3"
	default:
		return t // fallback
	}
}
func (r *Room) handleEmote(p *Player, emoteType string) {
	if !isValidEmote(emoteType) || r.currentHand == nil {
		return
	}
	now := time.Now()

	// cooldown
	if !p.nextEmoteAt.IsZero() && now.Before(p.nextEmoteAt) {
		return
	}
	p.nextEmoteAt = now.Add(EmoteCooldown)

	// set per-player emote runtime
	p.emoteText = emoteDisplay(emoteType)
	p.emoteAudio = emoteAudio(emoteType) // e.g. "static/sounds/angle.mp3"
	p.emoteUntil = now.Add(EmoteDuration)
}
