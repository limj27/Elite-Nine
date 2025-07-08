package main

import (
	"fmt"
	"strings"
)

func visualExplanation() {
	fmt.Println("=== WHAT ARE WE ACTUALLY COUNTING? ===")

	mat := [][]int{
		{1, 0, 1},
		{1, 1, 0},
		{1, 1, 0},
	}

	fmt.Println("Original matrix:")
	printMatrix(mat)

	fmt.Println("\nWe want to count ALL possible submatrices that contain only 1s")
	fmt.Println("Let's manually identify them:")

	// Let's manually count to understand what we're looking for
	count := 0
	fmt.Println("\n1x1 submatrices (single cells with 1):")
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			if mat[i][j] == 1 {
				count++
				fmt.Printf("  [%d,%d] = 1\n", i, j)
			}
		}
	}
	fmt.Printf("Total 1x1: %d\n", count-6) // Reset for next count
	count = 6

	fmt.Println("\n2x1 submatrices (2 rows, 1 column):")
	// Check all possible 2x1 rectangles
	for i := 0; i < 2; i++ { // Can start at row 0 or 1 (not 2, because we need 2 rows)
		for j := 0; j < 3; j++ {
			if mat[i][j] == 1 && mat[i+1][j] == 1 {
				count++
				fmt.Printf("  Rows %d-%d, Col %d: [%d,%d],[%d,%d]\n", i, i+1, j, i, j, i+1, j)
			}
		}
	}

	fmt.Println("\n1x2 submatrices (1 row, 2 columns):")
	for i := 0; i < 3; i++ {
		for j := 0; j < 2; j++ { // Can start at col 0 or 1
			if mat[i][j] == 1 && mat[i][j+1] == 1 {
				count++
				fmt.Printf("  Row %d, Cols %d-%d: [%d,%d],[%d,%d]\n", i, j, j+1, i, j, i, j+1)
			}
		}
	}

	fmt.Println("\n3x1 submatrices (3 rows, 1 column):")
	for j := 0; j < 3; j++ {
		if mat[0][j] == 1 && mat[1][j] == 1 && mat[2][j] == 1 {
			count++
			fmt.Printf("  All rows, Col %d: [0,%d],[1,%d],[2,%d]\n", j, j, j, j)
		}
	}

	fmt.Println("\n2x2 submatrices:")
	for i := 0; i < 2; i++ {
		for j := 0; j < 2; j++ {
			if mat[i][j] == 1 && mat[i][j+1] == 1 &&
				mat[i+1][j] == 1 && mat[i+1][j+1] == 1 {
				count++
				fmt.Printf("  Rows %d-%d, Cols %d-%d\n", i, i+1, j, j+1)
			}
		}
	}

	fmt.Printf("\nTotal manually counted: %d\n", count)
}

func explainHistogramApproach() {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("=== NOW THE HISTOGRAM APPROACH ===")

	mat := [][]int{
		{1, 0, 1},
		{1, 1, 0},
		{1, 1, 0},
	}

	fmt.Println("\nKey insight: Process row by row, building 'histograms'")
	fmt.Println("heights[j] = how many consecutive 1s END at current row in column j")

	heights := make([]int, 3)
	total := 0

	for row := 0; row < 3; row++ {
		fmt.Printf("\n--- ROW %d ---\n", row)
		fmt.Printf("Current row: %v\n", mat[row])

		// Update heights
		for j := 0; j < 3; j++ {
			if mat[row][j] == 1 {
				heights[j]++
			} else {
				heights[j] = 0
			}
		}

		fmt.Printf("Heights: %v\n", heights)
		fmt.Println("This means:")
		for j := 0; j < 3; j++ {
			if heights[j] > 0 {
				fmt.Printf("  Column %d: %d consecutive 1s ending here\n", j, heights[j])
			}
		}

		// Now count rectangles in this histogram
		fmt.Println("\nCounting rectangles in this histogram:")
		rowTotal := 0

		for left := 0; left < 3; left++ {
			if heights[left] == 0 {
				continue
			}

			fmt.Printf("  Starting from column %d (height %d):\n", left, heights[left])
			minHeight := heights[left]

			for right := left; right < 3 && heights[right] > 0; right++ {
				minHeight = min(minHeight, heights[right])

				fmt.Printf("    Width from col %d to %d: minHeight=%d\n", left, right, minHeight)
				fmt.Printf("    This gives us %d rectangles of different heights (1 to %d)\n",
					minHeight, minHeight)

				// Show what rectangles these are
				for h := 1; h <= minHeight; h++ {
					fmt.Printf("      Rectangle: rows %d-%d, cols %d-%d\n",
						row-h+1, row, left, right)
				}

				rowTotal += minHeight
			}
		}

		fmt.Printf("Row %d total: %d\n", row, rowTotal)
		total += rowTotal
	}

	fmt.Printf("\nFinal total: %d\n", total)
}

func whyMinHeight() {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("=== WHY minHeight? ===")

	fmt.Println("Consider heights = [3, 2, 0] (what we have at row 2)")
	fmt.Println("We're looking at columns 0 and 1 (heights 3 and 2)")
	fmt.Println()
	fmt.Println("Visual representation:")
	fmt.Println("Col: 0 1 2")
	fmt.Println("     █ █ .")
	fmt.Println("     █ █ .")
	fmt.Println("     █ . .")
	fmt.Println()
	fmt.Println("If we want a rectangle spanning columns 0-1:")
	fmt.Println("- We can only go as tall as the SHORTEST column in that range")
	fmt.Println("- Column 0 has height 3, column 1 has height 2")
	fmt.Println("- So max rectangle height = min(3,2) = 2")
	fmt.Println()
	fmt.Println("We can make rectangles of height 1 and height 2:")
	fmt.Println("Height 1: bottom row only")
	fmt.Println("Height 2: bottom 2 rows")
	fmt.Println("That's why we add minHeight (2) to our count!")
}

func printMatrix(mat [][]int) {
	for _, row := range mat {
		fmt.Println(row)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func main() {
	visualExplanation()
	explainHistogramApproach()
	whyMinHeight()
}
