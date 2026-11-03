//go:build windows

package main

import (
	"fmt"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/w32"
)

func TestTrayIconsCreateNativeHandles(t *testing.T) {
	icons, err := trayIcons(trayIcon)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	for kind, data := range icons {
		handle, err := w32.CreateSmallHIconFromImage(data)
		if err != nil || handle == 0 {
			t.Fatalf("icon %d: Windows could not create tray icon: %v", kind, err)
		}
		path := filepath.Join(dir, fmt.Sprintf("%d.png", kind))
		if err := w32.SaveHIconAsPNG(handle, path); err != nil {
			w32.DestroyIcon(handle)
			t.Fatalf("icon %d: could not inspect native icon: %v", kind, err)
		}
		if !w32.DestroyIcon(handle) {
			t.Errorf("icon %d: Windows could not release tray icon", kind)
		}
		file, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		img, err := png.Decode(file)
		file.Close()
		if err != nil {
			t.Fatal(err)
		}
		width, height := img.Bounds().Dx(), img.Bounds().Dy()
		if width != w32.GetSystemMetrics(w32.SM_CXSMICON) || height != w32.GetSystemMetrics(w32.SM_CYSMICON) {
			t.Fatalf("icon %d: unexpected native size %dx%d", kind, width, height)
		}
		background := color.NRGBAModel.Convert(img.At(width/2, height/8)).(color.NRGBA)
		mark := color.NRGBAModel.Convert(img.At(width/2, height/2)).(color.NRGBA)
		t.Logf("icon %d: native %dx%d, background %v, mark %v", kind, width, height, background, mark)
		if mark.R > 80 || mark.G > 80 || mark.B > 80 || mark.A < 240 {
			t.Errorf("icon %d: mark is not visible at native size: %v", kind, mark)
		}
		switch trayIconKind(kind) {
		case trayIconIdle:
			if int(background.B)-int(background.R) < 10 {
				t.Errorf("idle icon is not gray at native size: %v", background)
			}
		case trayIconConnected:
			if int(background.G)-int(background.R) < 40 {
				t.Errorf("connected icon is not mint at native size: %v", background)
			}
		case trayIconWarning:
			if int(background.R)-int(background.B) < 40 {
				t.Errorf("warning icon is not amber at native size: %v", background)
			}
		}
	}
}
