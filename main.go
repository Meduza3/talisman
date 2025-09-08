package main

import (
	"fmt"
	"math"
	"math/rand"
	"runtime"
	"strings"

	_ "github.com/gen2brain/raylib-go/raygui"
	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	screenWidth  = 1600
	screenHeight = 900
	tileSize     = 44
	fontSize     = 14
	tilePad      = 10 // spacing between tiles around a loop (affects loop radius)
	margin       = 80 // keep loops away from window edges
	loopGap      = 30 // minimum clearance between different loops (circle-to-circle)
	menuHeight   = 120

	crossLinkChance = 0.95
	shopChance      = 0.20
)

var gameFont rl.Font

// Helper function for drawing text with custom font
func drawText(text string, x, y int32, fontSize int32, color rl.Color) {
	// To use a custom font, uncomment the line below and comment out rl.DrawText:
	rl.DrawTextEx(gameFont, text, rl.NewVector2(float32(x), float32(y)), float32(fontSize), 1.0, color)
	// rl.DrawText(text, x, y, fontSize, color)
}

// Helper function to get card name for logging
func cardName(c *Card) string {
	switch c.Type {
	case monsterType:
		return fmt.Sprintf("Monster(STR %d)", c.Strength)
	case magicMonsterType:
		return fmt.Sprintf("Magic Monster(MAG %d)", c.Magic)
	case buffType:
		if c.Strength > 0 {
			return fmt.Sprintf("Buff(+%d STR)", c.Strength)
		}
		return fmt.Sprintf("Buff(+%d MAG)", c.Magic)
	}
	return "Unknown Card"
}

func main() {
	runtime.LockOSThread() // <-- must be first on macOS

	rl.InitWindow(int32(screenWidth), int32(screenHeight), "Talisman")
	defer rl.CloseWindow()
	rl.SetTargetFPS(60)

	// Completely disable escape key from closing the game
	rl.SetExitKey(0) // 0 = no key will exit the game

	// Initialize font - supports both .ttf and .otf files
	// Example:
	gameFont = rl.LoadFont("assets/mediaval_font.otf")
	// gameFont = rl.GetFontDefault()

	numOfLoops := rand.Intn(30) + 20
	loopCounts := []int{}
	for range numOfLoops {
		loopCounts = append(loopCounts, rand.Intn(20)+8)
	}
	world := BuildWorldCountsRandom(loopCounts)

	finalizeLoopIndices(&world)

	game := Game{
		Player: *NewPlayer(TileID{0, 0}, 3, 3, 5),
		World:  &world,
	}
	// Center the camera on the playfield above the menu bar
	cam = rl.Camera2D{
		Target: playerPos(&world, &game),
		// Offset is where the Target appears on the screen (we want it centered horizontally,
		// and vertically centered in the play area above the menu).
		Offset:   rl.NewVector2(float32(screenWidth)/2, float32(screenHeight-menuHeight)/2),
		Rotation: 0,
		Zoom:     1.0,
	}
	for !rl.WindowShouldClose() {
		dt := rl.GetFrameTime()

		// Input (disabled while auto-moving)
		// // Zoom with mouse wheel
		if wheel := rl.GetMouseWheelMove(); wheel != 0 {
			cam.Zoom *= 1 + 0.1*wheel
			cam.Zoom = clamp(cam.Zoom, 0.4, 5.5)
		}
		// Reset zoom with 0 key (optional)
		if rl.IsKeyPressed(rl.KeyZero) {
			cam.Zoom = 1.0
		}
		updateCamera(&cam, playerPos(&world, &game), dt)

		if rl.IsKeyPressed(rl.KeySpace) {
			numOfLoops := rand.Intn(30) + 20
			loopCounts := []int{}
			for range numOfLoops {
				loopCounts = append(loopCounts, rand.Intn(20)+8)
			}
			world = BuildWorldCountsRandom(loopCounts)
			// reset player to safe starting position
			game.Player.At = TileID{0, 0}
			game.StepsRemaining = 0
			game.Phase = PhaseIdle
		}
		// Handle inventory input
		if game.InventoryActive {
			if rl.IsKeyPressed(rl.KeyEscape) || rl.IsKeyPressed(rl.KeyQ) {
				game.InventoryActive = false
			}

			// Calculate total items (cards + 2 exchange buttons)
			totalCards := len(game.Player.Cards)
			totalItems := totalCards + 2 // cards + 2 exchange buttons

			// Handle horizontal navigation with left/right arrows
			if rl.IsKeyPressed(rl.KeyLeft) {
				game.InventorySelectedIndex = max(0, game.InventorySelectedIndex-1)
			}
			if rl.IsKeyPressed(rl.KeyRight) {
				game.InventorySelectedIndex = min(totalItems-1, game.InventorySelectedIndex+1)
			}

			// Handle vertical scrolling (up/down) - legacy support
			if rl.IsKeyPressed(rl.KeyUp) {
				game.InventorySelectedIndex = max(0, game.InventorySelectedIndex-5)
			}
			if rl.IsKeyPressed(rl.KeyDown) {
				game.InventorySelectedIndex = min(totalItems-1, game.InventorySelectedIndex+5)
			}

			// Adjust scroll offset to keep selected item visible
			if game.InventorySelectedIndex < totalCards {
				// Selected item is a card
				cardsPerRow := 6 // Adjust based on card display
				visibleRows := 3
				cardsPerPage := cardsPerRow * visibleRows

				if game.InventorySelectedIndex < game.InventoryCardScrollOffset {
					game.InventoryCardScrollOffset = game.InventorySelectedIndex
				} else if game.InventorySelectedIndex >= game.InventoryCardScrollOffset+cardsPerPage {
					game.InventoryCardScrollOffset = game.InventorySelectedIndex - cardsPerPage + 1
				}
			}

			// Handle item usage/exchange activation with Enter
			if rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter) {
				if game.InventorySelectedIndex < totalCards {
					// Selected item is a card - use it if it's a shop item or buff
					selectedCard := game.Player.Cards[game.InventorySelectedIndex]
					if selectedCard.Type == shopItemType {
						// Shop items do nothing when used from inventory (effects already applied at purchase)
						game.logf("You examine %s - its power has already been absorbed.", selectedCard.Title)

					}
				} else if game.InventorySelectedIndex == totalCards {
					// First exchange button (strength)
					if game.Player.ExchangeMonsterStrength() {
						game.logf("Exchanged monster strength for +1 STR")
					}
				} else if game.InventorySelectedIndex == totalCards+1 {
					// Second exchange button (magic)
					if game.Player.ExchangeMagicMonster() {
						game.logf("Exchanged magic monster strength for +1 MAG")
					}
				}
			}

			// Legacy number key and mouse support
			if rl.IsKeyPressed(rl.KeyOne) {
				if game.Player.ExchangeMonsterStrength() {
					game.logf("Exchanged monster strength for +1 STR")
				}
			}
			if rl.IsKeyPressed(rl.KeyTwo) {
				if game.Player.ExchangeMagicMonster() {
					game.logf("Exchanged magic monster strength for +1 MAG")
				}
			}
			// Handle mouse clicks on buttons
			if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
				mousePos := rl.GetMousePosition()
				if rl.CheckCollisionPointRec(mousePos, game.StrengthButtonBounds) {
					if game.Player.ExchangeMonsterStrength() {
						game.logf("Exchanged monster strength for +1 STR")
					}
				}
				if rl.CheckCollisionPointRec(mousePos, game.MagicButtonBounds) {
					if game.Player.ExchangeMagicMonster() {
						game.logf("Exchanged magic monster strength for +1 MAG")
					}
				}
			}
			goto AFTER_INPUT
		}

		// Handle shop input
		if game.ShopActive {
			if rl.IsKeyPressed(rl.KeyEscape) || rl.IsKeyPressed(rl.KeyQ) {
				game.ShopActive = false
			}
			// Navigate cards with arrow keys or 1/2/3
			if rl.IsKeyPressed(rl.KeyRight) {
				game.ShopSelected = (game.ShopSelected + 1) % 4 // 0-2 for cards, 3 for exit
			}
			if rl.IsKeyPressed(rl.KeyLeft) {
				game.ShopSelected = (game.ShopSelected - 1 + 4) % 4
			}
			if rl.IsKeyPressed(rl.KeyUp) {
				if game.ShopSelected == 3 { // if on exit button, go to card 2
					game.ShopSelected = 2
				} else {
					game.ShopSelected = (game.ShopSelected - 1 + 3) % 3
				}
			}
			if rl.IsKeyPressed(rl.KeyDown) {
				if game.ShopSelected < 3 { // if on a card, go to exit button
					game.ShopSelected = 3
				}
			}
			if rl.IsKeyPressed(rl.KeyOne) {
				game.ShopSelected = 0
			}
			if rl.IsKeyPressed(rl.KeyTwo) {
				game.ShopSelected = 1
			}
			if rl.IsKeyPressed(rl.KeyThree) {
				game.ShopSelected = 2
			}
			// Purchase with Enter or close shop if exit selected
			if rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter) {
				if game.ShopSelected == 3 { // Exit button selected
					game.ShopActive = false
				} else { // Card selected
					price := game.ShopPrices[game.ShopSelected]
					if game.Player.Gold >= price {
						game.Player.Gold -= price
						purchasedCard := game.ShopCards[game.ShopSelected]

						// Apply shop item effects immediately or add to inventory
						if purchasedCard.Type == shopItemType {
							// Apply effects immediately for shop items
							game.Player.Strength += purchasedCard.Strength
							game.Player.Magic += purchasedCard.Magic
							// Add to inventory for display purposes
							game.Player.Cards = append(game.Player.Cards, purchasedCard)
							if purchasedCard.Title != "" {
								effectMsg := ""
								if purchasedCard.Strength > 0 && purchasedCard.Magic > 0 {
									effectMsg = fmt.Sprintf(" (+%d STR, +%d MAG)", purchasedCard.Strength, purchasedCard.Magic)
								} else if purchasedCard.Strength > 0 {
									effectMsg = fmt.Sprintf(" (+%d STR)", purchasedCard.Strength)
								} else if purchasedCard.Magic > 0 {
									effectMsg = fmt.Sprintf(" (+%d MAG)", purchasedCard.Magic)
								}
								game.logf("Purchased %s for %d gold%s", purchasedCard.Title, price, effectMsg)
							} else {
								game.logf("Purchased shop item for %d gold", price)
							}
						} else {
							// Regular cards go to inventory without immediate effects
							game.Player.Cards = append(game.Player.Cards, purchasedCard)
							game.logf("Purchased item for %d gold (added to inventory)", price)
						}

						game.LastPurchaseTime = float32(rl.GetTime()) // Trigger shopkeeper animation

						// Replace purchased card with new one and update persistent shop data
						cur := world.Loops[game.Player.At.Loop].Tiles[game.Player.At.Index]
						var keeperType int
						for _, link := range cur.Links {
							if isShopTile(&world, link) {
								shopTile := &world.Loops[link.Loop].Tiles[link.Index]
								if shopTile.ShopData != nil {
									keeperType = shopTile.ShopData.KeeperType
									break
								}
							}
						}
						newCard := RandShopCard(keeperType)
						game.ShopCards[game.ShopSelected] = newCard
						baseCost := 0
						if newCard.Strength > 0 && newCard.Magic > 0 {
							baseCost = (newCard.Strength + newCard.Magic) * 4
						} else {
							baseCost = max(newCard.Strength, newCard.Magic) * 3
						}
						newPrice := baseCost + rand.Intn(5) + 1
						game.ShopPrices[game.ShopSelected] = newPrice

						// Update the persistent shop data
						cur = world.Loops[game.Player.At.Loop].Tiles[game.Player.At.Index]
						for _, link := range cur.Links {
							if isShopTile(&world, link) {
								shopTile := &world.Loops[link.Loop].Tiles[link.Index]
								if shopTile.ShopData != nil {
									shopTile.ShopData.Cards[game.ShopSelected] = newCard
									shopTile.ShopData.Prices[game.ShopSelected] = newPrice
									break
								}
							}
						}
					} else {
						game.logf("Not enough gold! Need %d, have %d", price, game.Player.Gold)
					}
				}
			}
			// Handle mouse clicks on exit button
			if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
				mousePos := rl.GetMousePosition()
				if rl.CheckCollisionPointRec(mousePos, game.ExitButtonBounds) {
					game.ShopActive = false
				}
			}
			goto AFTER_INPUT
		}

		// Handle card input
		if game.CardActive {
			confirm, cancel := game.Card.Display()
			if !game.CardResolved {
				if confirm {
					res := game.Player.Interact(&game.Card)

					switch res.Card.Type {
					case monsterType, magicMonsterType:
						// detailed fight log
						game.logf("Fight! You: STR %d + d6(%d) = %d", res.StrBefore, res.YourDie, res.YourTot)
						if res.Card.Type == monsterType {
							game.logf("       Monster: STR %d + d6(%d) = %d", res.Card.Strength, res.MonDie, res.MonTot)
						} else {
							game.logf("       Monster: MAG %d + d6(%d) = %d", res.Card.Magic, res.MonDie, res.MonTot)
						}
						switch res.Outcome {
						case OutcomeWin:
							goldGained := res.GoldAfter - res.GoldBefore
							if goldGained > 0 {
								game.logf("Result: You win and take the card. (+%d gold)", goldGained)
							} else {
								game.logf("Result: You win and take the card.")
							}
						case OutcomeLoss:
							game.logf("Result: You lose. HP %d → %d", res.HPBefore, res.HPAfter)
						case OutcomeTie:
							game.logf("Result: Tie. No effect.")
						}
					case buffType:
						// buff log - handle both strength and magic
						strDelta := res.StrAfter - res.StrBefore
						magicDelta := res.MagicAfter - res.MagicBefore

						if strDelta > 0 && magicDelta > 0 {
							game.logf("Buff: +%d STR, +%d MAG (STR %d → %d, MAG %d → %d)",
								strDelta, magicDelta, res.StrBefore, res.StrAfter, res.MagicBefore, res.MagicAfter)
						} else if strDelta > 0 {
							game.logf("Buff: +%d STR (STR %d → %d)", strDelta, res.StrBefore, res.StrAfter)
						} else if magicDelta > 0 {
							game.logf("Buff: +%d MAG (MAG %d → %d)", magicDelta, res.MagicBefore, res.MagicAfter)
						}
					}

					// Show the friendly one-line toast in the modal
					game.CardMsg = res.Message
					game.CardResolved = true
				}
				if cancel {
					// skipped; just close
					game.CardActive = false
				}
			} else {
				// Card is resolved, close on any key EXCEPT escape unless explicitly requested
				if rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter) || rl.IsKeyPressed(rl.KeySpace) {
					game.CardActive = false
				}
				// Only close with escape if user specifically wants to skip the message
				if rl.IsKeyPressed(rl.KeyEscape) {
					game.CardActive = false
				}
			}
			goto AFTER_INPUT
		}
		switch game.Phase {
		case PhaseIdle:
			if rl.IsKeyPressed(rl.KeyR) {
				game.Roll()
			}
			if rl.IsKeyPressed(rl.KeyQ) {
				game.InventoryActive = true
			}
			// (optional) manual testing when idle
			if rl.IsKeyPressed(rl.KeyRight) {
				game.Player.At = world.Loops[game.Player.At.Loop].Tiles[game.Player.At.Index].Next
			}
			if rl.IsKeyPressed(rl.KeyLeft) {
				game.Player.At = world.Loops[game.Player.At.Loop].Tiles[game.Player.At.Index].Prev
			}
			if rl.IsKeyPressed(rl.KeyE) {
				cur := world.Loops[game.Player.At.Loop].Tiles[game.Player.At.Index]
				// Check if player is standing next to a shop (through links)
				shopFound := false
				for _, link := range cur.Links {
					if isShopTile(&world, link) {
						shopTile := &world.Loops[link.Loop].Tiles[link.Index]
						if shopTile.ShopData != nil {
							game.InitShop(shopTile.ShopData)
							game.ShopActive = true
							shopFound = true
							break
						}
					}
				}
				// If no shop found, use original behavior (move through link)
				if !shopFound && len(cur.Links) > 0 {
					game.Player.At = cur.Links[0]
				}
			}
		case PhaseTargetSelect:
			// cycle through all legal landing spots
			if rl.IsKeyPressed(rl.KeyRight) || rl.IsKeyPressed(rl.KeyDown) {
				if len(game.Dests) > 0 {
					game.Selected = (game.Selected + 1) % len(game.Dests)
				}
			}
			if rl.IsKeyPressed(rl.KeyLeft) || rl.IsKeyPressed(rl.KeyUp) {
				if len(game.Dests) > 0 {
					game.Selected = (game.Selected - 1 + len(game.Dests)) % len(game.Dests)
				}
			}
			// confirm: build path and start animating
			if rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter) {
				target := game.Dests[game.Selected]
				game.Path = buildPathTo(&world, game.Player.At, game.LastRoll, target)
				game.StepsRemaining = len(game.Path) // drive animation by path length now
				game.Phase = PhaseAnimating
			}
			// cancel
			if rl.IsKeyPressed(rl.KeyEscape) {
				game.Phase = PhaseIdle
				game.Dests = nil
			}

		case PhaseAnimating:
			// no input while animating
		}
	AFTER_INPUT:
		// Auto-resolve movement for the current roll
		game.Update(dt, &world)
		// Smoothly follow player
		updateCamera(&cam, playerPos(&world, &game), dt)
		// Draw
		drawWorld(&world, &game)
	}
}

// BFS over Links only (zero-cost moves)
func collectLinkClosure(start TileID, w *World) []TileID {
	q := []TileID{start}
	seen := map[TileID]bool{start: true}
	out := []TileID{start}
	for len(q) > 0 {
		cur := q[0]
		q = q[1:]
		t := &w.Loops[cur.Loop].Tiles[cur.Index]
		for _, nb := range t.Links {
			if !seen[nb] {
				seen[nb] = true
				out = append(out, nb)
				q = append(q, nb)
			}
		}
	}
	return out
}

func drawLoops(w *World, highlights map[TileID]rl.Color) {
	for li, loop := range w.Loops {
		// base fill from LoopType, with safe defaults
		baseFill := loop.Type.Color
		if isZeroColor(baseFill) {
			baseFill = rl.NewColor(180, 170, 160, 255) // Neutral stone for bridges/shops
		}
		if baseFill.A == 0 {
			baseFill.A = 255
		}
		baseOutline := darken(baseFill, 0.45)

		for i, t := range loop.Tiles {
			fill, outline := baseFill, baseOutline

			// classify special tiles first
			if t.Bridge {
				fill = bridgeStone
				outline = darken(bridgeStone, 0.6)
			} else if t.Shop {
				fill = shopGold
				outline = darken(shopGold, 0.5)
			}

			// highlights win visually
			if c, ok := highlights[TileID{Loop: li, Index: i}]; ok {
				fill = c
			}

			x := int32(t.Pos.X - tileSize/2)
			y := int32(t.Pos.Y - tileSize/2)

			// Draw tile with subtle gradient effect
			rl.DrawRectangle(x, y, int32(tileSize), int32(tileSize), fill)

			// Add inner highlight for depth
			if !t.Bridge && !t.Shop {
				innerHighlight := lighten(fill, 0.15)
				rl.DrawRectangle(x+1, y+1, int32(tileSize)-2, int32(tileSize)-2, rl.Fade(innerHighlight, 0.3))
			}

			// Draw outline
			rl.DrawRectangleLines(x, y, int32(tileSize), int32(tileSize), outline)

			// Special effects for shops
			if t.Shop {
				centerX := int32(t.Pos.X)
				centerY := int32(t.Pos.Y)

				// Check if shop is discovered and get shopkeeper type
				if t.ShopData != nil && t.ShopData.Discovered {
					// Draw shopkeeper-specific symbols (160% larger)
					switch t.ShopData.KeeperType {
					case 0: // Mystic - Crystal
						rl.DrawCircle(centerX, centerY, 13, rl.NewColor(140, 60, 160, 255))
						rl.DrawCircleLines(centerX, centerY, 13, rl.NewColor(100, 40, 120, 255))
						rl.DrawCircle(centerX, centerY-3, 5, rl.NewColor(255, 255, 255, 200))
					case 1: // Ogre - Club
						rl.DrawRectangle(centerX-3, centerY-13, 6, 26, rl.NewColor(101, 67, 33, 255))
						rl.DrawRectangle(centerX-10, centerY-13, 20, 10, rl.NewColor(101, 67, 33, 255))
					case 2: // Monkey - Banana (proper curved banana shape)
						// Draw banana body as elongated oval
						rl.DrawCircle(centerX-2, centerY-6, 6, rl.NewColor(255, 255, 100, 255))
						rl.DrawCircle(centerX, centerY, 8, rl.NewColor(255, 255, 100, 255))
						rl.DrawCircle(centerX+3, centerY+8, 6, rl.NewColor(255, 255, 100, 255))
						// Connect them to form banana curve
						rl.DrawRectangle(centerX-6, centerY-6, 12, 8, rl.NewColor(255, 255, 100, 255))
						rl.DrawRectangle(centerX-2, centerY+2, 8, 8, rl.NewColor(255, 255, 100, 255))
						// Brown spots on banana
						rl.DrawCircle(centerX-1, centerY-2, 2, rl.NewColor(200, 150, 0, 255))
						rl.DrawCircle(centerX+2, centerY+3, 2, rl.NewColor(200, 150, 0, 255))
						// Green stem
						rl.DrawRectangle(centerX-1, centerY-10, 3, 6, rl.NewColor(101, 67, 33, 255))
					case 3: // Pig - Snout
						rl.DrawCircle(centerX, centerY, 13, rl.NewColor(255, 180, 180, 255))
						rl.DrawCircle(centerX-5, centerY, 3, rl.NewColor(200, 120, 120, 255))
						rl.DrawCircle(centerX+5, centerY, 3, rl.NewColor(200, 120, 120, 255))
					case 4: // Dolphin - Wave
						rl.DrawCircle(centerX, centerY, 13, rl.NewColor(120, 160, 200, 255))
						rl.DrawRectangle(centerX-10, centerY+3, 20, 3, rl.NewColor(100, 140, 180, 255))
						rl.DrawRectangle(centerX-6, centerY-3, 12, 3, rl.NewColor(100, 140, 180, 255))
					}
				} else {
					// Undiscovered shop - generic gold coin (also 160% larger)
					rl.DrawCircle(centerX, centerY, 13, rl.NewColor(255, 215, 0, 255))
					rl.DrawCircleLines(centerX, centerY, 13, darken(shopGold, 0.7))
				}
			}
		}
	}
}
func drawPlayer(p Player, w *World) {
	pos := w.Loops[p.At.Loop].Tiles[p.At.Index].Pos
	x, y := int32(pos.X), int32(pos.Y)

	// Draw player as a more detailed character token
	// Outer glow
	rl.DrawCircle(x, y, 13, rl.Fade(playerColor, 0.3))
	// Main body
	rl.DrawCircle(x, y, 10, playerColor)
	// Inner highlight
	rl.DrawCircle(x-2, y-2, 4, lighten(playerColor, 0.4))
	// Outline
	rl.DrawCircleLines(x, y, 10, darken(playerColor, 0.5))
}

func drawWorld(w *World, g *Game) {
	rl.BeginDrawing()
	// Talisman-inspired parchment background
	rl.ClearBackground(rl.NewColor(240, 235, 220, 255))

	// Build highlight map (only when choosing a direction)
	hi := map[TileID]rl.Color{}
	if g.Phase == PhaseTargetSelect && len(g.Dests) > 0 {
		for i, id := range g.Dests {
			// Mystical blue glow for possible destinations
			col := rl.NewColor(100, 150, 220, 180)
			if i == g.Selected {
				// Bright golden selection
				col = rl.NewColor(255, 215, 0, 220)
			}
			hi[id] = col
		}
	}

	// --- Draw playfield (everything except menu bar) ---
	// drawLinks(w)
	rl.BeginMode2D(cam)
	drawLoops(w, hi)
	// Draw selection circle for currently selected destination
	if g.Phase == PhaseTargetSelect && len(g.Dests) > 0 && g.Selected < len(g.Dests) {
		selectedTile := g.Dests[g.Selected]
		pos := w.Loops[selectedTile.Loop].Tiles[selectedTile.Index].Pos
		centerX := int32(pos.X)
		centerY := int32(pos.Y)
		// Draw thick circle outline around selected destination
		for thickness := 0; thickness < 4; thickness++ {
			rl.DrawCircleLines(centerX, centerY, float32(28+int32(thickness)), rl.NewColor(255, 255, 255, 200))
		}
	}
	drawPlayer(g.Player, w)
	rl.EndMode2D()
	drawCard(g)
	drawLog(g)
	// --- Draw menu bar ---
	drawMenuBar(g)
	// --- Draw inventory menu ---
	drawInventory(g)
	// --- Draw shop menu ---
	drawShop(g)

	rl.EndDrawing()
}

func drawLog(g *Game) {
	if len(g.Log) == 0 {
		return
	}

	// panel geometry
	pad := int32(10)
	lineH := int32(20)
	lines := int32(min(len(g.Log), logMax))
	w := int32(520)
	h := lines*lineH + pad*2

	x := int32(14)
	y := int32(screenHeight-menuHeight) - h - 10

	// panel
	rl.DrawRectangle(x-2, y-2, w+4, h+4, rl.NewColor(0, 0, 0, 30))
	rl.DrawRectangle(x, y, w, h, rl.NewColor(245, 245, 245, 235))
	rl.DrawRectangleLines(x, y, w, h, rl.NewColor(0, 0, 0, 60))
	drawText("Log", x+8, y+6, 18, rl.DarkGray)

	// lines (oldest to newest; clip to last `lines`)
	start := max(0, len(g.Log)-int(lines))
	curY := y + 8 + lineH
	for i := start; i < len(g.Log); i++ {
		drawText(g.Log[i], x+8, curY, 18, rl.Black)
		curY += lineH
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func drawCard(g *Game) {
	// --- card modal on top (if any) ---
	if g.CardActive {
		g.Card.Display() // Just draw the card, input is handled in main loop

		// Show result message if card is resolved
		if g.CardResolved {
			drawText(g.CardMsg, 40, int32(screenHeight-menuHeight)-80, 22, rl.DarkGreen)
		}
	}
}

func drawInventory(g *Game) {
	if !g.InventoryActive {
		return
	}

	// Larger margins for smaller screen coverage
	margin := int32(150)
	x := margin
	y := margin
	w := int32(screenWidth) - 2*margin
	h := int32(screenHeight) - 2*margin

	// Background overlay
	rl.DrawRectangle(0, 0, screenWidth, screenHeight, rl.NewColor(0, 0, 0, 180))
	rl.DrawRectangle(x-6, y-6, w+12, h+12, rl.NewColor(0, 0, 0, 120))
	rl.DrawRectangle(x, y, w, h, rl.NewColor(45, 35, 25, 250))
	rl.DrawRectangleLines(x, y, w, h, hudAccent)

	// Title bar
	rl.DrawRectangle(x, y, w, 50, rl.NewColor(30, 25, 20, 200))
	drawText("INVENTORY", x+20, y+15, 24, hudAccent)
	drawText("ESC/Q=close • ←→=navigate • Enter=use/exchange", w+x-370, y+15, 16, hudSub)

	// 4-quadrant layout with different proportions
	leftW := int32(float32(w) * 0.6) // 60% width for left side
	rightW := w - leftW
	topH := int32(float32(h-50) * 0.65) // 65% height for top
	bottomH := (h - 50) - topH
	contentY := y + 50

	// Draw quadrant dividers
	rl.DrawLine(x+leftW, contentY, x+leftW, y+h, hudLine)
	rl.DrawLine(x, contentY+topH, x+leftW, contentY+topH, hudLine) // Only for left side

	// TOP LEFT: Card Grid (larger)
	drawCardGrid(g, x+10, contentY+10, leftW-20, topH-20)

	// BOTTOM LEFT: Exchange Buttons (smaller)
	drawExchangeButtons(g, x+10, contentY+topH+10, leftW-20, bottomH-20)

	// RIGHT SIDE: Cool Visual
	drawInventoryVisual(g, x+leftW+10, contentY+10, rightW-20, h-70)
}

func drawCardGrid(g *Game, x, y, w, h int32) {
	// Navigation instructions
	drawText("COLLECTION - Arrow keys to scroll", x, y, 18, hudText)

	// Collect all cards (monsters, magic monsters, and shop items)
	allCards := []Card{}

	// Add all collected cards
	for _, card := range g.Player.Cards {
		allCards = append(allCards, card)
	}

	// Display cards with scrolling
	if len(allCards) == 0 {
		drawText("No cards collected yet", x+10, y+40, 16, hudSub)
		return
	}

	cardSize := int32(80)
	padding := int32(5)
	cardsPerRow := int((w - 10) / (cardSize + padding))
	cardsPerPage := cardsPerRow * int((h-50)/(cardSize+padding))

	// Calculate which cards to show based on scroll offset
	startIndex := g.InventoryCardScrollOffset
	endIndex := min(startIndex+cardsPerPage, len(allCards))

	if startIndex >= len(allCards) {
		g.InventoryCardScrollOffset = max(0, len(allCards)-cardsPerPage)
		startIndex = g.InventoryCardScrollOffset
		endIndex = min(startIndex+cardsPerPage, len(allCards))
	}

	currentX := x + 5
	currentY := y + 40
	cardIndex := 0

	// Draw visible cards
	for i := startIndex; i < endIndex; i++ {
		card := allCards[i]

		if cardIndex >= cardsPerRow {
			currentY += cardSize + padding
			currentX = x + 5
			cardIndex = 0
		}

		// Check if this card is selected
		isSelected := (i == g.InventorySelectedIndex)

		// Card background color based on type
		var cardColor rl.Color
		var labelText string
		isUsable := false
		switch card.Type {
		case monsterType:
			cardColor = rl.NewColor(180, 80, 60, 180)
			labelText = fmt.Sprintf("STR\n%d", card.Strength)
		case magicMonsterType:
			cardColor = rl.NewColor(120, 110, 140, 180)
			labelText = fmt.Sprintf("MAG\n%d", card.Magic)
		case shopItemType:
			cardColor = rl.NewColor(255, 215, 0, 200) // Brighter for usable items
			isUsable = true
			if card.Strength > 0 && card.Magic > 0 {
				labelText = fmt.Sprintf("+%d/+%d", card.Strength, card.Magic)
			} else if card.Strength > 0 {
				labelText = fmt.Sprintf("+%dS", card.Strength)
			} else {
				labelText = fmt.Sprintf("+%dM", card.Magic)
			}
		case buffType:
			cardColor = rl.NewColor(120, 180, 100, 180) // Normal opacity for non-usable items
			isUsable = false // Buffs are not usable from inventory
			if card.Strength > 0 && card.Magic > 0 {
				labelText = fmt.Sprintf("+%d/+%d", card.Strength, card.Magic)
			} else if card.Strength > 0 {
				labelText = fmt.Sprintf("+%dS", card.Strength)
			} else {
				labelText = fmt.Sprintf("+%dM", card.Magic)
			}
		default:
			cardColor = rl.NewColor(100, 100, 100, 180)
			labelText = "?"
		}

		// Draw selection highlight
		if isSelected {
			rl.DrawRectangle(currentX-3, currentY-3, cardSize+6, cardSize+6, rl.Fade(hudAccent, 0.6))
		}

		rl.DrawRectangle(currentX, currentY, cardSize, cardSize, cardColor)
		rl.DrawRectangleLines(currentX, currentY, cardSize, cardSize, darken(cardColor, 0.5))

		// Additional highlight border for selected card
		if isSelected {
			rl.DrawRectangleLines(currentX-2, currentY-2, cardSize+4, cardSize+4, hudAccent)
		}

		drawText(labelText, currentX+10, currentY+25, 14, hudText)

		// Add "USE" indicator for usable items
		if isUsable {
			drawText("USE", currentX+5, currentY+5, 10, rl.NewColor(255, 255, 255, 200))
		}

		currentX += cardSize + padding
		cardIndex++
	}

	// Scroll indicators
	if startIndex > 0 {
		drawText("▲", x+w-30, y+25, 16, hudAccent) // Up arrow
	}
	if endIndex < len(allCards) {
		drawText("▼", x+w-30, y+h-20, 16, hudAccent) // Down arrow
	}

	// Card count indicator
	drawText(fmt.Sprintf("%d/%d cards", endIndex-startIndex, len(allCards)), x+10, y+h-20, 14, hudSub)
}

func drawExchangeButtons(g *Game, x, y, w, h int32) {
	drawText("EXCHANGE - Arrow keys + Enter", x, y, 18, hudText)

	// Side by side buttons - thinner
	buttonW := (w - 40) / 2 // Two buttons with gaps
	buttonH := int32(40)    // Thinner height
	button1X := x + 10
	button2X := button1X + buttonW + 20
	buttonY := y + 40

	// Calculate totals
	monsterTotal := g.Player.GetMonsterStrengthTotal()
	magicTotal := g.Player.GetMagicMonsterTotal()
	totalCards := len(g.Player.Cards)

	// Strength Exchange Button
	canExchangeStr := monsterTotal >= 7
	btn1Color := hudAccent
	if !canExchangeStr {
		btn1Color = hudSub
	}

	// Highlight selected button (first button after all cards)
	isButton1Selected := (g.InventorySelectedIndex == totalCards)
	if isButton1Selected {
		rl.DrawRectangle(button1X-3, buttonY-3, buttonW+6, buttonH+6, rl.Fade(hudAccent, 0.4))
	}

	rl.DrawRectangle(button1X, buttonY, buttonW, buttonH, rl.Fade(btn1Color, 0.3))
	rl.DrawRectangleLines(button1X, buttonY, buttonW, buttonH, btn1Color)

	btn1Text := fmt.Sprintf("STR (%d/7)", monsterTotal)
	textW := rl.MeasureText(btn1Text, 14)
	drawText(btn1Text, button1X+(buttonW-textW)/2, buttonY+12, 14, btn1Color)

	// Magic Exchange Button
	canExchangeMag := magicTotal >= 7
	btn2Color := hudAccent
	if !canExchangeMag {
		btn2Color = hudSub
	}

	// Highlight selected button (second button after all cards)
	isButton2Selected := (g.InventorySelectedIndex == totalCards+1)
	if isButton2Selected {
		rl.DrawRectangle(button2X-3, buttonY-3, buttonW+6, buttonH+6, rl.Fade(hudAccent, 0.4))
	}

	rl.DrawRectangle(button2X, buttonY, buttonW, buttonH, rl.Fade(btn2Color, 0.3))
	rl.DrawRectangleLines(button2X, buttonY, buttonW, buttonH, btn2Color)

	btn2Text := fmt.Sprintf("MAG (%d/7)", magicTotal)
	textW = rl.MeasureText(btn2Text, 14)
	drawText(btn2Text, button2X+(buttonW-textW)/2, buttonY+12, 14, btn2Color)

	// Store button bounds for click detection
	g.StrengthButtonBounds = rl.NewRectangle(float32(button1X), float32(buttonY), float32(buttonW), float32(buttonH))
	g.MagicButtonBounds = rl.NewRectangle(float32(button2X), float32(buttonY), float32(buttonW), float32(buttonH))
}

func drawInventoryVisual(g *Game, x, y, w, h int32) {
	// Cool mystical visual on the right side
	centerX := x + w/2
	centerY := y + h/2

	// Draw animated mystical orbs
	time := float32(rl.GetTime())

	// Background gradient effect
	for i := 0; i < 5; i++ {
		radius := float32(i*20 + 30)
		alpha := uint8(30 - i*5)
		rl.DrawCircle(centerX, centerY, radius, rl.NewColor(120, 110, 140, alpha))
	}

	// Rotating stat crystals (Health, Gold, Magic, Strength + 1 random)
	for i := 0; i < 4; i++ {
		angle := time*30.2 + float32(i)*90 // Faster rotation, 72 degrees apart for 5 crystals
		angleRad := float64(angle) * math.Pi / 180

		// Slight vertical oscillation
		oscillation := float32(math.Sin(float64(time*2.0+float32(i)*1.5))) * 8 // ±8 pixels

		runeX := centerX + int32(80*math.Cos(angleRad))
		runeY := centerY + int32(80*math.Sin(angleRad)) + int32(oscillation)

		// Color based on stat type
		var crystalColor rl.Color
		var glowIntensity float32 = 0.8

		switch i {
		case 0: // Health crystal - red glow
			intensity := float32(g.Player.Health) / 10.0
			crystalColor = rl.NewColor(220, 80, 60, uint8(255*intensity*glowIntensity))
		case 1: // Gold crystal - golden glow
			intensity := float32(min(g.Player.Gold, 50)) / 50.0 // Cap at 50 for visualization
			crystalColor = rl.NewColor(255, 215, 0, uint8(255*intensity*glowIntensity))
		case 2: // Magic crystal - purple glow
			intensity := float32(g.Player.Magic) / 30.0 // Assume max 10 for now
			crystalColor = rl.NewColor(120, 110, 140, uint8(255*intensity*glowIntensity))
		case 3: // Strength crystal - green glow
			intensity := float32(g.Player.Strength) / 30.0 // Assume max 10 for now
			crystalColor = rl.NewColor(120, 180, 100, uint8(255*intensity*glowIntensity))
		}

		// Draw crystal with glow effect
		rl.DrawCircle(runeX, runeY, 12, rl.Fade(crystalColor, 0.3))      // Outer glow
		rl.DrawCircle(runeX, runeY, 8, crystalColor)                     // Core crystal
		rl.DrawCircleLines(runeX, runeY, 10, lighten(crystalColor, 0.3)) // Inner highlight
	}

	// Central power crystal
	rl.DrawCircle(centerX, centerY, 25, rl.NewColor(255, 215, 0, 100))
	rl.DrawCircleLines(centerX, centerY, 25, hudAccent)
	rl.DrawCircleLines(centerX, centerY, 30, rl.Fade(hudAccent, 0.3))

	// Player stats display with crystal-matching colors
	statsY := y + h - 120
	drawText("HERO STATUS", x+10, statsY-30, 18, hudText)

	// Match crystal colors with appropriate intensities
	healthColor := rl.NewColor(220, 80, 60, uint8(255*0.8))

	goldColor := rl.NewColor(255, 215, 0, uint8(255*0.8))

	magicColor := rl.NewColor(120, 110, 140, uint8(255*0.8))

	strengthColor := rl.NewColor(120, 180, 100, uint8(255*0.8))

	drawText(fmt.Sprintf("Strength: %d", g.Player.Strength), x+10, statsY, 16, strengthColor)
	drawText(fmt.Sprintf("Magic: %d", g.Player.Magic), x+10, statsY+25, 16, magicColor)
	drawText(fmt.Sprintf("Health: %d/10", g.Player.Health), x+10, statsY+50, 16, healthColor)
	drawText(fmt.Sprintf("Gold: %d", g.Player.Gold), x+10, statsY+75, 16, goldColor)
}

func drawShop(g *Game) {
	if !g.ShopActive {
		return
	}

	// Find the current shop data to get shopkeeper type
	var currentShop *ShopType
	cur := g.World.Loops[g.Player.At.Loop].Tiles[g.Player.At.Index]
	for _, link := range cur.Links {
		if isShopTile(g.World, link) {
			shopTile := &g.World.Loops[link.Loop].Tiles[link.Index]
			if shopTile.ShopData != nil {
				currentShop = shopTile.ShopData
				break
			}
		}
	}

	// Larger margins for shop overlay
	margin := int32(100)
	x := margin
	y := margin
	w := int32(screenWidth) - 2*margin
	h := int32(screenHeight) - 2*margin

	// Background overlay
	rl.DrawRectangle(0, 0, screenWidth, screenHeight, rl.NewColor(0, 0, 0, 200))
	rl.DrawRectangle(x-6, y-6, w+12, h+12, rl.NewColor(0, 0, 0, 120))
	rl.DrawRectangle(x, y, w, h, rl.NewColor(60, 45, 30, 250))
	rl.DrawRectangleLines(x, y, w, h, shopGold)

	// Title bar
	rl.DrawRectangle(x, y, w, 60, rl.NewColor(40, 30, 20, 200))
	shopTitle := "MYSTERIOUS SHOP"
	if currentShop != nil {
		shopTitle = strings.ToUpper(currentShop.Name)
	}
	drawText(shopTitle, x+20, y+20, 28, shopGold)
	drawText("ESC or Q to close • Arrow keys to navigate • Enter to buy", w+x-450, y+20, 16, hudSub)

	// Calculate areas: 70% top, 30% bottom
	contentH := h - 60 // Subtract title bar
	topH := int32(float32(contentH) * 0.7)
	bottomH := contentH - topH
	topY := y + 60
	bottomY := topY + topH

	// Draw divider line
	rl.DrawLine(x, bottomY, x+w, bottomY, shopGold)

	// TOP SECTION (70%): Three cards with prices
	drawShopCards(g, x+20, topY+20, w-40, topH-40)

	// BOTTOM SECTION (30%): Exit button and shopkeeper visual
	keeperType := 0
	if currentShop != nil {
		keeperType = currentShop.KeeperType
	}
	drawShopBottom(g, x+20, bottomY+20, w-40, bottomH-40, keeperType)
}

func drawShopCards(g *Game, x, y, w, h int32) {
	cardW := (w - 60) / 3 // 3 cards with gaps
	cardH := h - 40       // Leave space for title

	for i := 0; i < 3; i++ {
		cardX := x + int32(i)*(cardW+30)
		card := g.ShopCards[i]
		price := g.ShopPrices[i]

		// Card background - highlight selected
		cardColor := rl.NewColor(80, 60, 40, 200)
		if i == g.ShopSelected {
			cardColor = rl.NewColor(120, 90, 60, 220)
		}

		rl.DrawRectangle(cardX, y, cardW, cardH, cardColor)
		rl.DrawRectangleLines(cardX, y, cardW, cardH, shopGold)

		// Selected indicator
		if i == g.ShopSelected {
			rl.DrawRectangleLines(cardX-2, y-2, cardW+4, cardH+4, rl.NewColor(255, 215, 0, 255))
		}

		// Card content - show title if available
		cardTitle := fmt.Sprintf("Card %d", i+1)
		if card.Title != "" {
			cardTitle = card.Title
		}
		drawText(cardTitle, cardX+10, y+10, 18, hudText)

		// Card type and stats
		typeY := y + 35
		switch card.Type {
		case shopItemType:
			drawText("MAGIC ITEM", cardX+10, typeY, 16, rl.NewColor(255, 215, 0, 255))
			if card.Strength > 0 && card.Magic > 0 {
				drawText(fmt.Sprintf("STR: +%d, MAG: +%d", card.Strength, card.Magic), cardX+10, typeY+20, 14, hudText)
			} else if card.Strength > 0 {
				drawText(fmt.Sprintf("Strength: +%d", card.Strength), cardX+10, typeY+20, 14, hudText)
			} else if card.Magic > 0 {
				drawText(fmt.Sprintf("Magic: +%d", card.Magic), cardX+10, typeY+20, 14, hudText)
			}
		case monsterType:
			drawText("MONSTER", cardX+10, typeY, 18, rl.NewColor(180, 80, 60, 255))
			drawText(fmt.Sprintf("Strength: %d", card.Strength), cardX+10, typeY+25, 16, hudText)
		case magicMonsterType:
			drawText("MAGIC MONSTER", cardX+10, typeY, 16, rl.NewColor(120, 110, 140, 255))
			drawText(fmt.Sprintf("Magic: %d", card.Magic), cardX+10, typeY+25, 16, hudText)
		case buffType:
			drawText("BLESSING", cardX+10, typeY, 18, rl.NewColor(120, 180, 100, 255))
			if card.Strength > 0 {
				drawText(fmt.Sprintf("Strength: +%d", card.Strength), cardX+10, typeY+25, 16, hudText)
			} else {
				drawText(fmt.Sprintf("Magic: +%d", card.Magic), cardX+10, typeY+25, 16, hudText)
			}
		}

		// Price display
		priceY := y + cardH - 50
		affordable := g.Player.Gold >= price
		priceColor := hudAccent
		if !affordable {
			priceColor = rl.NewColor(180, 80, 60, 255) // Red if can't afford
		}

		rl.DrawRectangle(cardX+5, priceY, cardW-10, 35, rl.NewColor(0, 0, 0, 100))
		drawText(fmt.Sprintf("Price: %d gold", price), cardX+10, priceY+8, 18, priceColor)
	}
}

func drawShopBottom(g *Game, x, y, w, h int32, keeperType int) {
	// Exit button on the left
	buttonW := int32(150)
	buttonH := int32(50)
	buttonX := x + 20
	buttonY := y + (h-buttonH)/2

	// Highlight if selected
	buttonColor := hudAccent
	if g.ShopSelected == 3 {
		buttonColor = rl.NewColor(255, 215, 0, 255)                                    // Brighter when selected
		rl.DrawRectangleLines(buttonX-2, buttonY-2, buttonW+4, buttonH+4, buttonColor) // Extra border
	}

	rl.DrawRectangle(buttonX, buttonY, buttonW, buttonH, rl.Fade(buttonColor, 0.3))
	rl.DrawRectangleLines(buttonX, buttonY, buttonW, buttonH, buttonColor)

	exitText := "EXIT SHOP"
	textW := rl.MeasureText(exitText, 20)
	drawText(exitText, buttonX+(buttonW-textW)/2, buttonY+15, 20, buttonColor)

	// Store button bounds for click detection
	g.ExitButtonBounds = rl.NewRectangle(float32(buttonX), float32(buttonY), float32(buttonW), float32(buttonH))

	// Cool visual on the right side
	visualX := x + buttonW + 60
	visualW := w - buttonW - 80
	visualH := h

	// Draw animated shop keeper or mystical visual
	centerX := visualX + visualW/2
	centerY := y + visualH/2

	// Draw a larger pile of gold coins with shopkeeper
	time := float32(rl.GetTime())

	// Shopkeeper (moved right and made much more visible)
	keeperX := x + 800 // A little more to the right

	// Much wider gold pile (5 times larger, extending far to the right)
	pileStartX := centerX - 100
	pileEndX := centerX + 400 // 5 times wider to the right

	for layer := 0; layer < 8; layer++ { // More layers
		layerY := centerY + int32(layer*5) - 5
		// Calculate coins needed to fill the width
		coinSpacing := int32(12)
		pileWidth := pileEndX - pileStartX
		coinsInLayer := int(pileWidth/coinSpacing) - layer*2 // Taper as we go up
		if coinsInLayer < 3 {
			coinsInLayer = 3
		}

		for coin := 0; coin < coinsInLayer; coin++ {
			coinX := pileStartX + int32(int32(layer)*coinSpacing) + int32(int32(coin)*coinSpacing)

			// Add slight animation wobble
			wobble := float32(math.Sin(float64(time*1.5+float32(layer)+float32(coin)))) * 1.0
			finalX := coinX + int32(wobble)

			// Coin size decreases as we go up
			radius := float32(12 - layer)
			if radius < 3 {
				radius = 3
			}

			// Draw coin shadow
			rl.DrawCircle(finalX+2, layerY+2, radius, rl.NewColor(0, 0, 0, 40))

			// Draw coin
			rl.DrawCircle(finalX, layerY, radius, shopGold)
			rl.DrawCircleLines(finalX, layerY, radius, darken(shopGold, 0.6))

			// Inner shine
			if radius > 5 {
				rl.DrawCircle(finalX-1, layerY-1, radius*0.25, lighten(shopGold, 0.4))
			}
		}
	}

	// Enhanced sparkle effects across the wider pile
	sparkleCount := int(time*12) % 30
	for i := 0; i < 8; i++ { // More sparkles
		if (sparkleCount+i*4)%30 < 8 {
			sparkleX := pileStartX + int32(rand.Intn(int(pileEndX-pileStartX)))
			sparkleY := centerY + int32(rand.Intn(40)) - 5
			rl.DrawCircle(sparkleX, sparkleY, 2, rl.Fade(rl.White, 0.9))
		}
	}

	// Treasure chest behind the pile (larger)
	chestX := centerX + 40
	chestY := centerY + 15
	chestW := int32(60)
	chestH := int32(30)

	// Chest body
	rl.DrawRectangle(chestX, chestY, chestW, chestH, rl.NewColor(101, 67, 33, 255)) // Brown wood
	rl.DrawRectangleLines(chestX, chestY, chestW, chestH, rl.NewColor(70, 45, 20, 255))

	// Chest metal bands
	rl.DrawRectangle(chestX, chestY+10, chestW, 4, rl.NewColor(120, 120, 120, 255))
	rl.DrawRectangle(chestX, chestY+20, chestW, 4, rl.NewColor(120, 120, 120, 255))

	// Chest lock
	rl.DrawCircle(chestX+chestW-10, chestY+chestH/2, 5, rl.NewColor(150, 150, 150, 255))
	keeperY := centerY - 30

	// Check if we're in post-purchase animation (2 seconds)
	justPurchased := time-g.LastPurchaseTime < 2.0

	// Draw different shopkeeper types based on keeperType (0-3)
	switch keeperType {
	case 0: // Blue Mystic Shopkeeper
		// Shopkeeper body (150% larger with bright, contrasting colors)
		rl.DrawCircle(keeperX, keeperY+45, 27, rl.NewColor(60, 120, 180, 255))  // Bright blue body
		rl.DrawCircle(keeperX, keeperY+18, 15, rl.NewColor(220, 180, 140, 255)) // Warm skin tone head
		rl.DrawCircle(keeperX, keeperY+10, 18, rl.NewColor(140, 60, 160, 200))  // Purple hood

		// Animated eyes and mouth (scaled 150%)
		if justPurchased {
			// Happy eyes (crescents) - larger
			rl.DrawRectangle(keeperX-6, keeperY+15, 3, 2, rl.NewColor(255, 215, 0, 255))
			rl.DrawRectangle(keeperX+3, keeperY+15, 3, 2, rl.NewColor(255, 215, 0, 255))
			// Smiling mouth - larger
			rl.DrawCircle(keeperX, keeperY+21, 5, rl.NewColor(220, 180, 140, 255))
			rl.DrawRectangle(keeperX-5, keeperY+18, 10, 3, rl.NewColor(140, 60, 160, 200)) // Cover top half for smile
		} else {
			// Normal glowing eyes - larger
			rl.DrawCircle(keeperX-5, keeperY+15, 2, rl.NewColor(255, 215, 0, 255))
			rl.DrawCircle(keeperX+5, keeperY+15, 2, rl.NewColor(255, 215, 0, 255))
		}

		// Animated waving arm when just purchased (scaled 150%)
		if justPurchased {
			waveOffset := float32(math.Sin(float64(time*8.0))) * 15 // Larger waving motion
			// Waving arm - thicker and longer
			rl.DrawLineEx(rl.NewVector2(float32(keeperX+22), float32(keeperY+30)),
				rl.NewVector2(float32(keeperX+37)+waveOffset, float32(keeperY+7)), 3, rl.NewColor(220, 180, 140, 255))
			// Hand - larger
			rl.DrawCircle(keeperX+37+int32(waveOffset), keeperY+7, 5, rl.NewColor(220, 180, 140, 255))
		} else {
			// Static arm with staff - larger
			rl.DrawLineEx(rl.NewVector2(float32(keeperX-22), float32(keeperY+45)),
				rl.NewVector2(float32(keeperX-15), float32(keeperY-7)), 3, rl.NewColor(139, 90, 43, 255))
			rl.DrawCircle(keeperX-15, keeperY-7, 6, rl.NewColor(255, 100, 255, 255)) // Bright magical orb
			// Add sparkle to the orb
			if int(time*6)%12 < 3 {
				rl.DrawCircle(keeperX-15, keeperY-7, 8, rl.Fade(rl.White, 0.4))
			}
		}

	case 1: // Green Ogre Shopkeeper
		// Large ogre-like body with green skin
		rl.DrawCircle(keeperX, keeperY+50, 32, rl.NewColor(80, 120, 60, 255)) // Larger green body
		rl.DrawCircle(keeperX, keeperY+15, 18, rl.NewColor(90, 130, 70, 255)) // Green ogre head (bigger)

		// Ogre features - big nose and ears
		rl.DrawCircle(keeperX, keeperY+20, 4, rl.NewColor(70, 110, 50, 255))    // Large nose
		rl.DrawCircle(keeperX-12, keeperY+12, 6, rl.NewColor(90, 130, 70, 255)) // Left ear
		rl.DrawCircle(keeperX+12, keeperY+12, 6, rl.NewColor(90, 130, 70, 255)) // Right ear

		// Animated eyes and mouth
		if justPurchased {
			// Happy ogre eyes (crescents)
			rl.DrawRectangle(keeperX-8, keeperY+12, 4, 3, rl.NewColor(255, 215, 0, 255))
			rl.DrawRectangle(keeperX+4, keeperY+12, 4, 3, rl.NewColor(255, 215, 0, 255))
			// Big ogre grin
			rl.DrawCircle(keeperX, keeperY+24, 6, rl.NewColor(90, 130, 70, 255))
			rl.DrawRectangle(keeperX-6, keeperY+20, 12, 4, rl.NewColor(90, 130, 70, 255)) // Cover top for smile
		} else {
			// Normal ogre eyes - small and beady
			rl.DrawCircle(keeperX-6, keeperY+12, 2, rl.NewColor(255, 0, 0, 255))
			rl.DrawCircle(keeperX+6, keeperY+12, 2, rl.NewColor(255, 0, 0, 255))
		}

		// Animated waving arm when just purchased (ogre-like)
		if justPurchased {
			waveOffset := float32(math.Sin(float64(time*5.0))) * 20 // Slow, heavy ogre waving
			rl.DrawLineEx(rl.NewVector2(float32(keeperX+28), float32(keeperY+35)),
				rl.NewVector2(float32(keeperX+45)+waveOffset, float32(keeperY+8)), 5, rl.NewColor(90, 130, 70, 255))
			rl.DrawCircle(keeperX+45+int32(waveOffset), keeperY+8, 7, rl.NewColor(90, 130, 70, 255)) // Big ogre hand
		} else {
			// Static arm with club
			rl.DrawLineEx(rl.NewVector2(float32(keeperX-25), float32(keeperY+50)),
				rl.NewVector2(float32(keeperX-15), float32(keeperY-5)), 4, rl.NewColor(101, 67, 33, 255))
			rl.DrawRectangle(keeperX-20, keeperY-10, 10, 8, rl.NewColor(101, 67, 33, 255)) // Club head
		}

	case 2: // Red Monkey Shopkeeper
		// Monkey-like body with reddish-brown fur
		rl.DrawCircle(keeperX, keeperY+45, 25, rl.NewColor(140, 70, 40, 255)) // Brown monkey body
		rl.DrawCircle(keeperX, keeperY+18, 16, rl.NewColor(160, 90, 60, 255)) // Monkey head

		// Monkey features - long snout and big ears
		rl.DrawCircle(keeperX, keeperY+24, 8, rl.NewColor(180, 120, 90, 255))   // Long monkey snout
		rl.DrawCircle(keeperX, keeperY+26, 3, rl.NewColor(120, 60, 30, 255))    // Nose on snout
		rl.DrawCircle(keeperX-14, keeperY+10, 8, rl.NewColor(160, 90, 60, 255)) // Left ear
		rl.DrawCircle(keeperX+14, keeperY+10, 8, rl.NewColor(160, 90, 60, 255)) // Right ear

		// Monkey tail curling around
		tailWave := float32(math.Sin(float64(time*3.0))) * 5
		rl.DrawLineEx(rl.NewVector2(float32(keeperX-30), float32(keeperY+50)),
			rl.NewVector2(float32(keeperX-40)+tailWave, float32(keeperY+20)), 3, rl.NewColor(140, 70, 40, 255))

		// Animated eyes and mouth
		if justPurchased {
			// Happy monkey eyes (big and round)
			rl.DrawCircle(keeperX-6, keeperY+15, 4, rl.NewColor(255, 255, 255, 255))
			rl.DrawCircle(keeperX+6, keeperY+15, 4, rl.NewColor(255, 255, 255, 255))
			rl.DrawCircle(keeperX-6, keeperY+14, 2, rl.NewColor(0, 0, 0, 255)) // Pupils
			rl.DrawCircle(keeperX+6, keeperY+14, 2, rl.NewColor(0, 0, 0, 255))
			// Big monkey grin
			rl.DrawCircle(keeperX, keeperY+28, 4, rl.NewColor(255, 200, 150, 255))
			rl.DrawRectangle(keeperX-4, keeperY+26, 8, 2, rl.NewColor(160, 90, 60, 255))
		} else {
			// Normal monkey eyes - curious look
			rl.DrawCircle(keeperX-6, keeperY+15, 3, rl.NewColor(255, 255, 255, 255))
			rl.DrawCircle(keeperX+6, keeperY+15, 3, rl.NewColor(255, 255, 255, 255))
			rl.DrawCircle(keeperX-6, keeperY+15, 2, rl.NewColor(0, 0, 0, 255))
			rl.DrawCircle(keeperX+6, keeperY+15, 2, rl.NewColor(0, 0, 0, 255))
		}

		// Animated waving arm when just purchased (monkey-like)
		if justPurchased {
			waveOffset := float32(math.Sin(float64(time*12.0))) * 25 // Fast, energetic monkey waving
			rl.DrawLineEx(rl.NewVector2(float32(keeperX+20), float32(keeperY+30)),
				rl.NewVector2(float32(keeperX+40)+waveOffset, float32(keeperY)), 3, rl.NewColor(160, 90, 60, 255))
			rl.DrawCircle(keeperX+40+int32(waveOffset), keeperY, 5, rl.NewColor(160, 90, 60, 255))
		} else {
			// Static arm scratching head
			rl.DrawLineEx(rl.NewVector2(float32(keeperX-18), float32(keeperY+30)),
				rl.NewVector2(float32(keeperX-12), float32(keeperY+8)), 3, rl.NewColor(160, 90, 60, 255))
			rl.DrawCircle(keeperX-12, keeperY+8, 4, rl.NewColor(160, 90, 60, 255))
		}

	case 3: // Pink Pig Shopkeeper
		// Pig-like body with pink skin
		rl.DrawCircle(keeperX, keeperY+48, 30, rl.NewColor(255, 180, 180, 255)) // Round pink pig body
		rl.DrawCircle(keeperX, keeperY+20, 18, rl.NewColor(255, 200, 200, 255)) // Pink pig head

		// Pig features - snout and ears
		rl.DrawCircle(keeperX, keeperY+25, 10, rl.NewColor(255, 160, 160, 255))  // Big pig snout
		rl.DrawCircle(keeperX-3, keeperY+24, 2, rl.NewColor(200, 120, 120, 255)) // Left nostril
		rl.DrawCircle(keeperX+3, keeperY+24, 2, rl.NewColor(200, 120, 120, 255)) // Right nostril
		rl.DrawCircle(keeperX-12, keeperY+8, 5, rl.NewColor(255, 200, 200, 255)) // Left pig ear
		rl.DrawCircle(keeperX+12, keeperY+8, 5, rl.NewColor(255, 200, 200, 255)) // Right pig ear

		// Curly pig tail
		tailCurl := float32(math.Sin(float64(time*4.0))) * 3
		rl.DrawLineEx(rl.NewVector2(float32(keeperX-35), float32(keeperY+45)),
			rl.NewVector2(float32(keeperX-42), float32(keeperY+35)+tailCurl), 3, rl.NewColor(255, 180, 180, 255))
		rl.DrawLineEx(rl.NewVector2(float32(keeperX-42), float32(keeperY+35)+tailCurl),
			rl.NewVector2(float32(keeperX-38), float32(keeperY+28)+tailCurl), 3, rl.NewColor(255, 180, 180, 255))

		// Animated eyes and mouth
		if justPurchased {
			// Happy pig eyes (small and beady)
			rl.DrawCircle(keeperX-6, keeperY+12, 3, rl.NewColor(0, 0, 0, 255))
			rl.DrawCircle(keeperX+6, keeperY+12, 3, rl.NewColor(0, 0, 0, 255))
			rl.DrawCircle(keeperX-7, keeperY+11, 1, rl.NewColor(255, 255, 255, 255)) // Eye shine
			rl.DrawCircle(keeperX+7, keeperY+11, 1, rl.NewColor(255, 255, 255, 255))
			// Happy pig smile
			rl.DrawCircle(keeperX, keeperY+30, 3, rl.NewColor(200, 120, 120, 255))
		} else {
			// Normal pig eyes - small and content
			rl.DrawCircle(keeperX-6, keeperY+12, 2, rl.NewColor(0, 0, 0, 255))
			rl.DrawCircle(keeperX+6, keeperY+12, 2, rl.NewColor(0, 0, 0, 255))
		}

		// Animated waving arm when just purchased (pig-like)
		if justPurchased {
			waveOffset := float32(math.Sin(float64(time*4.0))) * 15 // Gentle pig waving
			rl.DrawLineEx(rl.NewVector2(float32(keeperX+25), float32(keeperY+35)),
				rl.NewVector2(float32(keeperX+40)+waveOffset, float32(keeperY+10)), 4, rl.NewColor(255, 200, 200, 255))
			rl.DrawCircle(keeperX+40+int32(waveOffset), keeperY+10, 6, rl.NewColor(255, 200, 200, 255)) // Chubby pig hand
		} else {
			// Static arm holding coin pouch
			rl.DrawLineEx(rl.NewVector2(float32(keeperX-20), float32(keeperY+40)),
				rl.NewVector2(float32(keeperX-10), float32(keeperY+20)), 4, rl.NewColor(255, 200, 200, 255))
			rl.DrawCircle(keeperX-10, keeperY+20, 5, rl.NewColor(160, 120, 40, 255)) // Gold coin pouch
			rl.DrawCircle(keeperX-10, keeperY+18, 2, rl.NewColor(255, 215, 0, 255))  // Golden coin peek
		}

	case 4: // Blue Dolphin Shopkeeper
		// Dolphin-like body with blue-gray skin
		rl.DrawCircle(keeperX, keeperY+45, 28, rl.NewColor(120, 160, 200, 255)) // Blue dolphin body
		rl.DrawCircle(keeperX, keeperY+18, 16, rl.NewColor(140, 180, 220, 255)) // Dolphin head

		// Dolphin features - elongated snout and dorsal fin
		rl.DrawRectangle(keeperX-3, keeperY+25, 6, 12, rl.NewColor(140, 180, 220, 255)) // Long dolphin beak
		rl.DrawCircle(keeperX, keeperY+31, 3, rl.NewColor(140, 180, 220, 255))          // Round beak tip

		// Dorsal fin on back
		finSway := float32(math.Sin(float64(time*2.0))) * 2
		rl.DrawCircle(keeperX+int32(finSway), keeperY-5, 8, rl.NewColor(100, 140, 180, 255))

		// Tail fin (flippers)
		tailSway := float32(math.Sin(float64(time*1.5))) * 8
		rl.DrawCircle(keeperX-40+int32(tailSway), keeperY+50, 12, rl.NewColor(120, 160, 200, 255)) // Tail fluke
		rl.DrawCircle(keeperX-38+int32(tailSway), keeperY+45, 8, rl.NewColor(120, 160, 200, 255))  // Tail connection

		// Side flippers
		flipperSway := float32(math.Sin(float64(time*3.0))) * 5
		rl.DrawCircle(keeperX-20, keeperY+35+int32(flipperSway), 10, rl.NewColor(120, 160, 200, 255)) // Left flipper
		rl.DrawCircle(keeperX+20, keeperY+35-int32(flipperSway), 10, rl.NewColor(120, 160, 200, 255)) // Right flipper

		// Animated eyes and mouth
		if justPurchased {
			// Happy dolphin eyes (large and friendly)
			rl.DrawCircle(keeperX-8, keeperY+12, 4, rl.NewColor(255, 255, 255, 255))
			rl.DrawCircle(keeperX+8, keeperY+12, 4, rl.NewColor(255, 255, 255, 255))
			rl.DrawCircle(keeperX-8, keeperY+10, 2, rl.NewColor(0, 0, 0, 255))      // Left pupil
			rl.DrawCircle(keeperX+8, keeperY+10, 2, rl.NewColor(0, 0, 0, 255))      // Right pupil
			rl.DrawCircle(keeperX-9, keeperY+9, 1, rl.NewColor(255, 255, 255, 255)) // Left eye shine
			rl.DrawCircle(keeperX+9, keeperY+9, 1, rl.NewColor(255, 255, 255, 255)) // Right eye shine
			// Happy dolphin smile on beak
			rl.DrawRectangle(keeperX-4, keeperY+28, 8, 2, rl.NewColor(100, 140, 180, 255))
		} else {
			// Normal dolphin eyes - intelligent look
			rl.DrawCircle(keeperX-8, keeperY+12, 3, rl.NewColor(255, 255, 255, 255))
			rl.DrawCircle(keeperX+8, keeperY+12, 3, rl.NewColor(255, 255, 255, 255))
			rl.DrawCircle(keeperX-8, keeperY+12, 2, rl.NewColor(0, 0, 0, 255))
			rl.DrawCircle(keeperX+8, keeperY+12, 2, rl.NewColor(0, 0, 0, 255))
		}

		// Animated flipper waving when just purchased (dolphin-like)
		if justPurchased {
			waveOffset := float32(math.Sin(float64(time*8.0))) * 20 // Smooth dolphin flipper waving
			// Right flipper waves up and down
			rl.DrawCircle(keeperX+25, keeperY+20+int32(waveOffset), 12, rl.NewColor(120, 160, 200, 255))
			// Water splash effects around the flipper
			if int(time*8)%16 < 8 {
				rl.DrawCircle(keeperX+30, keeperY+15+int32(waveOffset), 3, rl.Fade(rl.SkyBlue, 0.6))
				rl.DrawCircle(keeperX+20, keeperY+25+int32(waveOffset), 2, rl.Fade(rl.SkyBlue, 0.4))
			}
		} else {
			// Static pose - dolphin floating
			floatBob := float32(math.Sin(float64(time*1.0))) * 3
			// Gentle floating motion already applied to whole body via position offsets
			rl.DrawCircle(keeperX-8, keeperY+5+int32(floatBob), 3, rl.Fade(rl.SkyBlue, 0.3)) // Blowhole bubble
		}
	}
	// Instructions
	instrY := y + visualH - 30
	drawText("Use 1/2/3 or arrow keys to select • Enter to purchase", visualX, instrY, 14, hudSub)
}
func drawMenuBar(g *Game) {
	y := int32(screenHeight - menuHeight)

	// panel
	rl.DrawRectangle(0, y, screenWidth, menuHeight, hudBG)
	drawHairlineX(0, screenWidth, y) // top hairline

	// layout: | left (turn/steps/dice) | mid (stats) | right (cards) |
	padding := int32(18)
	colGap := int32(22)
	leftX := padding
	midX := int32(screenWidth)/2 - 220
	rightX := int32(screenWidth) - 520

	// --- LEFT: turn / steps / phase or hint ---
	drawText(fmt.Sprintf("Turn %d", g.Turn), leftX, y+12, 26, hudText)

	leftX = drawStat(leftX, y+44, "Steps", fmt.Sprintf("%d", g.StepsRemaining)) + colGap
	if g.LastRoll > 0 {
		// dice badge
		dtxt := fmt.Sprintf("Dice %d", g.LastRoll)
		drawText(dtxt, leftX, y+44, 18, hudSub)
		drawText(fmt.Sprintf("%d", g.LastRoll), leftX, y+64, 28, hudAccent)
	}
	// phase hint
	hint := ""
	switch g.Phase {
	case PhaseIdle:
		hint = "R to roll • Q for inventory"
	case PhaseTargetSelect:
		hint = "←/→ pick • Enter confirm • Esc cancel"
	case PhaseAnimating:
		hint = "Resolving…"
	}
	if hint != "" {
		drawText(hint, padding, y+menuHeight-28, 20, hudSub)
	}

	// separator
	drawHairlineY(int32(screenWidth)/2-260, y+10, y+menuHeight-10)

	statX := midX

	// Strength
	statX = drawStat(statX, y+10, "STR", fmt.Sprintf("%d", g.Player.Strength))

	// Magic (NEW)
	statX = drawStat(statX, y+10, "MAG", fmt.Sprintf("%d", g.Player.Magic))

	// Gold
	statX = drawStat(statX, y+10, "GOLD", fmt.Sprintf("%d", g.Player.Gold))

	// HP block (unchanged)
	lbl := "HP"
	if g.Player.Health <= 3 {
		lbl = "HP (low)"
	}
	drawText(lbl, statX, y+10, 18, hudSub)
	drawHPBar(statX, y+32, 200, 18, g.Player.Health, 10)
	drawText(fmt.Sprintf("%d/10", g.Player.Health), statX+210, y+28, 22, hudText)

	// separator
	drawHairlineY(int32(screenWidth)/2+240, y+10, y+menuHeight-10)

	// --- RIGHT: collected cards (disabled - now shown in inventory) ---
	drawText("Cards: Use Q to view inventory", rightX, y+12, 18, hudSub)
	// drawCardStrip(rightX, y+36, 500, g.Player.Cards) // Disabled
}

// --- HUD palette - Talisman-inspired ---
var hudBG = rl.NewColor(40, 35, 30, 240)      // Rich dark brown, parchment-like
var hudLine = rl.NewColor(160, 140, 100, 80)  // Warm bronze hairlines
var hudText = rl.NewColor(220, 210, 190, 255) // Warm ivory text
var hudSub = rl.NewColor(180, 160, 130, 255)  // Muted bronze secondary text
var hudAccent = rl.NewColor(255, 215, 0, 255) // Rich gold accent (dice, highlights)
var hpOK = rl.NewColor(120, 180, 100, 255)    // Forest green (matching loop1)
var hpWarn = rl.NewColor(220, 80, 60, 255)    // Warning red

func drawHairlineY(x int32, y1, y2 int32) {
	rl.DrawLine(x, y1, x, y2, hudLine)
}

func drawHairlineX(x1, x2, y int32) {
	rl.DrawLine(x1, y, x2, y, hudLine)
}

func drawStat(x, y int32, label string, value string) int32 {
	lh := int32(18)
	drawText(label, x, y, 18, hudSub)
	w := int32(rl.MeasureText(value, 28))
	drawText(value, x, y+lh+2, 28, hudText)
	return x + maxI32(w+8, 90) // column min width
}

func drawHPBar(x, y, w, h int32, hp, maxHP int) {
	// bar bg
	rl.DrawRectangle(x, y, w, h, rl.NewColor(255, 255, 255, 10))
	rl.DrawRectangleLines(x, y, w, h, rl.NewColor(255, 255, 255, 25))
	// fill
	ratio := float32(hp) / float32(maxHP)
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	fill := int32(float32(w) * ratio)
	col := hpOK
	if ratio <= 0.35 {
		col = hpWarn
	}
	rl.DrawRectangle(x, y, fill, h, col)
}

func drawCardStrip(x, y int32, maxW int32, cards []Card) {
	cursor := x
	const chipH int32 = 26
	for i, c := range cards {
		title := ""
		if c.Type == monsterType {
			title = fmt.Sprintf("Monster %d", c.Strength)
		} else if c.Type == buffType {
			title = fmt.Sprintf("Buff +%d", c.Strength)
		} else if c.Type == magicMonsterType {
			title = fmt.Sprintf("Magic Monster %d", c.Magic)
		}
		w := int32(rl.MeasureText(title, 18)) + 20
		if cursor+w > x+maxW {
			// truncated indicator
			if i < len(cards) {
				drawText("…", cursor+2, y+3, 22, hudSub)
			}
			break
		}
		// chip with Talisman-inspired colors
		bg := rl.NewColor(160, 140, 100, 40) // Default bronze
		if c.Type == buffType {
			bg = rl.NewColor(120, 180, 100, 60) // Forest green for buffs
		} else if c.Type == monsterType {
			bg = rl.NewColor(180, 80, 60, 60) // Danger red for monsters
		} else if c.Type == magicMonsterType {
			bg = rl.NewColor(120, 110, 140, 60) // Mystical purple for magic monsters
		}
		rl.DrawRectangleRounded(rl.NewRectangle(float32(cursor), float32(y), float32(w), float32(chipH)), 0.35, 8, bg)
		rl.DrawRectangleRoundedLines(rl.NewRectangle(float32(cursor), float32(y), float32(w), float32(chipH)), 0.35, 8, rl.NewColor(255, 255, 255, 32))
		drawText(title, cursor+10, y+4, 18, hudText)
		cursor += w + 6
	}
}

func maxI32(a, b int32) int32 {
	if a > b {
		return a
	}
	return b
}

func BuildWorldCountsRandom(counts []int) World {
	// even counts
	for i, n := range counts {
		if n%2 != 0 {
			counts[i] = n + 1
		}
	}

	g := Grid{Origin: rl.NewVector2(screenWidth/2, screenHeight/2), Cell: tileSize}

	// dims
	cols := make([]int, len(counts))
	rows := make([]int, len(counts))
	for i, n := range counts {
		c, r, ok := rectDimsForPerimeter(n)
		if !ok {
			panic("cannot factor even tile count")
		}
		cols[i], rows[i] = c, r
	}

	// placement specs + which earlier loop each loop is anchored to
	specs := make([]rectSpec, len(counts))
	parent := make([]int, len(counts)) // parent[i] = anchor loop index; -1 for the first
	occ := map[cell]bool{}

	// seed: first loop at (0,0)
	specs[0] = rectSpec{gx: 0, gy: 0, cols: cols[0], rows: rows[0]}
	parent[0] = -1
	markRectPerimeter(occ, specs[0].gx, specs[0].gy, specs[0].cols, specs[0].rows)

	// place the rest; sometimes attach to a random previous loop instead of (i-1)
	for i := 1; i < len(counts); i++ {
		// candidate anchors in priority order
		anchors := []int{i - 1}
		if i >= 2 && rand.Float32() < crossLinkChance {
			anchors = append([]int{rand.Intn(i - 1)}, anchors...) // pick from 0..i-2
		}

		placed := false
		for _, aidx := range anchors {
			A := specs[aidx]
			for _, d := range shuffledDirs() {
				pr := placeAround(A, cols[i], rows[i], d)

				// corridor must be free; B perimeter must be free
				if anyOccupied(occ, pr.bridge1, pr.bridge2) {
					continue
				}
				if wouldCollideRectPerimeter(occ, pr.specB.gx, pr.specB.gy, pr.specB.cols, pr.specB.rows) {
					continue
				}

				// accept
				specs[i] = pr.specB
				parent[i] = aidx
				markRectPerimeter(occ, pr.specB.gx, pr.specB.gy, pr.specB.cols, pr.specB.rows)
				markCells(occ, pr.bridge1, pr.bridge2)
				placed = true
				break
			}
			if placed {
				break
			}
		}

		if !placed {
			// fallback: attach to previous on the right
			pr := placeAround(specs[i-1], cols[i], rows[i], Right)
			specs[i] = pr.specB
			parent[i] = i - 1
			markRectPerimeter(occ, pr.specB.gx, pr.specB.gy, pr.specB.cols, pr.specB.rows)
			markCells(occ, pr.bridge1, pr.bridge2)
		}
	}

	// build loops
	world := World{Loops: make([]Loop, 0, len(counts)+(len(counts)-1))}
	for i := range counts {
		world.Loops = append(world.Loops, buildRectPerimeterLoopAtGrid(
			g, specs[i].gx, specs[i].gy, specs[i].cols, specs[i].rows, i))
	}
	finalizeLoopIndices(&world)

	// connect each loop i to its chosen anchor parent[i] with a 2-tile bridge
	for i := 1; i < len(specs); i++ {
		j := parent[i]
		if j < 0 {
			continue
		}

		// recompute the exact corridor cells and facing mids
		dir := inferDir(specs[j], specs[i])
		pr := placeAround(specs[j], specs[i].cols, specs[i].rows, dir)

		ai := findTileIndexAtCell(g, world.Loops[j], pr.aMidX, pr.aMidY)
		bi := findTileIndexAtCell(g, world.Loops[i], pr.bMidX, pr.bMidY)

		bridgeLoop := makeBridge(g, pr.bridge1.X, pr.bridge1.Y, pr.bridge2.X, pr.bridge2.Y)
		world.Loops = append(world.Loops, bridgeLoop)
		bridgeLoopIdx := len(world.Loops) - 1
		finalizeLoopIndices(&world)

		// one-to-one links
		world.Loops[bridgeLoopIdx].Tiles[0].Links = []TileID{{Loop: j, Index: ai}}
		world.Loops[bridgeLoopIdx].Tiles[1].Links = []TileID{{Loop: i, Index: bi}}
		world.Loops[j].Tiles[ai].Links = []TileID{{Loop: bridgeLoopIdx, Index: 0}}
		world.Loops[i].Tiles[bi].Links = []TileID{{Loop: bridgeLoopIdx, Index: 1}}
	}
	addExtraCycleBridges(g, specs, &world, occ)

	spawnShops(g, specs, &world, occ)
	return world
}

// infer which cardinal placement produced b from a
func inferDir(a, b rectSpec) Dir {
	_, ayR := sideMidCell(a.gx, a.gy, a.cols, a.rows, "right")
	if b.gx == a.gx+a.cols+2 && b.gy == ayR-b.rows/2 {
		return Right
	}
	_, ayL := sideMidCell(a.gx, a.gy, a.cols, a.rows, "left")
	if b.gx == a.gx-2-b.cols && b.gy == ayL-b.rows/2 {
		return Left
	}
	axB, _ := sideMidCell(a.gx, a.gy, a.cols, a.rows, "bottom")
	if b.gx == axB-b.cols/2 && b.gy == a.gy+a.rows+2 {
		return Down
	}
	axT, _ := sideMidCell(a.gx, a.gy, a.cols, a.rows, "top")
	if b.gx == axT-b.cols/2 && b.gy == a.gy-2-b.rows {
		return Up
	}
	return Right // fallback
}
func spawnShops(g Grid, specs []rectSpec, world *World, occ map[cell]bool) {
	shopCounter := 0
	for li := range specs {
		// only consider the original loops (skip bridge loops added later)
		if li >= len(specs) {
			break
		}

		loop := &world.Loops[li]
		sp := specs[li]

		for ti := range loop.Tiles {
			pt := &loop.Tiles[ti]
			if pt.Bridge { // skip bridge endpoints
				continue
			}
			if rand.Float32() >= shopChance {
				continue
			}

			cx, cy := g.CellOf(pt.Pos)
			dx, dy, ok := outwardDirForCellOnRect(sp, cx, cy)
			if !ok {
				continue // safety: tile not recognized as perimeter
			}
			shopCell := cell{cx + dx, cy + dy}
			if anyOccupied(occ, shopCell) {
				continue // don't overlap existing things
			}

			// create the 1-tile shop loop with persistent inventory
			shop := makeShop(g, shopCell.X, shopCell.Y)

			// Initialize shop with unique inventory
			keeperType := rand.Intn(5) // Random shopkeeper type (0-4)
			var shopName string
			switch keeperType {
			case 0:
				shopNames := []string{"Mystic Emporium", "Arcane Artifacts", "Crystal Cave", "Wizard's Workshop", "Magic Mirror Shop"}
				shopName = shopNames[rand.Intn(len(shopNames))]
			case 1:
				shopNames := []string{"Ogre's Armory", "Beast & Bone", "Iron Fist Trading", "Brutal Bargains", "Stone Club Store"}
				shopName = shopNames[rand.Intn(len(shopNames))]
			case 2:
				shopNames := []string{"Banana Bazaar", "Jungle Goods", "Monkey Business", "Vine & Vine", "Treetop Treasures"}
				shopName = shopNames[rand.Intn(len(shopNames))]
			case 3:
				shopNames := []string{"Pig & Whistle", "Truffle Traders", "Muddy Boots", "Farm Fresh Finds", "Snort & Shop"}
				shopName = shopNames[rand.Intn(len(shopNames))]
			case 4:
				shopNames := []string{"Ocean's Bounty", "Tidal Treasures", "Deep Sea Depot", "Whale Song Shop", "Coral Curiosities"}
				shopName = shopNames[rand.Intn(len(shopNames))]
			}
			shopData := &ShopType{
				Name:       shopName,
				Discovered: false,
				KeeperType: keeperType,
			}
			// Generate 3 random shop cards
			for i := 0; i < 3; i++ {
				shopData.Cards[i] = RandShopCard(shopData.KeeperType)
				// Calculate price based on card attributes
				baseCost := 0
				if shopData.Cards[i].Strength > 0 && shopData.Cards[i].Magic > 0 {
					baseCost = (shopData.Cards[i].Strength + shopData.Cards[i].Magic) * 4 // Mixed items cost more
				} else {
					baseCost = max(shopData.Cards[i].Strength, shopData.Cards[i].Magic) * 3
				}
				shopData.Prices[i] = baseCost + rand.Intn(5) + 1 // Add 1-5 gold randomness
			}

			shop.Tiles[0].ShopData = shopData
			world.Loops = append(world.Loops, shop)
			shopIdx := len(world.Loops) - 1
			finalizeLoopIndices(world)

			// link shop <-> perimeter tile
			world.Loops[shopIdx].Tiles[0].Links = []TileID{{Loop: li, Index: ti}}
			pt.Links = append(pt.Links, TileID{Loop: shopIdx, Index: 0})

			// reserve the cell so nothing else spawns here
			markCells(occ, shopCell)
			shopCounter++
		}
	}
}

func cardChipColor(t CardType) rl.Color {
	if t == monsterType || t == magicMonsterType {
		return rl.NewColor(235, 214, 186, 255) // light sand
	}
	return rl.NewColor(204, 231, 204, 255) // light green
}

// Try to add extra 2-tile bridges between any two original loops (not bridge loops)
// when they are already placed with the exact 2-cell corridor alignment.
func addExtraCycleBridges(g Grid, specs []rectSpec, world *World, occ map[cell]bool) {
	n := len(specs) // only original rectangles
DIRS:
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			// If they are placed with a 2-cell gap on any side, placeAround(spec[i], specs[j], d)
			// will reproduce specs[j] exactly. Use that to detect adjacency.
			for _, d := range []Dir{Right, Left, Up, Down} {
				pr := placeAround(specs[i], specs[j].cols, specs[j].rows, d)
				// exact match?
				if pr.specB.gx != specs[j].gx || pr.specB.gy != specs[j].gy ||
					pr.specB.cols != specs[j].cols || pr.specB.rows != specs[j].rows {
					continue
				}
				// corridor must be free and not already used by another bridge/shop
				if anyOccupied(occ, pr.bridge1, pr.bridge2) {
					continue
				}
				// mid tiles on each loop
				ai := findTileIndexAtCell(g, world.Loops[i], pr.aMidX, pr.aMidY)
				bi := findTileIndexAtCell(g, world.Loops[j], pr.bMidX, pr.bMidY)
				if ai < 0 || bi < 0 {
					continue
				}

				// build the bridge loop
				bridge := makeBridge(g, pr.bridge1.X, pr.bridge1.Y, pr.bridge2.X, pr.bridge2.Y)
				world.Loops = append(world.Loops, bridge)
				bridgeIdx := len(world.Loops) - 1
				finalizeLoopIndices(world)

				// link ends: i <-> bridge[0], j <-> bridge[1]
				world.Loops[bridgeIdx].Tiles[0].Links = []TileID{{Loop: i, Index: ai}}
				world.Loops[bridgeIdx].Tiles[1].Links = []TileID{{Loop: j, Index: bi}}
				world.Loops[i].Tiles[ai].Links = append(world.Loops[i].Tiles[ai].Links, TileID{Loop: bridgeIdx, Index: 0})
				world.Loops[j].Tiles[bi].Links = append(world.Loops[j].Tiles[bi].Links, TileID{Loop: bridgeIdx, Index: 1})

				// reserve the corridor cells so nothing else uses them
				markCells(occ, pr.bridge1, pr.bridge2)

				// We found/added the closing bridge for (i,j); don't try other dirs for this pair.
				continue DIRS
			}
		}
	}
}
