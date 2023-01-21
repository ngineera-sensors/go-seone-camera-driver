package fspdriver

import (
	"image"
	"math"

	"gocv.io/x/gocv"
)

const (
	MMI_EXTRACTION_ELLIPSE_RADIUS int = 8
)

// ExtractMMIsInefficient extracts luminence values out according to the grid.
// Value is defined as mean of all non-zero pixels inside the
// square patch with side size of MMI_EXTRACTION_ELLIPSE_RADIUS
// TODO: should be a circular patch, not square one
func ExtractMMIsInefficient(mat gocv.Mat, grid [MMI_N_NODES]GridNode) [MMI_N_NODES]float64 {
	var MMIs [MMI_N_NODES]float64

	for i, node := range grid {

		x0 := node.X - MMI_EXTRACTION_ELLIPSE_RADIUS
		if x0 < 0 {
			x0 = 0
		}
		y0 := node.Y - MMI_EXTRACTION_ELLIPSE_RADIUS
		if y0 < 0 {
			y0 = 0
		}
		x1 := node.X + MMI_EXTRACTION_ELLIPSE_RADIUS
		if x1 >= mat.Cols() {
			x1 = mat.Cols() - 1
		}
		y1 := node.Y + MMI_EXTRACTION_ELLIPSE_RADIUS
		if y1 >= mat.Rows() {
			y1 = mat.Rows() - 1
		}

		rect := image.Rect(
			x0, y0, x1, y1,
		)
		var mean float64
		roi := mat.Region(rect)
		nzCount := gocv.CountNonZero(roi)
		if nzCount > 0 {
			sum := roi.Sum()
			mean = sum.Val1 / float64(nzCount)
		} else {
			mean = 0
		}
		// log.Printf("MAT. I: %d; x0/y0: %d/%d; x1/y1: %d/%d; Count: %d; NZCount: %d; Mean: %f", i, x0, y0, x1, y1, rect.Size().X*rect.Size().Y, nzCount, mean)
		MMIs[i] = mean
		roi.Close()
	}
	return MMIs
}

func ExtractMMIsBuffer(buf []byte, grid [MMI_N_NODES]GridNode, darkValue byte) [MMI_N_NODES]float64 {
	var MMIs [MMI_N_NODES]float64

	for i, node := range grid {
		x0 := node.X - MMI_EXTRACTION_ELLIPSE_RADIUS
		if x0 < 0 {
			x0 = 0
		}
		y0 := node.Y - MMI_EXTRACTION_ELLIPSE_RADIUS
		if y0 < 0 {
			y0 = 0
		}
		x1 := node.X + MMI_EXTRACTION_ELLIPSE_RADIUS
		if x1 >= CAMERA_FRAME_WIDTH {
			x1 = CAMERA_FRAME_WIDTH - 1
		}
		y1 := node.Y + MMI_EXTRACTION_ELLIPSE_RADIUS
		if y1 >= CAMERA_FRAME_HEIGHT {
			y1 = CAMERA_FRAME_HEIGHT - 1
		}

		roiWidth := x1 - x0
		roiHeight := y1 - y0

		sum := 0
		count := 0
		nzCount := 0
		for roiRow := 0; roiRow < roiWidth; roiRow++ {
			for roiCol := 0; roiCol < roiHeight; roiCol++ {
				count++
				idx := (y0+roiRow)*CAMERA_FRAME_WIDTH + (x0 + roiCol)
				pixelValue := buf[idx]
				if pixelValue <= darkValue {
					continue
				}
				nzCount++
				sum += int(pixelValue)
			}
		}
		mean := .0
		if nzCount > 0 {
			mean = float64(sum) / float64(nzCount)
		}
		// log.Printf("BUF. I: %d; x0/y0: %d/%d; x1/y1: %d/%d; Count: %d; NZCount: %d; Mean: %f", i, x0, y0, x1, y1, count, nzCount, mean)
		MMIs[i] = mean
	}

	return MMIs
}

func ExtractMZIsInefficient(MMIs [MMI_N_NODES]float64, grid [MMI_N_NODES]GridNode) [MZI_N_NODES]float64 {
	var MZIs [MZI_N_NODES]float64
	for i, mziConfig := range MZI_MMI_GRID_MAP {
		a := mziConfig[0]
		b := mziConfig[1]
		c := mziConfig[2]

		var p1, p2, p3 float64

		// TODO: implement more efficient mmi grid search
		for j, mmi := range MMIs {
			gridNode := grid[j]
			if gridNode.Row == a[0] && gridNode.Col == a[1] {
				p1 = mmi
			}
			if gridNode.Row == b[0] && gridNode.Col == b[1] {
				p2 = mmi
			}
			if gridNode.Row == c[0] && gridNode.Col == c[1] {
				p3 = mmi
			}
			if p1 != 0 && p2 != 0 && p3 != 0 {
				break
			}
		}
		// log.Printf("STD. I: %d; Phasis values: %.2f, %.2f, %.2f", i, p1, p2, p3)

		it := 2*p2 - p1 - p3
		qt := math.Sqrt(3) * (p1 - p3)
		phase := -math.Atan2(qt, it)

		// log.Printf("It: %.2f; Qt: %.2f, dPh: %.2f\n", it, qt, phase)

		MZIs[i] = phase
	}

	return MZIs
}

func ExtractMZIsIndexed(MMIs [MMI_N_NODES]float64, grid [MMI_N_NODES]GridNode) [MZI_N_NODES]float64 {
	var MZIs [MZI_N_NODES]float64
	for i, mmiIndices := range MZI_MMI_INDICES_MAP {
		// TODO: check abc/cba order
		aIdx := mmiIndices[2]
		bIdx := mmiIndices[1]
		cIdx := mmiIndices[0]

		p1 := MMIs[aIdx]
		p2 := MMIs[bIdx]
		p3 := MMIs[cIdx]

		// log.Printf("IDX. I: %d; Phasis values: %.2f, %.2f, %.2f", i, p1, p2, p3)

		it := 2*p2 - p1 - p3
		qt := math.Sqrt(3) * (p1 - p3)
		phase := -math.Atan2(qt, it)

		// log.Printf("It: %.2f; Qt: %.2f, dPh: %.2f\n", it, qt, phase)

		MZIs[i] = phase
	}

	return MZIs
}
