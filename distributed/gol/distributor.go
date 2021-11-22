package gol

import (
	"fmt"
	"strconv"
	"time"

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

// creates an identical instance of the world
func makeCopy(start, end int, world [][]byte) [][]byte {
	newWorld := [][]byte{}
	for i := start; i < end; i++ {
		newWorld = append(newWorld, []byte{})
		for j := range world[i] {
			newWorld[i-start] = append(newWorld[i-start], world[i][j])
		}
	}

	return newWorld
}

// deals with informing GUI about changing the value of a cell
func flipCell(turn, y, x int, events chan<- Event) {
	cell := util.Cell{X: y, Y: x}
	events <- CellFlipped{turn, cell}
}

// calculates the next state for each of the cells in the specified dimensions
func calculateNextState(turn, start, end int, p Params, world [][]byte, events chan<- Event) [][]byte {
	newWorld := makeCopy(start, end, world)
	w := p.ImageWidth
	h := p.ImageHeight

	for y := start; y < end; y++ {
		for x := 0; x < w; x++ {
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
					newWorld[y-start][x] = 0
					flipCell(turn, y, x, events)
				}
			} else {
				if cellsAlive == 3 {
					newWorld[y-start][x] = 255
					flipCell(turn, y, x, events)
				}

			}
		}
	}

	return newWorld
}

// produces an array of the coordinates of the alive cells for the current state
func getAliveCells(p Params, world [][]byte) []util.Cell {
	alive := []util.Cell{}
	for y := range world {
		for x := range world[y] {
			if world[y][x] == 255 {
				alive = append(alive, util.Cell{Y: y, X: x})
			}
		}
	}

	return alive
}

// runs one worker thread that calculates part of the next state for the game of life
// size of area covered is based on number of workers and image dimensions
func worker(turn, start, end int, p Params, world [][]byte, out chan [][]byte, events chan<- Event) {
	newStates := calculateNextState(turn, start, end, p, world, events)
	out <- newStates
}

// executes one turn of the game of life
func execute(turn int, p Params, world [][]byte, out chan [][]byte, events chan<- Event) {

	var newStates [][]byte
	var channels []chan [][]byte
	hInc := p.ImageHeight / p.Threads
	for i := 0; i < p.Threads; i++ {
		channels = append(channels, make(chan [][]byte))
	}

	for i := 0; i < p.Threads; i++ {
		end := hInc * (i + 1)
		if i == p.Threads-1 {
			end = p.ImageHeight
		}
		go worker(turn, hInc*i, end, p, world, channels[i], events)
	}

	for i := 0; i < p.Threads; i++ {
		newStates = append(newStates, <-channels[i]...)
	}

	out <- newStates

}

// creates the world by taking in a file
func makeWorld(p Params, c distributorChannels) [][]byte {
	world := [][]byte{}
	c.ioCommand <- ioInput
	c.ioFilename <- strconv.Itoa(p.ImageHeight) + "x" + strconv.Itoa(p.ImageWidth)
	for y := 0; y < p.ImageHeight; y++ {
		world = append(world, []byte{})
		for x := 0; x < p.ImageWidth; x++ {
			world[y] = append(world[y], <-c.ioInput)
			if world[y][x] == 255 {
				turn := 0
				flipCell(turn, y, x, c.events)
			}
		}
	}
	return world
}

// creates Pgm file for current state of the board. Is always run after the final turn is finished.
func createPgm(p Params, turn int, world [][]byte, c distributorChannels) {
	filename := strconv.Itoa(p.ImageHeight) + "x" + strconv.Itoa(p.ImageWidth) + "x" + strconv.Itoa(turn)
	c.ioCommand <- ioOutput
	c.ioFilename <- filename
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			c.ioOutput <- world[y][x]
		}

	}
	c.events <- ImageOutputComplete{turn, filename}
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {

	world := makeWorld(p, c)
	turn := 0
	ticker := time.NewTicker(2000 * time.Millisecond)
	paused := false
loop:
	for turn < p.Turns {
		if !paused {
			out := make(chan [][]byte)
			go execute(turn, p, world, out, c.events)
			select {
			case <-ticker.C:
				cellsAlive := len(getAliveCells(p, world))
				c.events <- AliveCellsCount{turn, cellsAlive}
				world = <-out
				turn++
				c.events <- TurnComplete{turn}
			case key := <-c.keyPresses:
				if key == 's' {
					createPgm(p, turn, world, c)
				} else if key == 'q' {
					c.events <- StateChange{turn, Quitting}
					break loop

				} else if key == 'p' {
					if !paused {
						paused = true
						c.events <- StateChange{turn, Paused}
					}
				}
			default:
				world = <-out
				turn++
				c.events <- TurnComplete{turn}
			}
		} else {
			key := <-c.keyPresses
			if key == 'p' {
				fmt.Println("Continuing")
				paused = false
				c.events <- StateChange{turn, Executing}
			}
		}
	}
	ticker.Stop()
	createPgm(p, turn, world, c)

	c.events <- FinalTurnComplete{turn, getAliveCells(p, world)}

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{turn, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}