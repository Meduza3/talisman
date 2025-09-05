package main

import rl "github.com/gen2brain/raylib-go/raylib"

var cam rl.Camera2D

func clamp(v, lo, hi float32) float32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func playerPos(w *World, g *Game) rl.Vector2 {
	return w.Loops[g.Player.At.Loop].Tiles[g.Player.At.Index].Pos
}

// Smooth follow (lerp). Set `followSpeed` higher if you want snappier motion.
const followSpeed = float32(10.0)

func updateCamera(cam *rl.Camera2D, target rl.Vector2, dt float32) {
	// Lerp cam.Target -> target
	t := clamp(followSpeed*dt, 0, 1)
	cam.Target = rl.Vector2{
		X: cam.Target.X + (target.X-cam.Target.X)*t,
		Y: cam.Target.Y + (target.Y-cam.Target.Y)*t,
	}
}
