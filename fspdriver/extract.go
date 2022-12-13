package fspdriver

import (
	"image"
	"math"

	"gocv.io/x/gocv"
)

// ExtractMMIs extracts luminence values out according to the grid.
// Value is defined as mean of all non-zero pixels inside the
// square patch with side size of MMI_EXTRACTION_ELLIPSE_RADIUS
// TODO: should be a circular patch, not square one
func ExtractMMIs(mat gocv.Mat, grid [MMI_N_NODES]GridNode) [MMI_N_NODES]float64 {
	var MMIs [MMI_N_NODES]float64

	for i, node := range grid {
		rect := image.Rect(
			node.X-MMI_EXTRACTION_ELLIPSE_RADIUS,
			node.Y-MMI_EXTRACTION_ELLIPSE_RADIUS,
			node.X+MMI_EXTRACTION_ELLIPSE_RADIUS,
			node.Y+MMI_EXTRACTION_ELLIPSE_RADIUS,
		)
		roi := mat.Region(rect)
		nzCnt := gocv.CountNonZero(roi)
		sum := roi.Sum()
		mean := sum.Val1 / float64(nzCnt)
		MMIs[i] = mean
	}
	return MMIs
}

func ExtractMZIs(MMIs [MMI_N_NODES]float64, grid [MMI_N_NODES]GridNode) [MZI_N_NODES]float64 {
	var MZIs [MZI_N_NODES]float64
	for i, mziConfig := range MZI_MMI_MAP {
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

		it := 2*p1 - p2 - p3
		qt := math.Sqrt(3) * (p1 - p3)

		phase := -math.Atan2(qt, it)
		MZIs[i] = phase
	}

	return MZIs
}
