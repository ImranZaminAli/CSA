package main

import (
	//	"CSA-old/distributed/stubs"
	"errors"
	"flag"
	"fmt"
	"math/rand"
	"net"
	"net/rpc"
	"os"
	"time"

	"uk.ac.bris.cs/gameoflife/stubs"
)

var server1 *string = flag.String("server1", "127.0.0.1:8031", "IP:port string to connect to as server")
var server2 *string = flag.String("server2", "127.0.0.1:8032", "IP:port string to connect to as server")

type GameOfLifeOperations struct {
	turn     int
	world    [][]byte
	paused   bool
	finished bool
	end      chan bool
	clients  [](*rpc.Client)
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

func genHalo(start, end, height, width int, world [][]byte) [][]byte {
	var subWorld [][]byte
	subWorld = append(subWorld, []byte{})
	for i := 0; i < width; i++ {
		if start == 0 {
			subWorld[0] = append(subWorld[0], world[height-1][i])
		} else {
			subWorld[0] = append(subWorld[0], world[start-1][i])
		}
	}
	index := 1
	for i := 0; i < end-start; i++ {

		subWorld = append(subWorld, []byte{})

		for j := 0; j < width; j++ {
			subWorld[index] = append(subWorld[i+1], world[start+i][j])
		}
		index++
	}
	subWorld = append(subWorld, []byte{})
	for i := 0; i < width; i++ {
		if end == height {
			subWorld[index] = append(subWorld[index], world[0][i])
		} else {
			fmt.Println("here")
			subWorld[index] = append(subWorld[index], world[end][i])
		}
	}

	return subWorld
}

func worker(client *rpc.Client, req stubs.ExecuteTurnRequest, res *stubs.ExecuteTurnResponse, channel chan [][]byte) {
	client.Call(stubs.TurnHandler, req, res)
	channel <- res.World
}

func execute(height, width int, world [][]byte, clients []*rpc.Client) [][]byte {
	hInc := height / len(clients)
	var responses []*stubs.ExecuteTurnResponse
	var channels []chan [][]byte
	for i := range clients {
		start := hInc * i
		end := hInc * (i + 1)
		if i == len(clients)-1 {
			end = height
		}

		subWorld := genHalo(start, end, height, width, world)
		executeTurnRequest := stubs.ExecuteTurnRequest{subWorld, end - start, width}
		responses = append(responses, new(stubs.ExecuteTurnResponse))
		channels = append(channels, make(chan [][]byte))
		go worker(clients[i], executeTurnRequest, responses[i], channels[i])
		//go s.clients[i].Call(stubs.TurnHandler, executeTurnRequest, responses[i])
	}

	var newWorld [][]byte
	for i := range channels {
		newWorld = append(newWorld, <-channels[i]...)
	}

	return newWorld
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
			// executeTurnRequest := stubs.ExecuteTurnRequest{s.world, req.ImageHeight, req.ImageWidth}
			// executeTurnResponse := new(stubs.ExecuteTurnResponse)

			// s.client1.Call(stubs.TurnHandler, executeTurnRequest, executeTurnResponse)
			// s.world = executeTurnResponse.World

			s.world = execute(req.ImageHeight, req.ImageWidth, s.world, s.clients)

			res.World = s.world
			s.turn++
			res.Turns++
			fmt.Println(s.turn)
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

	client1, _ := rpc.Dial("tcp", *server1)
	client2, _ := rpc.Dial("tcp", *server2)
	clients := [](*rpc.Client){client1, client2}
	defer client1.Close()
	defer client2.Close()

	rpc.Register(&GameOfLifeOperations{turn, world, paused, finished, end, clients})

	defer listener.Close()
	go rpc.Accept(listener)
	<-end
	for i := range clients {
		clients[i].Call(stubs.ShutDownHandler, stubs.ShutDownRequest{}, new(stubs.ShutDownResponse))
	}
	os.Exit(0)
}
