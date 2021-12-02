package stubs

var ProcessTurnsHandler = "GameOfLifeOperations.ProcessTurns"
var AliveCellCountHandler = "GameOfLifeOperations.AliveCellCount"
var KeyPressHandler = "GameOfLifeOperations.KeyPress"
var QuitHandler = "GameOfLifeOperations.Quit"
var ShutDownHandler = "GameOfLifeOperations.ShutDown"
var TurnHandler = "GameOfLifeOperations.ExecuteTurn"

type ProcessTurnsRequest struct {
	World       [][]byte
	Turns       int
	Threads     int
	ImageHeight int
	ImageWidth  int
}

type ProcessTurnsResponse struct {
	World [][]byte
	Turns int
}

type AliveCellCountRequest struct{}

type AliveCellsCountResponse struct {
	Turns      int
	CellsAlive int
}

type KeyPressRequest struct{ Key rune }

type KeyPressResponse struct {
	World [][]byte
	Turns int
}

type QuitRequest struct{}

type QuitResponse struct{}

type ShutDownRequest struct{}

type ShutDownResponse struct{}

type ExecuteTurnRequest struct {
	World       [][]byte
	ImageHeight int
	ImageWidth  int
}

type ExecuteTurnResponse struct{ World [][]byte }
