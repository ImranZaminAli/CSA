package main

import (
	//	"CSA-old/distributed/stubs"
	"errors"
	"flag"
	"math/rand"
	"net"
	"net/rpc"
	"os"
	"time"

	"uk.ac.bris.cs/gameoflife/stubs"
)

var server *string = flag.String("server", "127.0.0.1:8030", "IP:port string to connect to as server")

type GameOfLifeOperations struct {
	turn     int
	world    [][]byte
	paused   bool
	finished bool
	end      chan bool
	client   *rpc.Client
}

func (s *GameOfLifeOperations) Quit(req stubs.QuitRequest, res *stubs.QuitResponse) (err error) {
	s.finished = true
	return
}

func (s *GameOfLifeOperations) ShutDown(req stubs.ShutDownRequest, res *stubs.ShutDownResponse) (err error) {
	s.end <- true
	return
}

func (s *GameOfLifeOperations) KeyPress(req stubs.KeyPressRequest, res *stubs.KeyPressResponse) (err error) {
	res.Turns = s.turn
	res.World = s.world

	if req.Key == 'p' {
		s.paused = !s.paused
	} else if req.Key == 'q' {
		s.finished = true
	} else if req.Key == 'k' {
		s.finished = true
	}

	return
}

func (s *GameOfLifeOperations) AliveCellCount(req stubs.AliveCellCountRequest, resp *stubs.AliveCellsCountResponse) (err error) {
	resp.Turns = s.turn

	cellsAlive := 0

	for y := range s.world {
		for x := range s.world {
			if s.world[y][x] == 255 {
				cellsAlive++
			}
		}
	}

	resp.CellsAlive = cellsAlive

	return
}

func (s *GameOfLifeOperations) ProcessTurns(req stubs.ProcessTurnsRequest, res *stubs.ProcessTurnsResponse) (err error) {
	if len(req.World) == 0 {
		err = errors.New("A world must be given!")
		return
	}
	s.finished = false
	s.world = req.World
	res.Turns = 0
	s.turn = 0
	res.World = req.World

	for s.turn < req.Turns && !s.finished {
		if !s.paused {
			executeTurnRequest := stubs.ExecuteTurnRequest{s.world, req.ImageHeight, req.ImageWidth}
			executeTurnResponse := new(stubs.ExecuteTurnResponse)

			s.client.Call(stubs.TurnHandler, executeTurnRequest, executeTurnResponse)
			s.world = executeTurnResponse.World
			res.World = s.world
			s.turn++
			res.Turns++
		}

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
	finished := false
	end := make(chan bool)
	listener, _ := net.Listen("tcp", ":"+*pAddr)

	client, _ := rpc.Dial("tcp", *server)
	defer client.Close()

	rpc.Register(&GameOfLifeOperations{turn, world, paused, finished, end, client})

	defer listener.Close()
	go rpc.Accept(listener)
	<-end
	client.Call(stubs.ShutDownHandler, stubs.ShutDownRequest{}, new(stubs.ShutDownResponse))
	os.Exit(0)
}
