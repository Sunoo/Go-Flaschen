package main

import (
	"context"
	"flag"
	"net"
	"bytes"
	"github.com/lmittmann/ppm"
	"github.com/mcuadros/go-rpi-rgb-led-matrix"
)

var (
	rows                     = flag.Int("led-rows", 16, "number of rows supported")
	cols                     = flag.Int("led-cols", 32, "number of columns supported")
	parallel                 = flag.Int("led-parallel", 1, "number of daisy-chained panels")
	chain                    = flag.Int("led-chain", 2, "number of displays daisy-chained")
	brightness               = flag.Int("brightness", 100, "brightness (0-100)")
	hardware_mapping         = flag.String("led-gpio-mapping", "regular", "Name of GPIO mapping used.")
	show_refresh             = flag.Bool("led-show-refresh", false, "Show refresh rate.")
	inverse_colors           = flag.Bool("led-inverse", false, "Switch if your matrix has inverse colors on.")
	disable_hardware_pulsing = flag.Bool("led-no-hardware-pulse", false, "Don't use hardware pin-pulse generation.")
)

func main() {
	config := &rgbmatrix.DefaultConfig
	config.Rows = *rows
	config.Cols = *cols / 2
	config.Parallel = *parallel
	config.ChainLength = *chain
	config.Brightness = *brightness
	config.HardwareMapping = *hardware_mapping
	config.ShowRefreshRate = *show_refresh
	config.InverseColors = *inverse_colors
	config.DisableHardwarePulsing = *disable_hardware_pulsing
	
	m, err := rgbmatrix.NewRGBLedMatrix(config)
	fatal(err)

	c := rgbmatrix.NewCanvas(m)
	defer c.Close()
	
	ctx, _ := context.WithCancel(context.Background())

	go serve(ctx, *c)

	<-ctx.Done()
}

func serve(ctx context.Context, canvas rgbmatrix.Canvas) (err error) {
	pc, err := net.ListenPacket("udp", ":1337")
	if err != nil {
		return
	}
	defer pc.Close()

	doneChan := make(chan error, 1)
	buffer := make([]byte, 65535)

	go func() {
		for {
			n, _, err := pc.ReadFrom(buffer)
			if err != nil {
				doneChan <- err
				return
			}
			
			image, err := ppm.Decode(bytes.NewReader(buffer[:n]))
			if err != nil {
				doneChan <- err
				return
			}
			
			bounds := image.Bounds()
    		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
        		for x := bounds.Min.X; x < bounds.Max.X; x++ {
        			canvas.Set(x, y, image.At(x, y))
        		}
    		}
    		canvas.Render()
		}
	}()

	select {
	case <-ctx.Done():
		err = ctx.Err()
	case err = <-doneChan:
	}

	return
}

func init() {
	flag.Parse()
}

func fatal(err error) {
	if err != nil {
		panic(err)
	}
}
