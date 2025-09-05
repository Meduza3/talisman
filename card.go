package main

import (
	"fmt"
	"math/rand"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type CardType string

var monsterType CardType = "monster"
var buffType CardType = "buff"

type Card struct {
	Type        CardType
	Strength    int
	Title, Text string
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
	rl.DrawText(title, x+18, y+16, 28, rl.Black)

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
	rl.DrawText("Enter = confirm   Esc = cancel", x+18, y+int32(h)-36, 20, rl.Gray)

	// input (edge-triggered)
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
		rl.DrawText(line, x, y+int32(i)*(fs+6), fs, col)
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
	return Card{
		Type:     monsterType,
		Strength: str,
		Title:    "A Monster Appears!",
		Text:     fmt.Sprintf("A fearsome foe blocks the way.\nMonster Strength: %d", str),
	}
}
func NewBuff(str int) Card {
	return Card{
		Type:     buffType,
		Strength: str,
		Title:    "A Blessing!",
		Text:     fmt.Sprintf("A boon empowers you.\nGain +%d Strength.", str),
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
