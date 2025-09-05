package main

import (
	"fmt"
	"math/rand"
	"runtime"

	_ "github.com/gen2brain/raylib-go/raygui"
	_ "github.com/gen2brain/raylib-go/raylib"
	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	screenWidth  = 2560
	screenHeight = 1440
	tileSize     = 44
	fontSize     = 14
	tilePad      = 10 // spacing between tiles around a loop (affects loop radius)
	margin       = 80 // keep loops away from window edges
	loopGap      = 30 // minimum clearance between different loops (circle-to-circle)
	menuHeight   = 120

	crossLinkChance = 0.95
	shopChance      = 0.70
)

func main() {
	runtime.LockOSThread() // <-- must be first on macOS

	rl.InitWindow(int32(screenWidth), int32(screenHeight), "Talisman")
	defer rl.CloseWindow()
	rl.SetTargetFPS(60)

	world := BuildWorldCountsRandom([]int{16, 14, 20, 10, 16, 28, 22, 19})

	finalizeLoopIndices(&world)

	game := Game{
		Player: *NewPlayer(TileID{0, 0}, 3, 5),
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
			cam.Zoom = clamp(cam.Zoom, 0.4, 2.5)
		}
		// Reset zoom with 0 key (optional)
		if rl.IsKeyPressed(rl.KeyZero) {
			cam.Zoom = 1.0
		}
		updateCamera(&cam, playerPos(&world, &game), dt)

		if rl.IsKeyPressed(rl.KeySpace) {
			world = BuildWorldCountsRandom([]int{16, 14, 20, 10, 16, 28, 22, 19})
			// reset player to safe starting position
			game.Player.At = TileID{0, 0}
			game.StepsRemaining = 0
			game.Phase = PhaseIdle
		}
		if game.CardActive {
			goto AFTER_INPUT
		}
		switch game.Phase {
		case PhaseIdle:
			if rl.IsKeyPressed(rl.KeyR) {
				game.Roll()
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
				if len(cur.Links) > 0 {
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

// A few browns we can rotate through for normal loops
var brownPalette = []rl.Color{
	rl.Beige,
}

// Outline for normal loop tiles
var brownOutline = rl.NewColor(60, 42, 33, 255)

func loopColors(i int) (rl.Color, rl.Color) {
	fill := brownPalette[i%len(brownPalette)]
	return fill, brownOutline
}

func drawLoops(w *World, highlights map[TileID]rl.Color) {
	for li, loop := range w.Loops {
		baseFill, baseOutline := loopColors(li)

		for i, t := range loop.Tiles {
			fill, outline := baseFill, baseOutline

			// classify special tiles first
			if t.Bridge {
				fill, outline = rl.Gray, rl.DarkGray
			} else if t.Shop {
				fill, outline = rl.Brown, brownOutline
			}

			// highlights should win visually
			if c, ok := highlights[TileID{Loop: li, Index: i}]; ok {
				fill = c
			}

			x := int32(t.Pos.X - tileSize/2)
			y := int32(t.Pos.Y - tileSize/2)
			rl.DrawRectangle(x, y, int32(tileSize), int32(tileSize), fill)
			rl.DrawRectangleLines(x, y, int32(tileSize), int32(tileSize), outline)

			// (optional) index label
			rl.DrawText(fmt.Sprintf("%d", i), x+int32(tileSize)/2-6, y+int32(tileSize)/2-8, fontSize, rl.Black)
		}
	}
}
func drawPlayer(p Player, w *World) {
	pos := w.Loops[p.At.Loop].Tiles[p.At.Index].Pos
	rl.DrawCircle(int32(pos.X), int32(pos.Y), 10, rl.Red)
}

func drawWorld(w *World, g *Game) {
	rl.BeginDrawing()
	rl.ClearBackground(rl.RayWhite)

	// Build highlight map (only when choosing a direction)
	hi := map[TileID]rl.Color{}
	if g.Phase == PhaseTargetSelect && len(g.Dests) > 0 {
		for i, id := range g.Dests {
			col := rl.Fade(rl.Orange, 0.45)
			if i == g.Selected {
				col = rl.Fade(rl.DarkGreen, 0.65)
			}
			hi[id] = col
		}
	}

	// --- Draw playfield (everything except menu bar) ---
	// drawLinks(w)
	rl.BeginMode2D(cam)
	drawLoops(w, hi)
	drawPlayer(g.Player, w)
	rl.EndMode2D()
	drawCard(g)
	// --- Draw menu bar ---
	drawMenuBar(g)

	rl.EndDrawing()
}
func drawCard(g *Game) {
	// --- card modal on top (if any) ---
	if g.CardActive {
		confirm, cancel := g.Card.Display()
		if !g.CardResolved {
			if confirm {
				// resolve the card effect
				beforeHP, beforeStr := g.Player.Health, g.Player.Strength
				g.Player.Interact(&g.Card)
				// craft a small result message
				switch g.Card.Type {
				case monsterType:
					if g.Player.Health < beforeHP {
						g.CardMsg = "The monster wounds you! (-1 Health)"
					} else {
						g.CardMsg = "You defeated the monster! (Card collected)"
					}
				case buffType:
					delta := g.Player.Strength - beforeStr
					if delta > 0 {
						g.CardMsg = fmt.Sprintf("You feel stronger! (+%d Strength)", delta)
					} else {
						g.CardMsg = "Nothing happens."
					}
				}
				g.CardResolved = true
			}
			if cancel {
				// skipped; just close
				g.CardActive = false
			}
		} else {
			// show a second line with the result; close on any key
			// draw the extra line under the modal (reuse the same rect placement)
			// (cheap: draw text near bottom of screen)
			rl.DrawText(g.CardMsg, 40, int32(screenHeight-menuHeight)-80, 22, rl.DarkGreen)
			if rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter) || rl.IsKeyPressed(rl.KeyEscape) || rl.IsKeyPressed(rl.KeySpace) {
				g.CardActive = false
			}
		}
	}

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
	rl.DrawText(fmt.Sprintf("Turn %d", g.Turn), leftX, y+12, 26, hudText)

	leftX = drawStat(leftX, y+44, "Steps", fmt.Sprintf("%d", g.StepsRemaining)) + colGap
	if g.LastRoll > 0 {
		// dice badge
		dtxt := fmt.Sprintf("Dice %d", g.LastRoll)
		rl.DrawText(dtxt, leftX, y+44, 18, hudSub)
		rl.DrawText(fmt.Sprintf("%d", g.LastRoll), leftX, y+64, 28, hudAccent)
	}
	// phase hint
	hint := ""
	switch g.Phase {
	case PhaseIdle:
		hint = "R to roll"
	case PhaseTargetSelect:
		hint = "←/→ pick • Enter confirm • Esc cancel"
	case PhaseAnimating:
		hint = "Resolving…"
	}
	if hint != "" {
		rl.DrawText(hint, padding, y+menuHeight-28, 20, hudSub)
	}

	// separator
	drawHairlineY(int32(screenWidth)/2-260, y+10, y+menuHeight-10)

	// --- MIDDLE: player stats ---
	statX := midX
	statX = drawStat(statX, y+10, "Strength", fmt.Sprintf("%d", g.Player.Strength))
	// HP block
	lbl := "HP"
	if g.Player.Health <= 3 {
		lbl = "HP (low)"
	}
	rl.DrawText(lbl, statX, y+10, 18, hudSub)
	drawHPBar(statX, y+32, 200, 18, g.Player.Health, 10) // assumes 10 max; make it a var if needed
	rl.DrawText(fmt.Sprintf("%d/10", g.Player.Health), statX+210, y+28, 22, hudText)

	// separator
	drawHairlineY(int32(screenWidth)/2+240, y+10, y+menuHeight-10)

	// --- RIGHT: collected cards (single row) ---
	rl.DrawText("Cards", rightX, y+12, 18, hudSub)
	drawCardStrip(rightX, y+36, 500, g.Player.Cards)
}

// --- HUD palette ---
var hudBG = rl.NewColor(22, 22, 26, 235)       // dark, slightly translucent
var hudLine = rl.NewColor(255, 255, 255, 30)   // hairlines
var hudText = rl.NewColor(235, 235, 238, 255)  // main text
var hudSub = rl.NewColor(185, 185, 192, 255)   // secondary text
var hudAccent = rl.NewColor(255, 196, 86, 255) // accent (dice, highlights)
var hpOK = rl.NewColor(104, 216, 164, 255)
var hpWarn = rl.NewColor(255, 120, 120, 255)

func drawHairlineY(x int32, y1, y2 int32) {
	rl.DrawLine(x, y1, x, y2, hudLine)
}

func drawHairlineX(x1, x2, y int32) {
	rl.DrawLine(x1, y, x2, y, hudLine)
}

func drawStat(x, y int32, label string, value string) int32 {
	lh := int32(18)
	rl.DrawText(label, x, y, 18, hudSub)
	w := int32(rl.MeasureText(value, 28))
	rl.DrawText(value, x, y+lh+2, 28, hudText)
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
		} else {
			title = fmt.Sprintf("Buff +%d", c.Strength)
		}
		w := int32(rl.MeasureText(title, 18)) + 20
		if cursor+w > x+maxW {
			// truncated indicator
			if i < len(cards) {
				rl.DrawText("…", cursor+2, y+3, 22, hudSub)
			}
			break
		}
		// chip
		bg := rl.NewColor(255, 255, 255, 18)
		if c.Type == buffType {
			bg = rl.NewColor(126, 194, 126, 35)
		} else {
			bg = rl.NewColor(194, 172, 142, 35)
		}
		rl.DrawRectangleRounded(rl.NewRectangle(float32(cursor), float32(y), float32(w), float32(chipH)), 0.35, 8, bg)
		rl.DrawRectangleRoundedLines(rl.NewRectangle(float32(cursor), float32(y), float32(w), float32(chipH)), 0.35, 8, rl.NewColor(255, 255, 255, 32))
		rl.DrawText(title, cursor+10, y+4, 18, hudText)
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
			g, specs[i].gx, specs[i].gy, specs[i].cols, specs[i].rows))
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

			// create the 1-tile shop loop
			shop := makeShop(g, shopCell.X, shopCell.Y)
			world.Loops = append(world.Loops, shop)
			shopIdx := len(world.Loops) - 1
			finalizeLoopIndices(world)

			// link shop <-> perimeter tile
			world.Loops[shopIdx].Tiles[0].Links = []TileID{{Loop: li, Index: ti}}
			pt.Links = append(pt.Links, TileID{Loop: shopIdx, Index: 0})

			// reserve the cell so nothing else spawns here
			markCells(occ, shopCell)
		}
	}
}

func drawPill(x, y int32, label string, value string, bg rl.Color) (right int32) {
	padding := int32(10)
	h := int32(30)
	w := int32(rl.MeasureText(label+" "+value, 20)) + padding*2
	rl.DrawRectangleRounded(rl.NewRectangle(float32(x), float32(y), float32(w), float32(h)), 0.4, 8, bg)
	rl.DrawRectangleRoundedLines(rl.NewRectangle(float32(x), float32(y), float32(w), float32(h)), 0.4, 8, rl.NewColor(0, 0, 0, 40))
	rl.DrawText(label, x+padding, y+6, 20, rl.Black)
	lw := int32(rl.MeasureText(label, 20))
	rl.DrawText(value, x+padding+lw+6, y+6, 20, rl.DarkGray)
	return x + w + 8
}

func drawHearts(x, y int32, hp int) (right int32) {
	size := int32(18)
	sp := int32(6)
	for i := 0; i < hp; i++ {
		rl.DrawCircle(x+int32(i)*(size+sp)+size/2, y+size/2, float32(size)/2, rl.Maroon)
		rl.DrawCircleLines(x+int32(i)*(size+sp)+size/2, y+size/2, float32(size)/2, rl.NewColor(0, 0, 0, 60))
	}
	return x + int32(hp)*(size+sp)
}

func cardChipColor(t CardType) rl.Color {
	if t == monsterType {
		return rl.NewColor(235, 214, 186, 255) // light sand
	}
	return rl.NewColor(204, 231, 204, 255) // light green
}

func drawCardChip(x, y int32, c Card) (right int32) {
	padding := int32(8)
	title := ""
	if c.Type == monsterType {
		title = fmt.Sprintf("Monster %d", c.Strength)
	} else {
		title = fmt.Sprintf("Buff +%d", c.Strength)
	}
	w := int32(rl.MeasureText(title, 18)) + padding*2
	h := int32(26)
	rl.DrawRectangleRounded(rl.NewRectangle(float32(x), float32(y), float32(w), float32(h)), 0.4, 8, cardChipColor(c.Type))
	rl.DrawRectangleRoundedLines(rl.NewRectangle(float32(x), float32(y), float32(w), float32(h)), 0.4, 8, rl.NewColor(0, 0, 0, 40))
	rl.DrawText(title, x+padding, y+5, 18, rl.Black)
	return x + w + 6
}
