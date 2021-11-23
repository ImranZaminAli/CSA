package main

import (
	_ "bufio"
	"errors"
	"flag"
	_ "fmt"
	"math/rand"
	"net"
	"net/rpc"
	_ "os"
	"time"
	"uk.ac.bris.cs/gameoflife/stubs"
	_ "uk.ac.bris.cs/gameoflife/stubs"
)

func makeCopy(world [][]byte) [][]byte{
	newWorld := [][]byte{}
	for i := range world {
		newWorld = append(newWorld, []byte{})
		for j := range world[i] {
			newWorld[i] = append(newWorld[i] , world[i][j])
		}
	}

	return newWorld
}

func calculateNextState(h, w int, world [][]byte) [][]byte{
	newWorld := makeCopy(world)

	for y := range world {
		for x := range world {
			up := (y + h - 1) % h
			down := (y + h + 1) % h
			left := (x + w - 1) % w
			right := (x + w + 1) % w

			neighbours := [8]byte{world[up][left], world[up][x], world[up][right], world[y][left], world[y][right], world[down][left], world[down][x], world[down][right]}
			cellsAlive := 0
			for n := range neighbours {
				if neighbours[n] == 255 {
					cellsAlive++
				}
			}

			if world[y][x] == 255 {
				if cellsAlive < 2 || cellsAlive > 3 {
					newWorld[y][x] = 0
				}
			} else {
				if cellsAlive == 3 {
					newWorld[y][x] = 255
				}

			}
		}
	}

	return newWorld
}

func (t *GameOfLifeOperations) processTurns(req stubs.Request, res *stubs.Response) error{
	if len(req.World) == 0 {
		err := errors.New("A world must be given!")
		return err
	}

	turn := 0
	world := req.World
	for turn < req.Turns {
		world = calculateNextState(req.ImageHeight, req.ImageWidth, world)
		turn++
	}
	res.Turns = turn
	res.World = world
	return nil
}

type GameOfLifeOperations struct {
}

func main(){
	pAddr := flag.String("port","8030","Port to listen on")
	flag.Parse()
	rand.Seed(time.Now().UnixNano())

	rpc.Register(&GameOfLifeOperations{})

	listener, _ := net.Listen("tcp", ":"+*pAddr)
	defer listener.Close()
	rpc.Accept(listener)
}
