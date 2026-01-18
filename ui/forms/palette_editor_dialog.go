package forms

import (
	"fmt"
	"image/color"

	"ffvi_editor/io"
	"ffvi_editor/models"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// PaletteEditorDialog provides a professional palette editor UI
type PaletteEditorDialog struct {
	window fyne.Window
	dialog dialog.Dialog

	// Palette display
	paletteGrid      *fyne.Container
	selectedColorIdx int
	colorBoxes       []*canvas.Rectangle

	// Color editor
	colorDisplay *canvas.Rectangle
	hexEntry     *widget.Entry
	rgbLabel     *widget.Label
	redSlider    *widget.Slider
	greenSlider  *widget.Slider
	blueSlider   *widget.Slider

	// Harmony generator
	harmonySelect *widget.Select
	generateBtn   *widget.Button

	// Transformations
	transformSelect *widget.Select
	transformBtn    *widget.Button

	// Preview
	previewContainer *fyne.Container

	// Buttons
	applyBtn  *widget.Button
	revertBtn *widget.Button
	exportBtn *widget.Button
	closeBtn  *widget.Button

	// State
	palette         *models.Palette
	originalPalette *models.Palette
	editor          *io.PaletteEditor
	onApply         func(*models.Palette)
}

// NewPaletteEditorDialog creates a new palette editor dialog
func NewPaletteEditorDialog(window fyne.Window, palette *models.Palette) *PaletteEditorDialog {
	if palette == nil {
		return nil
	}

	working := palette.Clone()
	if working == nil {
		return nil
	}

	p := &PaletteEditorDialog{
		window:           window,
		palette:          working,
		editor:           &io.PaletteEditor{},
		selectedColorIdx: 0,
	}

	// Make a copy of the original palette for revert
	p.originalPalette = working.Clone()

	p.buildUI()
	return p
}

// buildUI constructs the complete dialog UI
func (p *PaletteEditorDialog) buildUI() {
	// Palette grid (16 colors)
	p.paletteGrid = container.New(layout.NewGridLayout(4))
	p.colorBoxes = make([]*canvas.Rectangle, 16)

	for i := 0; i < 16; i++ {
		idx := i
		rect := canvas.NewRectangle(p.getRGB888Color(i))
		rect.SetMinSize(fyne.NewSize(48, 48))

		swatch := newColorSwatch(rect, func() {
			p.selectColor(idx)
		})

		p.colorBoxes[i] = rect
		p.paletteGrid.Add(swatch)
	}

	// Color editor section
	p.colorDisplay = canvas.NewRectangle(p.getRGB888Color(0))
	p.colorDisplay.SetMinSize(fyne.NewSize(80, 80))

	p.hexEntry = widget.NewEntry()
	p.hexEntry.SetPlaceHolder("#RRGGBB")
	p.hexEntry.OnChanged = func(s string) {
		p.updateColorFromHex(s)
	}

	p.rgbLabel = widget.NewLabel("RGB: 255, 255, 255")

	p.redSlider = widget.NewSlider(0, 31)
	p.redSlider.OnChanged = func(f float64) {
		p.updateFromSliders()
	}

	p.greenSlider = widget.NewSlider(0, 31)
	p.greenSlider.OnChanged = func(f float64) {
		p.updateFromSliders()
	}

	p.blueSlider = widget.NewSlider(0, 31)
	p.blueSlider.OnChanged = func(f float64) {
		p.updateFromSliders()
	}

	colorEditorBox := container.NewVBox(
		widget.NewLabel("Color Editor"),
		container.NewHBox(
			p.colorDisplay,
			container.NewVBox(
				p.hexEntry,
				p.rgbLabel,
			),
		),
		widget.NewSeparator(),
		container.NewBorder(nil, nil, widget.NewLabel("Red:"), nil, p.redSlider),
		container.NewBorder(nil, nil, widget.NewLabel("Green:"), nil, p.greenSlider),
		container.NewBorder(nil, nil, widget.NewLabel("Blue:"), nil, p.blueSlider),
	)

	// Harmony generator section
	p.harmonySelect = widget.NewSelect(
		[]string{
			"Complementary",
			"Triadic",
			"Analogous",
			"Monochromatic",
			"Split-Complementary",
			"Tetradic",
		},
		func(val string) {},
	)
	p.harmonySelect.SetSelected("Complementary")

	p.generateBtn = widget.NewButton("Generate Harmony", func() {
		p.generateHarmony()
	})

	harmonyBox := container.NewVBox(
		widget.NewLabel("Generate Color Harmony"),
		container.NewBorder(nil, nil, widget.NewLabel("Scheme:"), nil, p.harmonySelect),
		p.generateBtn,
	)

	// Transformations section
	p.transformSelect = widget.NewSelect(
		[]string{
			"Brighten",
			"Darken",
			"Saturate",
			"Desaturate",
			"Shift Hue",
			"Invert",
			"Grayscale",
			"Sepia",
		},
		func(val string) {},
	)
	p.transformSelect.SetSelected("Brighten")

	p.transformBtn = widget.NewButton("Apply Transform", func() {
		p.applyTransform()
	})

	transformBox := container.NewVBox(
		widget.NewLabel("Transform Colors"),
		container.NewBorder(nil, nil, widget.NewLabel("Effect:"), nil, p.transformSelect),
		p.transformBtn,
	)

	// Preview section
	p.previewContainer = container.NewVBox(
		widget.NewLabel("Preview"),
		p.createPaletteGrid(),
	)

	// Left side: palette grid + controls
	leftBox := container.NewVBox(
		widget.NewLabel("Palette (click to select)"),
		p.paletteGrid,
		widget.NewSeparator(),
		colorEditorBox,
		widget.NewSeparator(),
		harmonyBox,
		widget.NewSeparator(),
		transformBox,
	)

	leftScroll := container.NewVScroll(leftBox)
	leftScroll.SetMinSize(fyne.NewSize(400, 500))

	// Right side: preview
	rightScroll := container.NewVScroll(p.previewContainer)
	rightScroll.SetMinSize(fyne.NewSize(300, 500))

	// Main content
	contentBox := container.NewHBox(leftScroll, rightScroll)

	// Buttons
	p.applyBtn = widget.NewButton("Apply", func() {
		p.dialogApply()
	})
	p.applyBtn.Importance = widget.HighImportance

	p.revertBtn = widget.NewButton("Revert", func() {
		p.revert()
	})

	p.exportBtn = widget.NewButton("Export...", func() {
		// Future: export palette
	})

	p.closeBtn = widget.NewButton("Close", func() {
		p.dialog.Hide()
	})

	buttons := container.NewHBox(
		layout.NewSpacer(),
		p.exportBtn,
		p.revertBtn,
		p.closeBtn,
		p.applyBtn,
	)

	// Main dialog content
	mainContent := container.NewBorder(
		nil,
		buttons,
		nil,
		nil,
		contentBox,
	)

	p.dialog = dialog.NewCustom(
		"Palette Editor",
		"Close",
		WrapDialogWithDarkBackground(mainContent),
		p.window,
	)
	p.dialog.Resize(fyne.NewSize(800, 650))

	// Initialize color editor
	p.selectColor(0)
}

// Show displays the palette editor dialog
func (p *PaletteEditorDialog) Show() {
	p.dialog.Show()
}

// Hide hides the palette editor dialog
func (p *PaletteEditorDialog) Hide() {
	p.dialog.Hide()
}

// OnApply sets the callback for when palette is applied
func (p *PaletteEditorDialog) OnApply(fn func(*models.Palette)) {
	p.onApply = fn
}

// selectColor selects a color in the palette
func (p *PaletteEditorDialog) selectColor(idx int) {
	p.selectedColorIdx = idx

	// Update color display
	rgb888 := p.getRGB888Color(idx)
	p.colorDisplay.FillColor = rgb888
	p.colorDisplay.Refresh()

	// Update sliders
	rgb555 := p.palette.Colors[idx]
	r8, g8, b8 := rgb555.ToRGB888()

	p.redSlider.SetValue(float64(rgb555.R))
	p.greenSlider.SetValue(float64(rgb555.G))
	p.blueSlider.SetValue(float64(rgb555.B))

	// Update hex and labels
	p.hexEntry.SetText(fmt.Sprintf("#%02X%02X%02X", r8, g8, b8))
	p.rgbLabel.SetText(fmt.Sprintf("RGB: %d, %d, %d", r8, g8, b8))

	// Highlight selected color
	for i, box := range p.colorBoxes {
		if i == idx {
			box.StrokeColor = color.White
			box.StrokeWidth = 3
		} else {
			box.StrokeColor = color.Black
			box.StrokeWidth = 1
		}
		box.Refresh()
	}

	p.updatePreview()
}

// getRGB888Color converts RGB555 to RGB888 for display
func (p *PaletteEditorDialog) getRGB888Color(idx int) color.Color {
	if idx < 0 || idx >= len(p.palette.Colors) {
		return color.White
	}

	r8, g8, b8 := p.palette.Colors[idx].ToRGB888()

	return color.RGBA{R: r8, G: g8, B: b8, A: 255}
}

// updateFromSliders updates color from slider values
func (p *PaletteEditorDialog) updateFromSliders() {
	r := uint8(p.redSlider.Value)
	g := uint8(p.greenSlider.Value)
	b := uint8(p.blueSlider.Value)

	p.palette.Colors[p.selectedColorIdx] = models.RGB555{R: r, G: g, B: b}

	// Update display
	rgb888 := p.getRGB888Color(p.selectedColorIdx)
	p.colorDisplay.FillColor = rgb888
	p.colorDisplay.Refresh()

	r8, g8, b8 := p.palette.Colors[p.selectedColorIdx].ToRGB888()
	p.hexEntry.SetText(fmt.Sprintf("#%02X%02X%02X", r8, g8, b8))
	p.updatePreview()
}

// updateColorFromHex updates color from hex input
func (p *PaletteEditorDialog) updateColorFromHex(hexStr string) {
	if hexStr == "" {
		return
	}

	var rgb uint32
	_, err := fmt.Sscanf(hexStr, "#%06X", &rgb)
	if err != nil {
		return
	}

	r8 := uint8(rgb >> 16)
	g8 := uint8((rgb >> 8) & 0xFF)
	b8 := uint8(rgb & 0xFF)

	color555 := models.FromRGB888(r8, g8, b8)
	p.palette.Colors[p.selectedColorIdx] = color555

	p.redSlider.SetValue(float64(color555.R))
	p.greenSlider.SetValue(float64(color555.G))
	p.blueSlider.SetValue(float64(color555.B))

	p.colorDisplay.FillColor = p.getRGB888Color(p.selectedColorIdx)
	p.colorDisplay.Refresh()

	p.updatePreview()
}

// generateHarmony generates a harmony palette from selected color
func (p *PaletteEditorDialog) generateHarmony() {
	baseColor := p.palette.Colors[p.selectedColorIdx]
	scheme := mapHarmonyScheme(p.harmonySelect.Selected)

	// Use backend harmonizer
	harmonizer := io.NewColorHarmonizer()
	generated := harmonizer.Generate(baseColor, scheme)
	if generated != nil {
		p.palette.Colors = generated.Colors
	}

	// Refresh UI
	p.rebuildPaletteGrid()
	p.updatePreview()
}

// applyTransform applies a color transformation to the entire palette
func (p *PaletteEditorDialog) applyTransform() {
	transform := mapTransform(p.transformSelect.Selected)
	transformer := io.NewColorTransformer()
	amount := 0.5

	for i := range p.palette.Colors {
		p.palette.Colors[i] = transformer.Apply(p.palette.Colors[i], transform, amount)
	}

	p.rebuildPaletteGrid()
	p.updatePreview()
}

// rebuildPaletteGrid rebuilds the color grid display
func (p *PaletteEditorDialog) rebuildPaletteGrid() {
	for i, rect := range p.colorBoxes {
		rect.FillColor = p.getRGB888Color(i)
		rect.Refresh()
	}
}

// createPaletteGrid creates a visual grid of the palette
func (p *PaletteEditorDialog) createPaletteGrid() *fyne.Container {
	grid := container.New(layout.NewGridLayout(4))
	for i := 0; i < 16; i++ {
		rect := canvas.NewRectangle(p.getRGB888Color(i))
		rect.SetMinSize(fyne.NewSize(40, 40))
		grid.Add(rect)
	}
	return grid
}

// updatePreview updates the preview
func (p *PaletteEditorDialog) updatePreview() {
	p.previewContainer.RemoveAll()
	p.previewContainer.Add(widget.NewLabel("Preview"))
	p.previewContainer.Add(p.createPaletteGrid())
	p.previewContainer.Refresh()
}

// revert reverts to the original palette
func (p *PaletteEditorDialog) revert() {
	p.palette.Colors = p.originalPalette.Colors
	p.rebuildPaletteGrid()
	p.selectColor(0)
	p.updatePreview()
}

// dialogApply applies the palette changes
func (p *PaletteEditorDialog) dialogApply() {
	if p.onApply != nil {
		p.onApply(p.palette)
	}
	p.dialog.Hide()
}

func mapHarmonyScheme(display string) string {
	switch display {
	case "Complementary":
		return io.HarmonyComplementary
	case "Triadic":
		return io.HarmonyTriadic
	case "Analogous":
		return io.HarmonyAnalogous
	case "Monochromatic":
		return io.HarmonyMonochromatic
	case "Split-Complementary":
		return io.HarmonySplitComplementary
	case "Tetradic":
		return io.HarmonyTetradic
	default:
		return io.HarmonyMonochromatic
	}
}

func mapTransform(display string) string {
	switch display {
	case "Brighten":
		return "brighten"
	case "Darken":
		return "darken"
	case "Saturate":
		return "saturate"
	case "Desaturate":
		return "desaturate"
	case "Shift Hue":
		return "shift-hue"
	case "Invert":
		return "invert"
	case "Grayscale":
		return "grayscale"
	case "Sepia":
		return "sepia"
	default:
		return "brighten"
	}
}

// colorSwatch is a clickable rectangle used for palette selection.
type colorSwatch struct {
	widget.BaseWidget
	rect  *canvas.Rectangle
	onTap func()
}

func newColorSwatch(rect *canvas.Rectangle, onTap func()) *colorSwatch {
	c := &colorSwatch{
		rect:  rect,
		onTap: onTap,
	}
	c.ExtendBaseWidget(c)
	return c
}

func (c *colorSwatch) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(c.rect)
}

func (c *colorSwatch) Tapped(_ *fyne.PointEvent) {
	if c.onTap != nil {
		c.onTap()
	}
}
