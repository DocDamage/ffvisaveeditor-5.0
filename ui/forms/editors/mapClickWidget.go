package editors

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"

	_ "image/jpeg"
	_ "image/png"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
)

type MapClickWidget struct {
	widget.BaseWidget
	img           *canvas.Image
	onClick       func(x, y int)
	onHover       func(x, y int)
	imgWidth      int
	imgHeight     int
	offsetX       float32
	offsetY       float32
	dragging      bool
	dragStart     fyne.Position
	lastOffsetX   float32
	lastOffsetY   float32
	markerX       float64 // in image pixel coordinates
	markerY       float64 // in image pixel coordinates
	showMarker    bool
	renderImgPos  fyne.Position
	renderImgSize fyne.Size
	minSize       fyne.Size
}

func NewMapClickWidget(imgPath string, onClick func(x, y int), onHover func(x, y int)) *MapClickWidget {
	img, width, height := loadMapImage(imgPath)
	fyneImg := canvas.NewImageFromImage(img)
	fyneImg.FillMode = canvas.ImageFillContain
	fyneImg.SetMinSize(fyne.NewSize(1, 1))

	w := &MapClickWidget{
		img:        fyneImg,
		onClick:    onClick,
		onHover:    onHover,
		imgWidth:   width,
		imgHeight:  height,
		offsetX:    0,
		offsetY:    0,
		markerX:    -1,
		markerY:    -1,
		showMarker: false,
		minSize:    fyne.NewSize(1, 1),
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *MapClickWidget) SetImagePath(imgPath string) {
	img, width, height := loadMapImage(imgPath)
	w.img.Image = img
	w.img.FillMode = canvas.ImageFillContain
	w.img.SetMinSize(w.minSize)
	w.imgWidth = width
	w.imgHeight = height
	w.Refresh()
}

func (w *MapClickWidget) MinSize() fyne.Size {
	if w.minSize.Width > 0 && w.minSize.Height > 0 {
		return w.minSize
	}
	return fyne.NewSize(1, 1)
}

func (w *MapClickWidget) SetMinSize(size fyne.Size) {
	w.minSize = size
	if w.img != nil {
		w.img.SetMinSize(size)
	}
	w.Refresh()
}

func (w *MapClickWidget) ResetView() {
	w.offsetX, w.offsetY = 0, 0
	w.lastOffsetX, w.lastOffsetY = 0, 0
	w.Refresh()
}

func (w *MapClickWidget) CreateRenderer() fyne.WidgetRenderer {
	marker := canvas.NewCircle(color.NRGBA{R: 255, G: 0, B: 0, A: 255})
	marker.StrokeColor = color.NRGBA{R: 255, G: 0, B: 0, A: 255}
	marker.StrokeWidth = 2
	marker.Resize(fyne.NewSize(12, 12))
	marker.Hide()

	return &mapClickWidgetRenderer{
		w:       w,
		objects: []fyne.CanvasObject{w.img, marker},
		marker:  marker,
	}
}

func (w *MapClickWidget) Tapped(ev *fyne.PointEvent) {
	if w.onClick == nil || w.renderImgSize.Width <= 0 || w.renderImgSize.Height <= 0 {
		return
	}
	px := ev.Position.X - w.renderImgPos.X
	py := ev.Position.Y - w.renderImgPos.Y
	if px < 0 || py < 0 || px > w.renderImgSize.Width || py > w.renderImgSize.Height {
		return // Click outside rendered image
	}
	x := int(px * float32(w.imgWidth) / w.renderImgSize.Width)
	y := int(py * float32(w.imgHeight) / w.renderImgSize.Height)
	w.onClick(x, y)
}

func (w *MapClickWidget) Dragged(ev *fyne.DragEvent) {
	// Zoom is disabled; avoid accidental panning that "loses" the map.
}

func (w *MapClickWidget) DragEnd() {
}

type mapClickWidgetRenderer struct {
	w       *MapClickWidget
	marker  *canvas.Circle
	objects []fyne.CanvasObject
}

func (r *mapClickWidgetRenderer) Layout(size fyne.Size) {
	if r.w.imgWidth <= 0 || r.w.imgHeight <= 0 || size.Width <= 0 || size.Height <= 0 {
		r.w.renderImgPos = fyne.NewPos(0, 0)
		r.w.renderImgSize = fyne.NewSize(0, 0)
		return
	}

	imgW := float32(r.w.imgWidth)
	imgH := float32(r.w.imgHeight)
	scaleX := size.Width / imgW
	scaleY := size.Height / imgH
	scale := scaleX
	if scaleY < scaleX {
		scale = scaleY
	}
	if scale <= 0 {
		scale = 1
	}

	drawW := imgW * scale
	drawH := imgH * scale
	baseX := (size.Width - drawW) / 2
	baseY := (size.Height - drawH) / 2

	imgPos := fyne.NewPos(baseX+r.w.offsetX, baseY+r.w.offsetY)
	imgSize := fyne.NewSize(drawW, drawH)
	r.w.renderImgPos = imgPos
	r.w.renderImgSize = imgSize

	r.w.img.Move(imgPos)
	r.w.img.Resize(imgSize)

	if r.w.showMarker && r.w.markerX >= 0 && r.w.markerY >= 0 {
		mx := imgPos.X + (float32(r.w.markerX)/imgW)*drawW - 6
		my := imgPos.Y + (float32(r.w.markerY)/imgH)*drawH - 6
		r.marker.Move(fyne.NewPos(mx, my))
		r.marker.Show()
	} else {
		r.marker.Hide()
	}
}

func (r *mapClickWidgetRenderer) MinSize() fyne.Size {
	return r.w.MinSize()
}

func (r *mapClickWidgetRenderer) Refresh() {
	r.Layout(r.w.Size())
	canvas.Refresh(r.w.img)
	canvas.Refresh(r.marker)
}

func (r *mapClickWidgetRenderer) Destroy() {}

func (r *mapClickWidgetRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

func (r *mapClickWidgetRenderer) BackgroundColor() color.Color {
	return color.Transparent
}

func resolveMapImagePath(imgPath string) string {
	if imgPath == "" {
		return imgPath
	}
	if filepath.IsAbs(imgPath) {
		return imgPath
	}

	exe, err := os.Executable()
	if err == nil && exe != "" {
		exeDir := filepath.Dir(exe)
		candidate := filepath.Join(exeDir, imgPath)
		if _, statErr := os.Stat(candidate); statErr == nil {
			return candidate
		}
	}

	if cwd, err := os.Getwd(); err == nil && cwd != "" {
		candidate := filepath.Join(cwd, imgPath)
		if _, statErr := os.Stat(candidate); statErr == nil {
			return candidate
		}
	}

	return imgPath
}

func loadMapImage(imgPath string) (image.Image, int, int) {
	absPath, _ := os.Getwd()
	resolvedPath := resolveMapImagePath(imgPath)
	fmt.Printf("[DEBUG] Attempting to open map image: %s (resolved: %s, cwd: %s)\n", imgPath, resolvedPath, absPath)

	file, err := os.Open(resolvedPath)
	if err != nil {
		fmt.Printf("[ERROR] Failed to open map image: %s (resolved: %s, cwd: %s): %v\n", imgPath, resolvedPath, absPath, err)
		blank := image.NewNRGBA(image.Rect(0, 0, 1, 1))
		return blank, 1, 1
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		fmt.Printf("[ERROR] Failed to decode map image: %s (resolved: %s, cwd: %s): %v\n", imgPath, resolvedPath, absPath, err)
		blank := image.NewNRGBA(image.Rect(0, 0, 1, 1))
		return blank, 1, 1
	}

	width := img.Bounds().Dx()
	height := img.Bounds().Dy()
	fmt.Printf("[DEBUG] Successfully loaded map image: %s (resolved: %s, %dx%d)\n", imgPath, resolvedPath, width, height)
	return img, width, height
}
