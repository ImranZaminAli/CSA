package gol

import (
	//"CSA/distributed/stubs"
	"flag"
	"fmt"
	"net/rpc"
	"strconv"
	"time"

	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

type distributorChannels struct {
	events     chan<- Event
	ioCommand  chan<- ioCommand
	ioIdle     <-chan bool
	ioFilename chan<- string
	ioOutput   chan<- uint8
	ioInput    <-chan uint8
	keyPresses <-chan rune
}

var server *string = flag.String("server", "127.0.0.1:8030", "IP:port string to connect to as server")

// func makeCall(client rpc.Client, world *[][]byte, p Params, c distributorChannels, newTurn *int) {
// 	paused := 0
// 	ticker := time.NewTicker(2 * time.Second)
// loop:
// 	for *newTurn < p.Turns {
// 		request := stubs.Request{*world, *newTurn, p.Threads, p.ImageWidth, p.ImageHeight, paused}
// 		response := new(stubs.Response)
// 		client.Call(stubs.ProcessTurnsHandler, request, response)
// 		*world = response.World
// 		c.events <- TurnComplete{*newTurn}
// 		(*newTurn)++

// 		select {
// 		case <-ticker.C:

// 			cellCountRequest := stubs.EventRequest{*newTurn, *world}
// 			cellCountResponse := new(stubs.EventResponse)
// 			client.Call(stubs.CellCount, cellCountRequest, cellCountResponse)
// 			c.events <- AliveCellsCount{*newTurn, cellCountResponse.CellsAlive}
// 		case key := <-c.keyPresses:
// 			if key == 's' {
// 				createPgm(p, *newTurn, *world, c)
// 			} else if key == 'q' {
// 				c.events <- StateChange{*newTurn, Quitting}
// 				break loop
// 			} else if key == 'p' {
// 				if paused == 0 {
// 					paused = 1
// 					fmt.Println(*newTurn)
// 					c.events <- StateChange{*newTurn, Paused}
// 				} else {
// 					paused = 0
// 					fmt.Println("Continuing")
// 					c.events <- StateChange{*newTurn, Executing}
// 				}
// 			} else if key == 'k' {
// 				shutDownReq := stubs.ShutDownRequest{}
// 				shutDownRes := new(stubs.ShutDownResponse)

// 				client.Call(stubs.ShutDown, shutDownReq, shutDownRes)
// 				break loop
// 			}
// 		default:
// 		}
// 	}

// }
/*
func makeCall(client rpc.Client, world *[][]byte, p Params, c distributorChannels, turn *int, output *bool) {
	ticker := time.NewTicker(2 * time.Second)

	processTurnsRequest := stubs.ProcessTurnsRequest{*world, p.Turns, p.Threads, p.ImageHeight, p.ImageWidth}
	processTurnsResponse := new(stubs.ProcessTurnsResponse)

	finished := make(chan bool)
	paused := false
	go func() {
		for {
			select {
			case <-ticker.C:
				aliveCellCountRequest := stubs.AliveCellCountRequest{}
				aliveCellCountResponse := new(stubs.AliveCellsCountResponse)
				client.Call(stubs.AliveCellCountHandler, aliveCellCountRequest, aliveCellCountResponse)
				c.events <- AliveCellsCount{aliveCellCountResponse.Turns, aliveCellCountResponse.CellsAlive}
				c.events <- TurnComplete{aliveCellCountResponse.Turns}
				*turn = aliveCellCountResponse.Turns

			case key := <-c.keyPresses:
				keyPressRequest := stubs.KeyPressRequest{key}
				keyPressResponse := new(stubs.KeyPressResponse)
				client.Call(stubs.KeyPressHandler, keyPressRequest, keyPressResponse)
				*turn = keyPressResponse.Turns
				*world = keyPressResponse.World
				fmt.Println("\n\n", keyPressResponse.Turns, len(calculateAliveCells(keyPressResponse.World)), "\n\n")

				switch key {
				case 'p':
					paused = !paused
					if paused {
						fmt.Println(keyPressResponse.Turns)
						c.events <- StateChange{keyPressResponse.Turns, Paused}
					} else {
						fmt.Println("Continuing")
						c.events <- StateChange{keyPressResponse.Turns, Executing}
					}
				case 'q':
					c.events <- StateChange{keyPressResponse.Turns, Quitting}
					*output = false
					finished <- true
				case 's':
					createPgm(p, keyPressResponse.Turns, keyPressResponse.World, c)
				case 'k':

					createPgm(p, keyPressResponse.Turns, keyPressResponse.World, c)
					fmt.Println(len(keyPressResponse.World))
					c.events <- StateChange{keyPressResponse.Turns, Quitting}
					finished <- true
				}
				// case 'k':
				// 	shut down server - done
				// 	shut down distributor
				// 	output pgm of latest state
				// 	change state event latest turn
			}

		}
	}()

	// go func (){
	// 	client.Call(stubs.ProcessTurnsHandler, processTurnsRequest, processTurnsResponse)
	// 	finished <- true
	// }()

	go func() {
		client.Call(stubs.ProcessTurnsHandler, processTurnsRequest, processTurnsResponse)
		*output = true
		finished <- true
	}()

	*world = processTurnsResponse.World
	*turn = processTurnsResponse.Turns

	<-finished

}*/
func makeCall(client rpc.Client, world *[][]byte, p Params, c distributorChannels, newTurn *int, output *bool) {
	processTurnsRequest := stubs.ProcessTurnsRequest{*world, p.Turns, p.Threads, p.ImageHeight, p.ImageWidth}
	processTurnsResponse := new(stubs.ProcessTurnsResponse)

	ticker := time.NewTicker(2 * time.Second)
	paused := false
	go func() {
		for {
			if !paused {
				select {
				case <-ticker.C:
					aliveCellCountRequest := stubs.AliveCellCountRequest{}
					aliveCellCountResponse := new(stubs.AliveCellsCountResponse)
					client.Call(stubs.AliveCellCountHandler, aliveCellCountRequest, aliveCellCountResponse)
					c.events <- TurnComplete{aliveCellCountResponse.Turns}
					c.events <- AliveCellsCount{aliveCellCountResponse.Turns, aliveCellCountResponse.CellsAlive}

				case key := <-c.keyPresses:
					keyPressRequest := stubs.KeyPressRequest{key}
					keyPressResponse := new(stubs.KeyPressResponse)
					client.Call(stubs.KeyPressHandler, keyPressRequest, keyPressResponse)
					*newTurn = keyPressResponse.Turns
					*world = keyPressResponse.World
					switch key {
					case 'p':
						paused = true
						fmt.Println(*newTurn)
						c.events <- StateChange{*newTurn, Paused}
						// do something
					case 'q':
						// do something
					case 's':
						// do something
					case 'k':
						// do something
					}

				}
			} else {
				key := <-c.keyPresses
				if key == 'p' {
					paused = false
					fmt.Println("Continuing")
					c.events <- StateChange{*newTurn, Executing}
				} else if key == 'k' {

				}

			}
		}
		ticker.Stop()
	}()

	client.Call(stubs.ProcessTurnsHandler, processTurnsRequest, processTurnsResponse)

	*newTurn = processTurnsResponse.Turns
	*world = processTurnsResponse.World

}

func calculateAliveCells(world [][]byte) []util.Cell {
	cells := []util.Cell{}

	for y := range world {
		for x := range world[y] {
			if world[y][x] == 255 {
				cells = append(cells, util.Cell{X: x, Y: y})
			}
		}
	}

	return cells
}

func createPgm(p Params, turn int, world [][]byte, c distributorChannels) {
	filename := strconv.Itoa(p.ImageHeight) + "x" + strconv.Itoa(p.ImageWidth) + "x" + strconv.Itoa(turn)
	c.ioCommand <- ioOutput
	c.ioFilename <- filename
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			c.ioOutput <- world[y][x]
		}

	}

	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- ImageOutputComplete{turn, filename}
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {
	// TODO: Create a 2D slice to store the world.
	var world [][]byte

	c.ioCommand <- ioInput
	c.ioFilename <- strconv.Itoa(p.ImageHeight) + "x" + strconv.Itoa(p.ImageWidth)
	for y := 0; y < p.ImageHeight; y++ {
		world = append(world, []byte{})
		for x := 0; x < p.ImageWidth; x++ {
			world[y] = append(world[y], <-c.ioInput)
		}
	}

	flag.Parse()
	client, _ := rpc.Dial("tcp", *server)
	defer client.Close()

	//var newWorld [][]byte
	newTurn := 0
	output := true

	makeCall(*client, &world, p, c, &newTurn, &output)

	// if output {
	// 	createPgm(p, newTurn, world, c)
	// }

	// TODO: Report the final state using FinalTurnCompleteEvent.
	c.events <- FinalTurnComplete{newTurn, calculateAliveCells(world)}

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{newTurn, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}
