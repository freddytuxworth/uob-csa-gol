package main

//
//func printGrid(grid Grid) {
//	output := ""
//	for y := 0; y < grid.height; y++ {
//		row := ""
//		for x := 0; x < grid.width; x++ {
//			if grid.cells[y][x] > 0 {
//				row += "#"
//			} else {
//				row += "."
//			}
//		}
//		output += fmt.Sprintf("%02d: %s\n", y, row)
//	}
//	fmt.Println(output)
//
//}
//
//func printJobI(n int, job Job) {
//	output := fmt.Sprintf("Thread %d got job:\n", n)
//	for y := 0; y < job.grid.height; y++ {
//		row := ""
//		for x := 0; x < job.grid.width; x++ {
//			 if job.grid.cells[y][x] > 0 {
//			 	row += "#"
//			 } else {
//			 	row += "."
//			 }
//		}
//		output += fmt.Sprintf("%d: %s\n", y - 1 + job.offsetY, row)
//	}
//	fmt.Println(output)
//}
//
//func isFlipped(x, y int, flips []util.Cell, offsetY int) bool {
//	for _, flip := range flips {
//		if flip.X == x && flip.Y-offsetY == y {
//			return true
//		}
//	}
//	return false
//}
//
//func printJob(n int, job Job, flips []util.Cell) {
//	output := fmt.Sprintf("Thread %d got job:\n", n)
//	for y := 0; y < job.grid.height; y++ {
//		row := ""
//		for x := 0; x < job.grid.width; x++ {
//			f := isFlipped(x, y, flips, job.offsetY)
//			if f {
//				row += "\u001b[31m"
//			}
//			if job.grid.cells[y][x] > 0 {
//				row += "#"
//			} else {
//				row += "."
//			}
//			if f {
//				row += "\u001b[0m"
//			}
//		}
//		output += fmt.Sprintf("%d: %s\n", y-1+job.offsetY, row)
//	}
//	fmt.Println(output)
//}
