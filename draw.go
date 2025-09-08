package main

import rl "github.com/gen2brain/raylib-go/raylib"

// --- add these helpers somewhere near your draw code ---
func darken(c rl.Color, factor float32) rl.Color {
	if factor < 0 {
		factor = 0
	}
	if factor > 1 {
		factor = 1
	}
	return rl.NewColor(
		uint8(float32(c.R)*factor),
		uint8(float32(c.G)*factor),
		uint8(float32(c.B)*factor),
		c.A,
	)
}

func isZeroColor(c rl.Color) bool { return c.R == 0 && c.G == 0 && c.B == 0 && c.A == 0 }

func lighten(c rl.Color, factor float32) rl.Color {
	if factor < 0 {
		factor = 0
	}
	invFactor := 1.0 - factor
	return rl.NewColor(
		uint8(float32(c.R)*invFactor+255*factor),
		uint8(float32(c.G)*invFactor+255*factor),
		uint8(float32(c.B)*invFactor+255*factor),
		c.A,
	)
}

// Talisman-inspired colors
var shopGold = rl.NewColor(220, 180, 60, 255)      // Rich gold for shops
var bridgeStone = rl.NewColor(140, 130, 120, 255)  // Ancient stone bridges
var playerColor = rl.NewColor(220, 50, 50, 255)    // Bright red for player
