package gol

import (
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

func makeCall(client rpc.Client, world *[][]byte, p Params, c distributorChannels, newTurn *int, output, finished, shutdown *bool) {
	processTurnsRequest := stubs.ProcessTurnsRequest{*world, p.Turns, p.Threads, p.ImageHeight, p.ImageWidth}
	processTurnsResponse := new(stubs.ProcessTurnsResponse)

	ticker := time.NewTicker(1 * time.Second)
	paused := false
	go func() {
		for !*finished {
			if !paused {
				select {
				case <-ticker.C:
					aliveCellCountRequest := stubs.AliveCellCountRequest{}
					aliveCellCountResponse := new(stubs.AliveCellsCountResponse)
					client.Call(stubs.AliveCellCountHandler, aliveCellCountRequest, aliveCellCountResponse)
					if !*finished {
						c.events <- TurnComplete{aliveCellCountResponse.Turns}
						c.events <- AliveCellsCount{aliveCellCountResponse.Turns, aliveCellCountResponse.CellsAlive}
					}

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
					case 'q':
						*output = false
					case 's':
						createPgm(p, *newTurn, *world, c)
					case 'k':
						*shutdown = true
					}
				}
			} else {
				key := <-c.keyPresses
				keyPressRequest := stubs.KeyPressRequest{key}
				keyPressResponse := new(stubs.KeyPressResponse)
				client.Call(stubs.KeyPressHandler, keyPressRequest, keyPressResponse)
				*newTurn = keyPressResponse.Turns
				*world = keyPressResponse.World
				if key == 'p' {
					paused = false
					fmt.Println("Continuing")
					c.events <- StateChange{*newTurn, Executing}
				} else if key == 'k' {
					*shutdown = true
				}

			}
		}
		ticker.Stop()
	}()

	client.Call(stubs.ProcessTurnsHandler, processTurnsRequest, processTurnsResponse)
	*finished = true
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

	newTurn := 0
	output := true
	finished := false
	shutdown := false
	makeCall(*client, &world, p, c, &newTurn, &output, &finished, &shutdown)

	if shutdown {
		client.Call(stubs.ShutDownHandler, stubs.ShutDownRequest{}, new(stubs.ShutDownResponse))
	}

	if output {
		client.Call(stubs.QuitHandler, stubs.QuitRequest{}, new(stubs.QuitResponse))
		createPgm(p, newTurn, world, c)
	}

	c.ioCommand <- ioCheckIdle
	<-c.ioIdle
	c.events <- FinalTurnComplete{newTurn, calculateAliveCells(world)}

	// Make sure that the Io has finished any output before exiting.

	c.events <- StateChange{newTurn, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}
