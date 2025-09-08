package main

import (
	"fmt"
	"math/rand"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Helper function for drawing text with custom font (shared with main.go)
func drawTextCard(text string, x, y int32, fontSize int32, color rl.Color) {
	// To use a custom font, uncomment the line below and comment out rl.DrawText:
	rl.DrawTextEx(gameFont, text, rl.NewVector2(float32(x), float32(y)), float32(fontSize), 1.0, color)
	// rl.DrawText(text, x, y, fontSize, color)
}

type CardType string

var monsterType CardType = "monster"
var magicMonsterType CardType = "magicMonster"
var buffType CardType = "buff"
var shopItemType CardType = "shopItem"

type Card struct {
	Type        CardType
	Strength    int
	Magic       int
	Title, Text string
}

type Deck struct {
	Name  string
	Cards []Card
}

var deck1 Deck = Deck{
	Name: "deck1",
	Cards: []Card{
		NewMonster(2),
		NewBuff(1),
		NewMonster(3),
	},
}

var deck2 Deck = Deck{
	Name: "deck2",
	Cards: []Card{
		NewMonster(2),
		NewBuff(2),
		NewMonster(5),
	},
}

var deck3 Deck = Deck{
	Name: "deck3",
	Cards: []Card{
		NewMonster(10),
		NewBuff(5),
		NewMonster(15),
	},
}

var deck4 Deck = Deck{
	Name: "deck4",
	Cards: []Card{
		NewMonster(8),
		NewBuff(3),
		NewMonster(12),
		NewMonster(10),
	},
}

var deck5 Deck = Deck{
	Name: "deck5",
	Cards: []Card{
		NewMonster(15),
		NewBuff(6),
		NewMonster(18),
		NewMonster(20),
	},
}

var deck6 Deck = Deck{
	Name: "deck6",
	Cards: []Card{
		NewMonster(25),
		NewBuff(8),
		NewMonster(30),
		NewMonster(22),
	},
}

func (c *Card) Display() (confirm, cancel bool) {
	// modal rect (centered above the menu bar)
	w, h := float32(540), float32(260)
	cx := float32(screenWidth) / 2
	cy := float32(screenHeight-menuHeight)/2 + 40
	x, y := int32(cx-w/2), int32(cy-h/2)

	// panel
	rl.DrawRectangle(x-4, y-4, int32(w)+8, int32(h)+8, rl.NewColor(0, 0, 0, 40))
	rl.DrawRectangle(x, y, int32(w), int32(h), rl.NewColor(245, 245, 245, 255))
	rl.DrawRectangleLines(x, y, int32(w), int32(h), rl.DarkGray)

	// title
	title := c.Title
	if title == "" {
		if c.Type == monsterType {
			title = "A Monster Appears!"
		} else {
			title = "You Found a Buff!"
		}
	}
	drawTextCard(title, x+18, y+16, 28, rl.Black)

	// body
	body := c.Text
	if body == "" {
		switch c.Type {
		case monsterType:
			body = fmt.Sprintf("Monster Strength: %d\nPress Enter to fight.\nEsc to ignore.", c.Strength)
		case buffType:
			body = fmt.Sprintf("Gain +%d Strength.\nPress Enter to take it.\nEsc to ignore.", c.Strength)
		}
	}
	drawMultiline(body, x+18, y+64, 22, rl.DarkGray, int(w)-36)

	// footer
	drawTextCard("Enter = confirm   Esc = cancel", x+18, y+int32(h)-36, 20, rl.Gray)

	// Return input states for external handling
	confirm = rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter)
	cancel = rl.IsKeyPressed(rl.KeyEscape)
	return
}
func drawMultiline(s string, x, y int32, fs int32, col rl.Color, maxWidth int) {
	lines := []string{""}
	for _, word := range splitWordsPreserveNL(s) {
		if word == "\n" {
			lines = append(lines, "")
			continue
		}
		next := lines[len(lines)-1]
		if next != "" {
			next += " "
		}
		next += word
		if rl.MeasureText(next, fs) > int32(maxWidth) {
			// put word on a new line
			lines = append(lines, word)
		} else {
			lines[len(lines)-1] = next
		}
	}
	for i, line := range lines {
		drawTextCard(line, x, y+int32(i)*(fs+6), fs, col)
	}
}

func splitWordsPreserveNL(s string) []string {
	out, cur := []string{}, ""
	for _, r := range s {
		if r == '\n' {
			if cur != "" {
				out = append(out, cur)
				cur = ""
			}
			out = append(out, "\n")
			continue
		}
		if r == ' ' || r == '\t' {
			if cur != "" {
				out = append(out, cur)
				cur = ""
			}
		} else {
			cur += string(r)
		}
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

// ---------- Card factories ----------

func NewMonster(str int) Card {
	if rand.Intn(2) == 1 {
		return NewMonsterStrength(str)
	} else {
		return NewMonsterMagic(str)
	}
}

func NewMonsterStrength(str int) Card {
	return Card{
		Type:     monsterType,
		Strength: str,
		Title:    "A Monster Appears!",
		Text:     fmt.Sprintf("A fearsome foe blocks the way.\nMonster Strength: %d", str),
	}
}
func NewMonsterMagic(magic int) Card {
	return Card{
		Type:  magicMonsterType,
		Magic: magic,
		Title: "A Magic Monster Appears!",
		Text:  fmt.Sprintf("A fearsome foe blocks the way.\nMonster Magic: %d", magic),
	}
}
func NewBuff(str int) Card {
	if rand.Intn(2) == 1 {
		return NewBuffStrength(str)
	} else {
		return NewBuffMagic(str)
	}
}

func NewBuffStrength(str int) Card {
	return Card{
		Type:     buffType,
		Strength: str,
		Title:    "A Blessing!",
		Text:     fmt.Sprintf("A boon empowers you.\nGain +%d Strength.", str),
	}
}
func NewBuffMagic(str int) Card {
	return Card{
		Type:  buffType,
		Magic: str,
		Title: "A Magic Blessing!",
		Text:  fmt.Sprintf("A boon empowers you.\nGain +%d Magic.", str),
	}
}

// Randomized helpers (tweak ranges to taste)
func RandMonster(minStr, maxStr int) Card {
	if maxStr < minStr {
		maxStr = minStr
	}
	return NewMonster(minStr + rand.Intn(maxStr-minStr+1))
}
func RandBuff(minStr, maxStr int) Card {
	if maxStr < minStr {
		maxStr = minStr
	}
	return NewBuff(minStr + rand.Intn(maxStr-minStr+1))
}
func RandCard() Card {
	if rand.Float32() < 0.55 {
		return RandMonster(2, 8)
	}
	return RandBuff(1, 3)
}

// Shop-specific decks for each shopkeeper type
var mysticShopDeck Deck = Deck{
	Name: "mysticShopDeck",
	Cards: []Card{
		{Type: shopItemType, Magic: 2, Title: "Wizard's Amulet", Text: "An ancient amulet crackling with arcane power.\nGain +2 Magic."},
		{Type: shopItemType, Magic: 3, Title: "Staff of Elements", Text: "A powerful staff that channels elemental forces.\nGain +3 Magic."},
		{Type: shopItemType, Magic: 1, Title: "Mana Potion", Text: "A shimmering blue liquid that restores magical essence.\nGain +1 Magic."},
		{Type: shopItemType, Magic: 2, Title: "Enchanted Robes", Text: "Mystical robes woven with silver thread.\nGain +2 Magic."},
		{Type: shopItemType, Magic: 3, Title: "Archmage's Crown", Text: "A crown that once belonged to the greatest wizard.\nGain +3 Magic."},
		{Type: shopItemType, Magic: 3, Title: "Void Crystal", Text: "A dark crystal pulsing with otherworldly energy.\nGain +3 Magic."},
	},
}

var ogreShopDeck Deck = Deck{
	Name: "ogreShopDeck",
	Cards: []Card{
		{Type: shopItemType, Strength: 3, Title: "Ogre's Club", Text: "A massive wooden club used by ogre warriors.\nGain +3 Strength."},
		{Type: shopItemType, Strength: 2, Title: "Beast Hide Armor", Text: "Thick armor made from giant beast hide.\nGain +2 Strength."},
		{Type: shopItemType, Strength: 3, Title: "Titan's Hammer", Text: "A massive war hammer forged by titans.\nGain +3 Strength."},
		{Type: shopItemType, Strength: 2, Title: "Giant's Belt", Text: "A leather belt once worn by a mountain giant.\nGain +2 Strength."},
		{Type: shopItemType, Strength: 1, Title: "Stone Knuckles", Text: "Heavy stone gauntlets that crush enemies.\nGain +1 Strength."},
		{Type: shopItemType, Strength: 3, Title: "Berserker's Charm", Text: "A wild charm that unleashes inner rage.\nGain +3 Strength."},
	},
}

var monkeyShopDeck Deck = Deck{
	Name: "monkeyShopDeck",
	Cards: []Card{
		{Type: shopItemType, Strength: 1, Magic: 1, Title: "Banana of Wisdom", Text: "A magical fruit that enhances mind and body.\nGain +1 Strength and +1 Magic."},
		{Type: shopItemType, Magic: 2, Title: "Jungle Vine Staff", Text: "A staff made from enchanted jungle vines.\nGain +2 Magic."},
		{Type: shopItemType, Strength: 2, Title: "Monkey Paw Gloves", Text: "Nimble gloves that increase dexterity.\nGain +2 Strength."},
		{Type: shopItemType, Strength: 1, Magic: 2, Title: "Scholar's Banana", Text: "A fruit inscribed with ancient knowledge.\nGain +1 Strength and +2 Magic."},
		{Type: shopItemType, Magic: 1, Title: "Chattering Scroll", Text: "A scroll that whispers jungle secrets.\nGain +1 Magic."},
		{Type: shopItemType, Strength: 1, Title: "Swinging Rope", Text: "A rope that increases agility and strength.\nGain +1 Strength."},
	},
}

var pigShopDeck Deck = Deck{
	Name: "pigShopDeck",
	Cards: []Card{
		{Type: shopItemType, Strength: 1, Title: "Truffle Hunter's Nose", Text: "Enhances your ability to find hidden treasures.\nGain +1 Strength."},
		{Type: shopItemType, Magic: 1, Title: "Mud Bath Essence", Text: "A relaxing potion that clears the mind.\nGain +1 Magic."},
		{Type: shopItemType, Strength: 2, Title: "Bacon Shield", Text: "A surprisingly sturdy shield made of cured meat.\nGain +2 Strength."},
		{Type: shopItemType, Magic: 2, Title: "Snorting Powder", Text: "A magical dust that enhances mental abilities.\nGain +2 Magic."},
		{Type: shopItemType, Strength: 1, Magic: 1, Title: "Farm Fresh Meal", Text: "A hearty meal that nourishes body and soul.\nGain +1 Strength and +1 Magic."},
		{Type: shopItemType, Strength: 2, Title: "Pig Iron Gauntlets", Text: "Heavy iron gloves forged in pig-shaped molds.\nGain +2 Strength."},
	},
}

var dolphinShopDeck Deck = Deck{
	Name: "dolphinShopDeck",
	Cards: []Card{
		{Type: shopItemType, Magic: 2, Title: "Pearl of the Deep", Text: "A lustrous pearl containing ocean magic.\nGain +2 Magic."},
		{Type: shopItemType, Strength: 2, Title: "Whale Bone Sword", Text: "A sword carved from ancient whale bone.\nGain +2 Strength."},
		{Type: shopItemType, Magic: 3, Title: "Tidal Wave Orb", Text: "An orb that commands the power of the tides.\nGain +3 Magic."},
		{Type: shopItemType, Strength: 1, Magic: 2, Title: "Dolphin's Wisdom", Text: "Ancient knowledge from the ocean depths.\nGain +1 Strength and +2 Magic."},
		{Type: shopItemType, Magic: 1, Title: "Sea Foam Potion", Text: "A bubbly potion made from ocean waves.\nGain +1 Magic."},
		{Type: shopItemType, Strength: 1, Title: "Coral Armor", Text: "Living coral that protects and strengthens.\nGain +1 Strength."},
	},
}

func RandShopCard(keeperType int) Card {
	switch keeperType {
	case 0:
		return mysticShopDeck.Cards[rand.Intn(len(mysticShopDeck.Cards))]
	case 1:
		return ogreShopDeck.Cards[rand.Intn(len(ogreShopDeck.Cards))]
	case 2:
		return monkeyShopDeck.Cards[rand.Intn(len(monkeyShopDeck.Cards))]
	case 3:
		return pigShopDeck.Cards[rand.Intn(len(pigShopDeck.Cards))]
	case 4:
		return dolphinShopDeck.Cards[rand.Intn(len(dolphinShopDeck.Cards))]
	default:
		return mysticShopDeck.Cards[rand.Intn(len(mysticShopDeck.Cards))]
	}
}
