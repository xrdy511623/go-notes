package main

import (
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
	"runtime/trace"
	"sync"
)

const (
	output     = "out.png"
	width      = 2048
	height     = 2048
	numWorkers = 8
)

func main() {
	trace.Start(os.Stdout)
	defer trace.Stop()

	f, err := os.Create(output)
	if err != nil {
		log.Fatal(err)
	}

	img := createColumnParallel(width, height)

	if err = png.Encode(f, img); err != nil {
		log.Fatal(err)
	}
}

func createColumnParallel(width, height int) image.Image {
	m := image.NewGray(image.Rect(0, 0, width, height))
	var wg sync.WaitGroup
	wg.Add(width) // 为每一列启动一个goroutine
	for i := 0; i < width; i++ {
		go func(colIdx int) {
			defer wg.Done()
			for j := 0; j < height; j++ { // 该goroutine负责计算一整列
				m.Set(colIdx, j, pixel(colIdx, j, width, height))
			}
		}(i)
	}
	wg.Wait()
	return m
}

// pixel returns the color of a Mandelbrot fractal at the given point.
func pixel(i, j, width, height int) color.Color {
	// Play with this constant to increase the complexity of the fractal.
	// In the justforfunc.com video this was set to 4.
	const complexity = 1024

	xi := norm(i, width, -1.0, 2)
	yi := norm(j, height, -1, 1)

	const maxI = 1000
	x, y := 0., 0.

	for i := 0; (x*x+y*y < complexity) && i < maxI; i++ {
		x, y = x*x-y*y+xi, 2*x*y+yi
	}

	return color.Gray{uint8(x)}
}

func norm(x, total int, min, max float64) float64 {
	return (max-min)*float64(x)/float64(total) - max
}
