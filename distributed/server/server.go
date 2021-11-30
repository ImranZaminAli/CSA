package main

import (
	"errors"
	"flag"
	"math/rand"
	"net"
	"net/rpc"
	"os"
	"time"

	"uk.ac.bris.cs/gameoflife/stubs"
)

/** Super-Secret method we can't allow clients to see. **/
func worldCopy(world [][]byte) [][]byte {
	newWorld := [][]byte{}
	for i := range world {
		newWorld = append(newWorld, []byte{})
		for j := range world[i] {
			newWorld[i] = append(newWorld[i], world[i][j])
		}
	}
	return newWorld
}

/** Super-Secret method we can't allow clients to see. **/

func calculateNextState(req stubs.ProcessTurnsRequest, world [][]byte) [][]byte {
	newWorld := worldCopy(world)
	w := req.ImageWidth
	h := req.ImageHeight
	for y := range world {
		for x := range world[y] {

			// say y is 0 (goes from 0 to 254 for 255 height matrix)
			// above 0 would be -1 (if thinking about it in terms of the matrix going ^)
			// (0 + 255 - 1) % 255 = (254) % 255 = 254 (bottom of matrix)
			// taking the pixel matrix value MOD height/width whatever is relevant allows for the overlapping
			up := (y + h - 1) % h
			down := (y + h + 1) % h

			// (3 + 255 - 1) % 255 = 257 % 255 = 2 (remainder - which corresponds to the correct value as left of 3 is 2)
			left := (x + w - 1) % w
			right := (x + w + 1) % w
			neighbours := [8]byte{world[up][left], world[up][x], world[up][right], world[y][left], world[y][right], world[down][left], world[down][x], world[down][right]}

			// local count for the neighbours of a particular pixel
			cellsAlive := 0
			for n := range neighbours {
				if neighbours[n] == 255 {
					cellsAlive++
				}
			}

			if world[y][x] == 255 { // if alive
				if cellsAlive < 2 {
					newWorld[y][x] = 0
				} else if cellsAlive > 3 {
					newWorld[y][x] = 0
				}
			} else { // if dead
				if cellsAlive == 3 {
					newWorld[y][x] = 255
				}
			}
		}
	}

	return newWorld
}

type GameOfLifeOperations struct {
	turn   int
	world  [][]byte
	paused bool
}

func (s *GameOfLifeOperations) AliveCellCount(req stubs.AliveCellCountRequest, res *stubs.AliveCellsCountResponse) (err error) {
	res.Turns = s.turn

	cellsAlive := 0

	for y := range s.world {
		for x := range s.world {
			if s.world[y][x] == 255 {
				cellsAlive++
			}
		}
	}

	res.CellsAlive = cellsAlive

	return
}

func (s *GameOfLifeOperations) KeyPress(req stubs.KeyPressRequest, res *stubs.KeyPressResponse) (err error) {
	res.Turns = s.turn
	res.World = s.world

	if req.Key == 'p' {
		s.paused = !s.paused
	}

	return
}

func (s *GameOfLifeOperations) ProcessTurns(req stubs.ProcessTurnsRequest, res *stubs.ProcessTurnsResponse) (err error) {
	if len(req.World) == 0 {
		err = errors.New("A world must be given!")
		return
	}
	s.world = req.World
	res.World = req.World
	s.turn = 0
	for s.turn < req.Turns {
		if !s.paused {
			s.world = calculateNextState(req, s.world)
			s.turn++
		} else {

		}
		res.World = s.world
	}

	return
}

func main() {
	pAddr := flag.String("port", "8030", "Port to listen on")
	flag.Parse()
	rand.Seed(time.Now().UnixNano())

	turn := 0
	var world [][]byte
	paused := false
	//finished := make(chan bool)
	listener, _ := net.Listen("tcp", ":"+*pAddr)

	rpc.Register(&GameOfLifeOperations{turn, world, paused})

	defer listener.Close()
	rpc.Accept(listener)
	//<-finished
	os.Exit(0)
}
