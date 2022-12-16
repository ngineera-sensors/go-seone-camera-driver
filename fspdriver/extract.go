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
		nzCnt := gocv.CountNonZero(roi)
		if nzCnt > 0 {
			sum := roi.Sum()
			mean = sum.Val1 / float64(nzCnt)
		} else {
			mean = 0
		}
		// log.Println(nzCnt, sum, mean, roi.Mean().Val1)
		MMIs[i] = mean
		roi.Close()
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

		// Indexing: col-major, with interlacing rows
		// 12 - number of interlaced rows
		// row is int-divided by 2 because
		// the MZI_MMI_MAP uses deinterlaced row indexing
		// aIdx := a[1]*12 + a[0]/2
		// bIdx := b[1]*12 + b[0]/2
		// cIdx := c[1]*12 + c[0]/2
		// log.Printf("Extracting MZI #%d. A: %d (%v), B: %d (%v), C: %d (%v)",
		// 	i, aIdx, a, bIdx, b, cIdx, c,
		// )
		// p1 = MMIs[aIdx]
		// p2 = MMIs[bIdx]
		// p3 = MMIs[cIdx]

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
		// log.Printf("Phasis values: %.2f, %.2f, %.2f", p1, p2, p3)

		it := 2*p2 - p1 - p3
		qt := math.Sqrt(3) * (p1 - p3)
		phase := -math.Atan2(qt, it)

		// log.Printf("It: %.2f; Qt: %.2f, dPh: %.2f\n", it, qt, phase)

		MZIs[i] = phase
	}
	// panic("")

	return MZIs
}
