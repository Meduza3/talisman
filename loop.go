package main

import (
	"math"
	"math/rand"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type Grid struct {
	Origin rl.Vector2 // top-left of grid in pixels
	Cell   float32    // tileSize
}

func (g Grid) Center(cx, cy int) rl.Vector2 {
	return rl.NewVector2(g.Origin.X+float32(cx)*g.Cell+g.Cell/2,
		g.Origin.Y+float32(cy)*g.Cell+g.Cell/2)
}
func (g Grid) CellOf(p rl.Vector2) (int, int) {
	fx := (p.X - g.Origin.X) / g.Cell
	fy := (p.Y - g.Origin.Y) / g.Cell
	cx := int(math.Floor(float64(fx)))
	cy := int(math.Floor(float64(fy)))
	return cx, cy
}

type World struct {
	Loops []Loop
}

type Loop struct {
	Tiles []Tile
	Type  LoopType
}

type LoopType struct {
	Name  string
	Color rl.Color
	Deck  Deck
}

var loop1 LoopType = LoopType{
	Name:  "Outer Fields",
	Color: rl.NewColor(120, 180, 100, 255), // Peaceful meadow green
	Deck:  deck1,
}

var loop2 LoopType = LoopType{
	Name:  "Forest Paths",
	Color: rl.NewColor(80, 140, 70, 255), // Deep forest green
	Deck:  deck2,
}

var loop3 LoopType = LoopType{
	Name:  "Desert Sands",
	Color: rl.NewColor(200, 160, 80, 255), // Warm desert sand
	Deck:  deck3,
}

var loop4 LoopType = LoopType{
	Name:  "Mountain Caves",
	Color: rl.NewColor(120, 110, 140, 255), // Mysterious cave purple-gray
	Deck:  deck4,
}

var loop5 LoopType = LoopType{
	Name:  "Fire Peaks",
	Color: rl.NewColor(180, 80, 60, 255), // Volcanic red-orange
	Deck:  deck5,
}

var loop6 LoopType = LoopType{
	Name:  "Shadow Realm",
	Color: rl.NewColor(60, 40, 80, 255), // Dark mystical purple
	Deck:  deck6,
}

var loops []LoopType = []LoopType{loop1, loop2, loop3, loop4, loop5, loop6}

type TileID struct {
	Loop, Index int
}

type ShopType struct {
	Name       string
	Cards      [3]Card
	Prices     [3]int
	Discovered bool
	KeeperType int // 0-3 for different shopkeeper types
}

type Tile struct {
	Pos        rl.Vector2
	Next, Prev TileID
	Links      []TileID
	Bridge     bool
	Shop       bool
	ShopData   *ShopType // Pointer to shop data if this is a shop tile
}

// Choose rectangle dimensions (cols, rows) s.t. perimeter = n and near-square.
// Perimeter formula: P = 2*cols + 2*rows - 4  (with cols>=2, rows>=2).
func rectDimsForPerimeter(n int) (cols, rows int, ok bool) {
	bestCols, bestRows, bestDelta := 0, 0, math.MaxInt32
	for r := 2; r <= n; r++ {
		// 2*cols + 2*r - 4 = n  => cols = (n - 2*r + 4)/2
		cnum := n - 2*r + 4
		if cnum%2 != 0 {
			continue
		}
		c := cnum / 2
		if c < 2 {
			continue
		}
		delta := int(math.Abs(float64(c - r)))
		if delta < bestDelta {
			bestDelta, bestCols, bestRows = delta, c, r
			ok = true
		}
	}
	return bestCols, bestRows, ok
}

// Build rect perimeter at top-left grid cell (gx, gy) with exact (cols, rows).
func buildRectPerimeterLoopAtGrid(g Grid, gx, gy, cols, rows int, loopIndex int) Loop {
	n := 2*cols + 2*rows - 4
	tiles := make([]Tile, 0, n)

	push := func(cx, cy int) {
		tiles = append(tiles, Tile{Pos: g.Center(cx, cy)})
	}

	// Perimeter walk
	for c := 0; c < cols; c++ {
		push(gx+c, gy+0)
	} // top
	for r := 1; r < rows-1; r++ {
		push(gx+cols-1, gy+r)
	} // right
	for c := cols - 1; c >= 0; c-- {
		push(gx+c, gy+rows-1)
	} // bottom
	for r := rows - 2; r >= 1; r-- {
		push(gx+0, gy+r)
	} // left

	// Wire CW neighbors (loop-local for now)
	for i := range tiles {
		tiles[i].Next = TileID{-1, (i + 1) % n}
		tiles[i].Prev = TileID{-1, (i - 1 + n) % n}
	}
	
	// Choose loop type: sequential for first 3, then random
	var loopType LoopType
	if loopIndex < 3 {
		loopType = loops[loopIndex] // loops[0], loops[1], loops[2]
	} else {
		loopType = loops[rand.Intn(len(loops))]
	}
	
	return Loop{Tiles: tiles, Type: loopType}
}

func sideMidCell(gx, gy, cols, rows int, side string) (cx, cy int) {
	switch side {
	case "right":
		return gx + cols - 1, gy + rows/2
	case "left":
		return gx, gy + rows/2
	case "top":
		return gx + cols/2, gy
	default: // "bottom"
		return gx + cols/2, gy + rows - 1
	}
}

func findTileIndexAtCell(g Grid, loop Loop, cx, cy int) int {
	tx := g.Center(cx, cy)
	for i, t := range loop.Tiles {
		if int(t.Pos.X) == int(tx.X) && int(t.Pos.Y) == int(tx.Y) {
			return i
		}
	}
	return -1
}

// Prefer horizontal for side middles, vertical for corners & top/bottom.
func outwardDirForCellOnRect(sp rectSpec, cx, cy int) (dx, dy int, ok bool) {
	onLeft := cx == sp.gx
	onRight := cx == sp.gx+sp.cols-1
	onTop := cy == sp.gy
	onBottom := cy == sp.gy+sp.rows-1

	// sides (excluding corners) → horizontal
	if onLeft && !onTop && !onBottom {
		return -1, 0, true
	}
	if onRight && !onTop && !onBottom {
		return +1, 0, true
	}

	// top/bottom rows (including corners) → vertical outward
	if onTop {
		return 0, -1, true
	}
	if onBottom {
		return 0, +1, true
	}

	return 0, 0, false
}
func makeBridge(g Grid, cellAx, cellAy, cellBx, cellBy int) Loop {
	t0 := Tile{Pos: g.Center(cellAx, cellAy), Bridge: true}
	t1 := Tile{Pos: g.Center(cellBx, cellBy), Bridge: true}
	// wire the two tiles as a tiny loop
	t0.Next, t0.Prev = TileID{-1, 1}, TileID{-1, 1}
	t1.Next, t1.Prev = TileID{-1, 0}, TileID{-1, 0}
	return Loop{Tiles: []Tile{t0, t1}}
}

func makeShop(g Grid, cellX, cellY int) Loop {
	t := Tile{Pos: g.Center(cellX, cellY), Shop: true}
	t.Next, t.Prev = TileID{-1, 0}, TileID{-1, 0} // 1-tile loop points to itself
	return Loop{Tiles: []Tile{t}}
}

func addBridgeTile(w *World, bridge TileID, a, b TileID) {
	// Link bridge ↔ both ends
	linkBoth(w, bridge, a)
	linkBoth(w, bridge, b)
}

func finalizeLoopIndices(w *World) {
	for li := range w.Loops {
		n := len(w.Loops[li].Tiles)
		for i := range w.Loops[li].Tiles {
			t := &w.Loops[li].Tiles[i]
			t.Next.Loop, t.Prev.Loop = li, li
			t.Next.Index = (i + 1) % n
			t.Prev.Index = (i - 1 + n) % n
		}
	}
}

func linkBoth(w *World, a, b TileID) {
	wa := &w.Loops[a.Loop].Tiles[a.Index]
	wb := &w.Loops[b.Loop].Tiles[b.Index]
	wa.Links = append(wa.Links, b)
	wb.Links = append(wb.Links, a)
}

type circle struct {
	C rl.Vector2
	R float32
}

// --- Directions ---
type Dir int

const (
	Right Dir = iota
	Left
	Up
	Down
)

func shuffledDirs() []Dir {
	d := []Dir{Right, Left, Up, Down}
	for i := range d {
		j := rand.Intn(i + 1)
		d[i], d[j] = d[j], d[i]
	}
	return d
}

// Mark which grid cells are used so loops/bridges don't overlap.
type cell struct{ X, Y int }

func markRectPerimeter(occ map[cell]bool, gx, gy, cols, rows int) {
	for c := 0; c < cols; c++ {
		occ[cell{gx + c, gy + 0}] = true
	}
	for r := 1; r < rows-1; r++ {
		occ[cell{gx + cols - 1, gy + r}] = true
	}
	for c := cols - 1; c >= 0; c-- {
		occ[cell{gx + c, gy + rows - 1}] = true
	}
	for r := rows - 2; r >= 1; r-- {
		occ[cell{gx + 0, gy + r}] = true
	}
}
func wouldCollideRectPerimeter(occ map[cell]bool, gx, gy, cols, rows int) bool {
	// Same perimeter walk, but check only
	for c := 0; c < cols; c++ {
		if occ[cell{gx + c, gy + 0}] {
			return true
		}
	}
	for r := 1; r < rows-1; r++ {
		if occ[cell{gx + cols - 1, gy + r}] {
			return true
		}
	}
	for c := cols - 1; c >= 0; c-- {
		if occ[cell{gx + c, gy + rows - 1}] {
			return true
		}
	}
	for r := rows - 2; r >= 1; r-- {
		if occ[cell{gx + 0, gy + r}] {
			return true
		}
	}
	return false
}
func markCells(occ map[cell]bool, pts ...cell) {
	for _, p := range pts {
		occ[p] = true
	}
}
func anyOccupied(occ map[cell]bool, pts ...cell) bool {
	for _, p := range pts {
		if occ[p] {
			return true
		}
	}
	return false
}

type rectSpec struct{ gx, gy, cols, rows int }

type placeResult struct {
	specB                      rectSpec
	bridge1, bridge2           cell // exact two bridge grid cells
	sideA, sideB               string
	aMidX, aMidY, bMidX, bMidY int
}

// Given A and B sizes, place B around A in dir with 2-cell gap for bridge.
func placeAround(A rectSpec, colsB, rowsB int, dir Dir) placeResult {
	switch dir {
	case Right:
		// A right mid → B left mid
		ax, ay := sideMidCell(A.gx, A.gy, A.cols, A.rows, "right")
		// gxB := ax + 1 + 1 // gap 2 cells: bridge at ax+1 and ax+2; B starts at ax+3 - (colsB-1)? No: align by left-mid
		gyB := ay - rowsB/2
		B := rectSpec{gx: A.gx + A.cols + 2, gy: gyB, cols: colsB, rows: rowsB}
		// recompute mids from finalized B pos
		bx, by := sideMidCell(B.gx, B.gy, B.cols, B.rows, "left")
		return placeResult{
			specB:   B,
			bridge1: cell{ax + 1, ay},
			bridge2: cell{ax + 2, ay}, // == bx-1, by
			sideA:   "right", sideB: "left",
			aMidX: ax, aMidY: ay, bMidX: bx, bMidY: by,
		}
	case Left:
		ax, ay := sideMidCell(A.gx, A.gy, A.cols, A.rows, "left")
		B := rectSpec{gx: A.gx - 2 - colsB, gy: ay - rowsB/2, cols: colsB, rows: rowsB}
		bx, by := sideMidCell(B.gx, B.gy, B.cols, B.rows, "right")
		return placeResult{
			specB:   B,
			bridge1: cell{ax - 1, ay},
			bridge2: cell{ax - 2, ay}, // == bx+1, by
			sideA:   "left", sideB: "right",
			aMidX: ax, aMidY: ay, bMidX: bx, bMidY: by,
		}
	case Down:
		ax, ay := sideMidCell(A.gx, A.gy, A.cols, A.rows, "bottom")
		B := rectSpec{gx: ax - colsB/2, gy: A.gy + A.rows + 2, cols: colsB, rows: rowsB}
		bx, by := sideMidCell(B.gx, B.gy, B.cols, B.rows, "top")
		return placeResult{
			specB:   B,
			bridge1: cell{ax, ay + 1},
			bridge2: cell{ax, ay + 2}, // == bx, by-1
			sideA:   "bottom", sideB: "top",
			aMidX: ax, aMidY: ay, bMidX: bx, bMidY: by,
		}
	default: // Up
		ax, ay := sideMidCell(A.gx, A.gy, A.cols, A.rows, "top")
		B := rectSpec{gx: ax - colsB/2, gy: A.gy - 2 - rowsB, cols: colsB, rows: rowsB}
		bx, by := sideMidCell(B.gx, B.gy, B.cols, B.rows, "bottom")
		return placeResult{
			specB:   B,
			bridge1: cell{ax, ay - 1},
			bridge2: cell{ax, ay - 2}, // == bx, by+1
			sideA:   "top", sideB: "bottom",
			aMidX: ax, aMidY: ay, bMidX: bx, bMidY: by,
		}
	}
}
