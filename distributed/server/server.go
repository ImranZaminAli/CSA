package main

import (
	"flag"
	"math/rand"
	"net"
	"net/rpc"
	"os"
	"time"

	"uk.ac.bris.cs/gameoflife/stubs"
)

type GameOfLifeOperations struct {
	end chan bool
}

func (s *GameOfLifeOperations) ShutDown(req stubs.ShutDownRequest, res *stubs.ShutDownResponse) (err error) {
	s.end <- true
	return
}

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

func (s *GameOfLifeOperations) ExecuteTurn(req stubs.ExecuteTurnRequest, res *stubs.ExecuteTurnResponse) (err error) {
	newWorld := worldCopy(req.World)
	w := req.ImageWidth
	h := req.ImageHeight
	for y := range req.World {
		for x := range req.World[y] {

			// say y is 0 (goes from 0 to 254 for 255 height matrix)
			// above 0 would be -1 (if thinking about it in terms of the matrix going ^)
			// (0 + 255 - 1) % 255 = (254) % 255 = 254 (bottom of matrix)
			// taking the pixel matrix value MOD height/width whatever is relevant allows for the overlapping
			up := (y + h - 1) % h
			down := (y + h + 1) % h

			// (3 + 255 - 1) % 255 = 257 % 255 = 2 (remainder - which corresponds to the correct value as left of 3 is 2)
			left := (x + w - 1) % w
			right := (x + w + 1) % w
			neighbours := [8]byte{req.World[up][left], req.World[up][x], req.World[up][right], req.World[y][left], req.World[y][right], req.World[down][left], req.World[down][x], req.World[down][right]}

			// local count for the neighbours of a particular pixel
			cellsAlive := 0
			for n := range neighbours {
				if neighbours[n] == 255 {
					cellsAlive++
				}
			}

			if req.World[y][x] == 255 { // if alive
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

	res.World = newWorld

	return
}

func main() {
	pAddr := flag.String("port", "8030", "Port to listen on")
	flag.Parse()
	rand.Seed(time.Now().UnixNano())

	end := make(chan bool)
	listener, _ := net.Listen("tcp", ":"+*pAddr)

	rpc.Register(&GameOfLifeOperations{end})

	defer listener.Close()
	go rpc.Accept(listener)
	<-end
	os.Exit(0)
}
