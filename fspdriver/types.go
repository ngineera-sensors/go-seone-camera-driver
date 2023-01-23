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

type CameraState byte

type CameraStateMessage struct {
	State CameraState
}

type CameraFramerateMessage struct {
	Framerate int
}

type CameraCalibrationMessage struct {
	TargetMaxValue        int
	EffectiveMaxValue     int
	EffectiveShutterSpeed int
	EffectiveDarkValue    byte
	EffectiveGrid         [MMI_N_NODES]GridNode
}
