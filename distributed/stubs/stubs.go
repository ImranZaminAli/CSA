package stubs

var ProcessTurnsHandler = "GameOfLifeOperations.ProcessTurns"

type Request struct {
	World       [][]byte
	Turns       int
	Threads     int
	ImageWidth  int
	ImageHeight int
}

type Response struct {
	World [][]byte
	Turns int
}
