package main

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"

	"github.com/slonikmak/XRay-mini/internal/core"
)

type trayIconKind uint8

const (
	trayIconIdle trayIconKind = iota
	trayIconConnected
	trayIconWarning
)

func trayIconForStatus(status core.Status) trayIconKind {
	switch status.State {
	case "Error", "Degraded":
		return trayIconWarning
	case "Connecting", "Connected":
		if status.ProcessState == "running" && status.NetworkHealth == "unhealthy" {
			return trayIconWarning
		}
		if status.State == "Connected" && status.ProcessState == "running" && status.NetworkHealth == "healthy" {
			return trayIconConnected
		}
	}
	return trayIconIdle
}

// trayIcons preserves the app mark while changing its background to indicate
// connection health. The original mint icon is reserved for a verified proxy.
func trayIcons(source []byte) ([3][]byte, error) {
	var icons [3][]byte
	original, err := png.Decode(bytes.NewReader(source))
	if err != nil {
		return icons, err
	}
	icons[trayIconConnected] = source
	for kind, background := range map[trayIconKind]color.NRGBA{
		trayIconIdle:    {R: 151, G: 163, B: 177, A: 255},
		trayIconWarning: {R: 226, G: 189, B: 119, A: 255},
	} {
		img := image.NewNRGBA(original.Bounds())
		draw.Draw(img, img.Bounds(), original, original.Bounds().Min, draw.Src)
		for i := 0; i < len(img.Pix); i += 4 {
			// The source mark uses #102a25 on #81ebcd. Interpolate the
			// background change at antialiased edges while preserving alpha.
			weight := min(max(int(img.Pix[i+1])-42, 0), 193)
			img.Pix[i] = uint8(min(max(int(img.Pix[i])+(int(background.R)-129)*weight/193, 0), 255))
			img.Pix[i+1] = uint8(min(max(int(img.Pix[i+1])+(int(background.G)-235)*weight/193, 0), 255))
			img.Pix[i+2] = uint8(min(max(int(img.Pix[i+2])+(int(background.B)-205)*weight/193, 0), 255))
		}
		var encoded bytes.Buffer
		if err := png.Encode(&encoded, img); err != nil {
			return icons, err
		}
		icons[kind] = encoded.Bytes()
	}
	return icons, nil
}
