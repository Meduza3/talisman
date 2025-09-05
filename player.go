package main

import "math/rand"

type Player struct {
	At       TileID // current tile
	Strength int
	Health   int
	Cards    []Card
}

func NewPlayer(at TileID, strength int, health int) *Player {
	return &Player{
		At:       at,
		Strength: strength,
		Health:   health,
	}
}

func (p *Player) Interact(card *Card) {
	switch card.Type {
	case monsterType:
		yourRoll := p.Strength + rand.Intn(6) + 1
		monsterRoll := card.Strength + rand.Intn(6) + 1
		if yourRoll > monsterRoll {
			// win: keep the trophy
			p.Cards = append(p.Cards, *card)
		} else if monsterRoll > yourRoll {
			p.Health -= 1
			if p.Health < 0 {
				p.Health = 0
			}
		}
	case buffType:
		// apply and keep a record so it’s visible in the HUD
		p.Strength += card.Strength
		p.Cards = append(p.Cards, *card)
	}
}
