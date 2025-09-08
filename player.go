package main

import (
	"fmt"
	"math/rand"
)

type Player struct {
	At       TileID // current tile
	Strength int
	Magic    int
	Health   int
	Gold     int
	Cards    []Card
}

func NewPlayer(at TileID, strength int, magic int, health int) *Player {
	return &Player{
		At:       at,
		Strength: strength,
		Magic:    magic,
		Health:   health,
		Gold:     10,
	}
}

// Put near your other types
type Outcome int

const (
	OutcomeNone Outcome = iota
	OutcomeWin          // beat the monster; card collected
	OutcomeLoss         // lost to the monster; took damage
	OutcomeTie          // tie; no effect (tweak if you want)
	OutcomeBuff         // took a buff
)

type InteractResult struct {
	Card    Card
	Outcome Outcome

	// Monster fight rolls/totals (only meaningful for monster cards)
	YourDie int
	MonDie  int
	YourTot int
	MonTot  int

	// State deltas (useful for logging)
	HPBefore    int
	HPAfter     int
	StrBefore   int
	StrAfter    int
	MagicBefore int
	MagicAfter  int
	GoldBefore  int
	GoldAfter   int

	// Friendly summary you can show in the UI
	Message string
}

func (p *Player) Interact(card *Card) InteractResult {
	res := InteractResult{
		Card:        *card,
		HPBefore:    p.Health,
		HPAfter:     p.Health,
		StrBefore:   p.Strength,
		StrAfter:    p.Strength,
		MagicBefore: p.Magic,
		MagicAfter:  p.Magic,
		GoldBefore:  p.Gold,
		GoldAfter:   p.Gold,
		Outcome:     OutcomeNone,
	}

	switch card.Type {
	case monsterType, magicMonsterType:
		yourDie := rand.Intn(6) + 1

		monDie := rand.Intn(6) + 1
		yourTot := p.Strength + yourDie
		var monTot int
		if card.Type == monsterType {
			monTot = card.Strength + monDie
		} else {
			monTot = card.Magic + monDie
		}

		res.YourDie = yourDie
		res.MonDie = monDie
		res.YourTot = yourTot
		res.MonTot = monTot

		if yourTot > monTot {
			// win: collect the card and gain gold
			p.Cards = append(p.Cards, *card)
			goldReward := rand.Intn(3) + 1 // 1-3 gold
			p.Gold += goldReward
			res.GoldAfter = p.Gold
			res.Outcome = OutcomeWin
			res.Message = fmt.Sprintf("You defeated the monster! (Card collected, +%d gold)", goldReward)
		} else if monTot > yourTot {
			// loss: take damage
			if p.Health > 0 {
				p.Health -= 1
			}
			res.Outcome = OutcomeLoss
			res.HPAfter = p.Health
			res.Message = "The monster wounds you! (-1 Health)"
		} else {
			res.Outcome = OutcomeTie
			res.Message = "Stalemate… no effect."
		}

	case buffType:
		p.Strength += card.Strength
		p.Magic += card.Magic
		res.StrAfter = p.Strength
		res.MagicAfter = p.Magic
		res.Outcome = OutcomeBuff
		p.Cards = append(p.Cards, *card)

		if card.Strength > 0 && card.Magic > 0 {
			res.Message = fmt.Sprintf("You feel empowered! (+%d Strength, +%d Magic)", card.Strength, card.Magic)
		} else if card.Strength > 0 {
			res.Message = fmt.Sprintf("You feel stronger! (+%d Strength)", card.Strength)
		} else if card.Magic > 0 {
			res.Message = fmt.Sprintf("You feel more magical! (+%d Magic)", card.Magic)
		}

	case shopItemType:
		p.Strength += card.Strength
		p.Magic += card.Magic
		res.StrAfter = p.Strength
		res.MagicAfter = p.Magic
		res.Outcome = OutcomeBuff
		// Note: Shop items are consumed when used, unlike buffs

		if card.Strength > 0 && card.Magic > 0 {
			res.Message = fmt.Sprintf("Used %s! (+%d Strength, +%d Magic)", card.Title, card.Strength, card.Magic)
		} else if card.Strength > 0 {
			res.Message = fmt.Sprintf("Used %s! (+%d Strength)", card.Title, card.Strength)
		} else if card.Magic > 0 {
			res.Message = fmt.Sprintf("Used %s! (+%d Magic)", card.Title, card.Magic)
		} else {
			res.Message = fmt.Sprintf("Used %s!", card.Title)
		}
	}

	return res
}

// Helper functions for inventory system
func (p *Player) GetMonsterStrengthTotal() int {
	total := 0
	for _, card := range p.Cards {
		if card.Type == monsterType {
			total += card.Strength
		}
	}
	return total
}

func (p *Player) GetMagicMonsterTotal() int {
	total := 0
	for _, card := range p.Cards {
		if card.Type == magicMonsterType {
			total += card.Magic
		}
	}
	return total
}

func (p *Player) ExchangeMonsterStrength() bool {
	needed := 7
	if p.GetMonsterStrengthTotal() < needed {
		return false
	}

	// Remove monster cards with total strength >= 7
	remaining := needed
	newCards := []Card{}

	for _, card := range p.Cards {
		if card.Type == monsterType && remaining > 0 {
			// Consume this monster card to fulfill the exchange
			remaining -= card.Strength
			if remaining <= 0 {
				remaining = 0 // We've consumed enough
			}
			// Card is consumed, don't add to newCards
		} else {
			newCards = append(newCards, card)
		}
	}

	p.Cards = newCards
	p.Strength++
	return true
}

func (p *Player) ExchangeMagicMonster() bool {
	needed := 7
	if p.GetMagicMonsterTotal() < needed {
		return false
	}

	// Remove magic monster cards with total magic >= 7
	remaining := needed
	newCards := []Card{}

	for _, card := range p.Cards {
		if card.Type == magicMonsterType && remaining > 0 {
			// Consume this magic monster card to fulfill the exchange
			remaining -= card.Magic
			if remaining <= 0 {
				remaining = 0 // We've consumed enough
			}
			// Card is consumed, don't add to newCards
		} else {
			newCards = append(newCards, card)
		}
	}

	p.Cards = newCards
	p.Magic++
	return true
}
