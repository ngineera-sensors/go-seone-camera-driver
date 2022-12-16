package fspdriver

type GridNode struct {
	X   int
	Y   int
	Row int
	Col int
}

type Frame struct {
	I         int
	Timestamp int
	Values    []float64
}
