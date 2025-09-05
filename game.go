package main

import (
	"math/rand"
	"sort"
)

const stepDelay = 0.18 // seconds between tile steps while resolving a roll
type Phase int

const (
	PhaseIdle Phase = iota
	PhaseTargetSelect
	PhaseAnimating
)

type Game struct {
	Player         Player
	World          *World
	Turn           int
	LastRoll       int
	StepsRemaining int
	stepAccum      float32

	Dir   int   // +1 = CW (Right/Next), -1 = CCW (Left/Prev)
	Phase Phase // Idle -> ChooseDir -> Animating

	// selection
	Dests    []TileID // all valid landing spots this turn
	Selected int      // index into Dests
	Path     []TileID // animation path after confirm (excluding current tile)

	CardActive   bool
	Card         Card
	CardResolved bool
	CardMsg      string // result message after Interact
}

func (g *Game) CanRoll() bool { return g.Phase == PhaseIdle }

func (g *Game) Roll() {
	g.LastRoll = rand.Intn(6) + 1
	g.StepsRemaining = g.LastRoll
	g.Turn++

	// build destinations immediately (both directions; bridges allowed; no shops)
	g.Dests = gatherAllLandingSpots(g.Player.At, g.StepsRemaining, g.World) // see below
	g.Selected = 0
	g.Path = nil
	g.stepAccum = 0
	g.Phase = PhaseTargetSelect
}

func (g *Game) Update(dt float32, w *World) {
	if g.Phase != PhaseAnimating || g.StepsRemaining <= 0 {
		return
	}
	g.stepAccum += dt
	for g.StepsRemaining > 0 && g.stepAccum >= stepDelay {
		if len(g.Path) > 0 {
			g.Player.At = g.Path[0]
			g.Path = g.Path[1:]
		}
		g.StepsRemaining--
		g.stepAccum -= stepDelay
	}
	if g.StepsRemaining == 0 {
		// Spawn a card for the tile we just landed on
		g.Card = RandCard()
		g.CardActive = true
		g.CardResolved = false
		g.CardMsg = ""
		g.Phase = PhaseIdle // keep idle for input; modal will capture keys
		g.Dests = nil
		g.Path = nil
	}
}

func isShopTile(w *World, id TileID) bool {
	t := w.Loops[id.Loop].Tiles[id.Index]
	if t.Shop { // explicit flag for 1-tile shop loops
		return true
	}
	// if ever you mark perimeter tiles as Shop in future, this still works
	return false
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

// helper: is this a bridge tile?
func isBridgeTile(w *World, id TileID) bool {
	return w.Loops[id.Loop].Tiles[id.Index].Bridge
}

// helper: next/prev step along the loop in a fixed direction
func stepAlongDir(w *World, id TileID, dir int) TileID {
	if dir >= 0 {
		return w.Loops[id.Loop].Tiles[id.Index].Next
	}
	return w.Loops[id.Loop].Tiles[id.Index].Prev
}

// Bridge FSM phases while searching a single roll.
const (
	bridgeNone   = 0 // not on a bridge, not in a forced sequence
	bridgePhase1 = 1 // just stepped ONTO the first bridge tile -> must go to the other bridge tile next
	bridgePhase2 = 2 // now on the second bridge tile -> must EXIT to its linked perimeter tile
)

// BFS in a fixed direction. Every hop costs 1 step.
// Returns: set of legal endpoints (not shops, not bridge tiles) and a parent map to backtrack one path.
func bfsFixedDir(w *World, start TileID, steps int, dir int) (endpoints map[TileID]bool, parent map[TileID]TileID) {
	type State struct {
		id          TileID
		k           int  // steps left
		usedBridge  bool // true once we've stepped onto any bridge tile this roll
		bridgePhase int  // bridgeNone / bridgePhase1 / bridgePhase2
	}

	// queue & visited (include usedBridge and bridgePhase in the key)
	queue := []State{{id: start, k: steps, usedBridge: false, bridgePhase: bridgeNone}}
	seen := map[[5]int]bool{
		{start.Loop, start.Index, steps, 0, bridgeNone}: true,
	}

	parent = map[TileID]TileID{}  // store one predecessor per tile (enough to backtrack a shortest path)
	endpoints = map[TileID]bool{} // tiles reachable with exactly 0 steps

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		// If no steps left, we can end here (but never on shop or bridge tiles).
		if cur.k == 0 {
			if !isShopTile(w, cur.id) && !isBridgeTile(w, cur.id) {
				endpoints[cur.id] = true
			}
			continue
		}

		// Current tile info
		ct := w.Loops[cur.id.Loop].Tiles[cur.id.Index]
		onBridge := ct.Bridge

		// Generate candidate neighbors BEFORE gating:
		stepNeighbor := stepAlongDir(w, cur.id, dir) // along-loop step in fixed dir

		// Candidate link neighbors from here (shops will be filtered out)
		linkNeighbors := ct.Links

		// Now apply the movement rules / gating based on the bridge FSM.
		enqueue := func(nextID TileID, nextK int, used bool, phase int) {
			key := [5]int{nextID.Loop, nextID.Index, nextK, btoi(used), phase}
			if seen[key] {
				return
			}
			seen[key] = true
			// store parent only once (shortest due to BFS)
			if _, ok := parent[nextID]; !ok {
				parent[nextID] = cur.id
			}
			queue = append(queue, State{id: nextID, k: nextK, usedBridge: used, bridgePhase: phase})
		}

		switch cur.bridgePhase {
		case bridgeNone:
			// Normal movement: along the loop (if not a shop), and possibly enter a bridge (if not used yet).
			// 1) along-loop step
			if !isShopTile(w, stepNeighbor) && !isBridgeTile(w, stepNeighbor) {
				enqueue(stepNeighbor, cur.k-1, cur.usedBridge, bridgeNone)
			}

			// 2) follow any links that are legal
			for _, nb := range linkNeighbors {
				// skip shops
				if isShopTile(w, nb) {
					continue
				}
				if isBridgeTile(w, nb) {
					// entering a bridge is allowed ONLY if we haven't used one yet
					if cur.usedBridge {
						continue
					}
					// Step ONTO the first bridge tile (cost 1), must go to the other bridge tile next.
					enqueue(nb, cur.k-1, true /*used now*/, bridgePhase1)
				} else {
					// non-bridge, non-shop link (rare in your map, but keep rule clean)
					enqueue(nb, cur.k-1, cur.usedBridge, bridgeNone)
				}
			}

			// Edge case: if we *start* already standing on a bridge tile, force the same logic as Phase1.
			// (This shouldn't happen in normal flow, but keeps the search robust.)
			if onBridge {
				// The bridge loop has next/prev both pointing to the other bridge tile,
				// so just force the "go to other bridge tile" step.
				other := stepNeighbor
				if !isShopTile(w, other) && isBridgeTile(w, other) {
					enqueue(other, cur.k-1, true, bridgePhase2)
				}
			}

		case bridgePhase1:
			// We are on the FIRST bridge tile: the ONLY legal move is to the other bridge tile via loop step.
			other := stepNeighbor // in the 2-tile bridge loop, this is the other bridge tile
			if isBridgeTile(w, other) {
				enqueue(other, cur.k-1, cur.usedBridge, bridgePhase2)
			}
			// No links allowed from this tile in this phase (can't hop off mid-bridge).

		case bridgePhase2:
			// We are on the SECOND bridge tile: the ONLY legal move is to EXIT via its link to perimeter.
			for _, nb := range linkNeighbors {
				if isShopTile(w, nb) {
					continue
				}
				// The exit must be to a NON-bridge tile.
				if isBridgeTile(w, nb) {
					continue
				}
				enqueue(nb, cur.k-1, cur.usedBridge, bridgeNone)
			}
			// Do NOT allow stepping along the tiny bridge loop here (would bounce back).
		}
	}

	return
}

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

// union of CW and CCW endpoints + remember a parent map for later path extraction
type reachResult struct {
	endpoints map[TileID]bool
	parentCW  map[TileID]TileID
	parentCCW map[TileID]TileID
}

// run both directions and merge
func reachBothDirs(w *World, start TileID, steps int) reachResult {
	endsCW, pCW := bfsFixedDir(w, start, steps, +1)
	endsCC, pCC := bfsFixedDir(w, start, steps, -1)
	union := map[TileID]bool{}
	for id := range endsCW {
		union[id] = true
	}
	for id := range endsCC {
		union[id] = true
	}
	// never include the start as a “destination” if steps>0
	if steps > 0 {
		delete(union, start)
	}
	return reachResult{endpoints: union, parentCW: pCW, parentCCW: pCC}
}

// gather + stable order for UI
func gatherAllLandingSpots(start TileID, steps int, w *World) []TileID {
	r := reachBothDirs(w, start, steps)
	out := make([]TileID, 0, len(r.endpoints))
	for id := range r.endpoints {
		out = append(out, id)
	}
	// stable-ish ordering: loop asc, index asc
	sort.Slice(out, func(i, j int) bool {
		if out[i].Loop != out[j].Loop {
			return out[i].Loop < out[j].Loop
		}
		return out[i].Index < out[j].Index
	})
	return out
}

func buildPathTo(w *World, start TileID, steps int, goal TileID) []TileID {
	endsCW, pCW := bfsFixedDir(w, start, steps, +1)
	if endsCW[goal] {
		return backtrackPath(pCW, start, goal)
	}
	endsCC, pCC := bfsFixedDir(w, start, steps, -1)
	if endsCC[goal] {
		return backtrackPath(pCC, start, goal)
	}
	return nil
}

func backtrackPath(parent map[TileID]TileID, start, goal TileID) []TileID {
	// parent maps each node to *a* predecessor; rebuild then reverse.
	cur := goal
	path := []TileID{cur}
	for cur != start {
		p, ok := parent[cur]
		if !ok {
			break
		}
		cur = p
		path = append(path, cur)
	}
	// reverse to be start->goal, and drop the first (=start) because you’re already there
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	if len(path) > 0 && path[0] == start {
		return path[1:]
	}
	return path
}

// small helpers
func stepDir(id TileID, dir int, w *World) TileID {
	if dir >= 0 {
		return w.Loops[id.Loop].Tiles[id.Index].Next
	}
	return w.Loops[id.Loop].Tiles[id.Index].Prev
}

func destAfterSteps(start TileID, steps, dir int, w *World) TileID {
	cur := start
	for i := 0; i < steps; i++ {
		cur = stepDir(cur, dir, w)
	}
	return cur
}
