package main

import (
	"context"
	"flag"
	"net"
	"bytes"
	"time"
	"image"
	"image/draw"
	"github.com/lmittmann/ppm"
	"github.com/mcuadros/go-rpi-rgb-led-matrix"
)

var (
	rows                     = flag.Int("led-rows", 16, "number of rows supported")
	cols                     = flag.Int("led-cols", 32, "number of columns supported")
	parallel                 = flag.Int("led-parallel", 1, "number of daisy-chained panels")
	chain                    = flag.Int("led-chain", 1, "number of displays daisy-chained")
	brightness               = flag.Int("brightness", 100, "brightness (0-100)")
	hardware_mapping         = flag.String("led-gpio-mapping", "regular", "Name of GPIO mapping used.")
	show_refresh             = flag.Bool("led-show-refresh", false, "Show refresh rate.")
	inverse_colors           = flag.Bool("led-inverse", false, "Switch if your matrix has inverse colors on.")
	disable_hardware_pulsing = flag.Bool("led-no-hardware-pulse", false, "Don't use hardware pin-pulse generation.")
)

func main() {
	ctx, _ := context.WithCancel(context.Background())

	go serve(ctx)

	<-ctx.Done()
}

func serve(ctx context.Context) (err error) {
	config := &rgbmatrix.DefaultConfig
	config.Rows = *rows
	config.Cols = *cols
	config.Parallel = *parallel
	config.ChainLength = *chain
	config.Brightness = *brightness
	config.HardwareMapping = *hardware_mapping
	config.ShowRefreshRate = *show_refresh
	config.InverseColors = *inverse_colors
	config.DisableHardwarePulsing = *disable_hardware_pulsing
	
	pc, err := net.ListenPacket("udp", ":1337")
	if err != nil {
		return
	}
	defer pc.Close()

	doneChan := make(chan error, 1)
	buffer := make([]byte, 65535)
	
	var canvas *rgbmatrix.Canvas = nil

	duration := time.Duration(5) * time.Second
	f := func() {
		canvas.Close()
		canvas = nil
	}
	timer := time.AfterFunc(duration, f)
	timer.Stop()

	go func() {
		for {
			n, _, err := pc.ReadFrom(buffer)
			if err != nil {
				doneChan <- err
				return
			}
			
			if canvas == nil {
				m, err := rgbmatrix.NewRGBLedMatrix(config)
				fatal(err)

				canvas = rgbmatrix.NewCanvas(m)
				defer canvas.Close()
			}
			
			timer.Reset(duration)
			
			img, err := ppm.Decode(bytes.NewReader(buffer[:n]))
			if err != nil {
				doneChan <- err
				return
			}
			
			draw.Draw(canvas, canvas.Bounds(), img, image.ZP, draw.Src)
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
