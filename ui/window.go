package ui

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"ffvi_editor/browser"
	"ffvi_editor/global"
	"ffvi_editor/io/config"
	"ffvi_editor/io/pr"
	"ffvi_editor/ui/forms"
	"ffvi_editor/ui/forms/selections"

	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
)

type (
	Gui interface {
		App() fyne.App
		Window() fyne.Window
		Canvas() *fyne.Container
		Load()
		Save()
		Run()
	}
	gui struct {
		app             fyne.App
		window          fyne.Window
		canvas          *fyne.Container
		open            *fyne.MenuItem
		save            *fyne.MenuItem
		saveState       *fyne.MenuItem
		loadState       *fyne.MenuItem
		quickSaveStates []*fyne.MenuItem
		quickLoadStates []*fyne.MenuItem
		prev            fyne.CanvasObject
		pr              *pr.PR
		background      *fyne.Container
	}
	MenuItem interface {
		Item() *fyne.MenuItem
		Disable()
		Enable()
		OnSelected(func())
	}
	menuItem struct {
		*fyne.MenuItem
		onSelected func()
	}
)

var (
	_ MenuItem = menuItem{}
)

func New() Gui {
	if wd, err := os.Getwd(); err == nil {
		var dir []os.DirEntry
		if dir, err = os.ReadDir(wd); err == nil {
			for _, f := range dir {
				if !f.IsDir() && strings.HasSuffix(f.Name(), ".ttf") {
					_ = os.Setenv("FYNE_FONT", f.Name())
					break
				}
			}
		}
	}
	var (
		a = app.New()
		g = &gui{
			app:    a,
			window: a.NewWindow(fmt.Sprintf("Final Fantasy VI Save Editor - v%s", browser.Version)),
			canvas: container.NewStack(),
		}
	)

	// Load background image
	if bgImg, err := fyne.LoadResourceFromPath("ChatGPT Image Jan 27, 2026, 08_28_25 PM.png"); err == nil {
		img := canvas.NewImageFromResource(bgImg)
		img.FillMode = canvas.ImageFillStretch

		// Create semi-transparent overlay for better text readability
		overlay := canvas.NewRectangle(color.NRGBA{R: 0, G: 0, B: 0, A: 128}) // Semi-transparent black

		g.background = container.NewStack(
			img,
			overlay,
			g.canvas,
		)
	} else {
		g.background = g.canvas
	}
	g.window.SetIcon(fyne.NewStaticResource("icon", resourceIconIco.StaticContent))
	g.window.SetContent(g.background)
	g.open = fyne.NewMenuItem("Open", func() {
		g.Load()
	})
	g.save = fyne.NewMenuItem("Save", func() {
		g.Save()
	})
	g.save.Disabled = true
	g.saveState = fyne.NewMenuItem("Save State", func() {
		g.SaveState()
	})
	g.saveState.Disabled = true
	g.loadState = fyne.NewMenuItem("Load State", func() {
		g.LoadState()
	})
	g.loadState.Disabled = true
	// Initialize quick save/load states
	g.quickSaveStates = make([]*fyne.MenuItem, 5)
	g.quickLoadStates = make([]*fyne.MenuItem, 5)
	for i := 0; i < 5; i++ {
		num := i + 1
		g.quickSaveStates[i] = fyne.NewMenuItem(fmt.Sprintf("Quick Save %d", num), func() {
			g.QuickSaveState(num)
		})
		g.quickSaveStates[i].Disabled = true
		g.quickLoadStates[i] = fyne.NewMenuItem(fmt.Sprintf("Quick Load %d", num), func() {
			g.QuickLoadState(num)
		})
		g.quickLoadStates[i].Disabled = true
	}
	x, y := config.WindowSize()
	g.window.Resize(fyne.NewSize(x, y))
	g.window.SetFixedSize(false)
	quickStatesMenu := fyne.NewMenuItem("Quick States", func() {})
	quickStatesMenu.ChildMenu = fyne.NewMenu("", append(g.quickSaveStates, g.quickLoadStates...)...)
	g.window.SetMainMenu(fyne.NewMainMenu(
		fyne.NewMenu("File",
			g.open,
			fyne.NewMenuItemSeparator(),
			g.save,
			fyne.NewMenuItemSeparator(),
			g.saveState,
			g.loadState,
			fyne.NewMenuItemSeparator(),
			quickStatesMenu,
		)))
	return g
}

func (g *gui) App() fyne.App {
	return g.app
}

func (g *gui) Window() fyne.Window {
	return g.window
}

func (g *gui) Canvas() *fyne.Container {
	return g.canvas
}

func (g *gui) Load() {
	if len(g.canvas.Objects) > 0 {
		g.prev = g.canvas.Objects[0]
	}
	g.open.Disabled = true
	g.canvas.RemoveAll()
	g.canvas.Add(
		forms.NewFileIO(forms.Load, g.window, config.SaveDir(), func(name, dir, file string, _ int, saveType global.SaveFileType) {
			defer func() { g.open.Disabled = false }()
			// Load file
			config.SetSaveDir(dir)
			p := pr.New()
			if err := p.Load(filepath.Join(dir, file), saveType); err != nil {
				if g.prev != nil {
					g.canvas.RemoveAll()
					g.canvas.Add(g.prev)
					g.window.Content().Refresh()
				}
				dialog.NewError(err, g.window).Show()
			} else {
				// Success
				g.canvas.RemoveAll()
				g.prev = nil
				g.save.Disabled = false
				g.saveState.Disabled = false
				g.loadState.Disabled = false
				for _, item := range g.quickSaveStates {
					item.Disabled = false
				}
				for _, item := range g.quickLoadStates {
					item.Disabled = false
				}
				g.pr = p
				g.canvas.Add(selections.NewEditor())
				g.window.Content().Refresh()
			}
		}, func() {
			defer func() { g.open.Disabled = false }()
			// Cancel
			if g.prev != nil {
				g.canvas.RemoveAll()
				g.canvas.Add(g.prev)
				g.window.Content().Refresh()
			}
		}))
}

func (g *gui) Save() {
	if len(g.canvas.Objects) > 0 {
		g.prev = g.canvas.Objects[0]
	}
	g.open.Disabled = true
	g.save.Disabled = true
	g.canvas.RemoveAll()
	g.canvas.Add(
		forms.NewFileIO(forms.Save, g.window, config.SaveDir(), func(name, dir, file string, slot int, saveType global.SaveFileType) {
			defer func() {
				g.open.Disabled = false
				g.save.Disabled = false
			}()
			// Save file
			config.SetSaveDir(dir)
			if err := g.pr.Save(slot, filepath.Join(dir, file), saveType); err != nil {
				if g.prev != nil {
					g.canvas.RemoveAll()
					g.canvas.Add(g.prev)
					g.window.Content().Refresh()
				}
				dialog.NewError(err, g.window).Show()
			} else {
				// Success
				if g.prev != nil {
					g.canvas.RemoveAll()
					g.canvas.Add(g.prev)
					g.window.Content().Refresh()
				}
			}
		}, func() {
			defer func() {
				g.open.Disabled = false
				g.save.Disabled = false
			}()
			// Cancel
			if g.prev != nil {
				g.canvas.RemoveAll()
				g.canvas.Add(g.prev)
				g.window.Content().Refresh()
			}
		}))
}

func (g *gui) Run() {
	g.window.ShowAndRun()
}

func (g *gui) SaveState() {
	if g.pr == nil {
		return
	}
	// Open file dialog to save state
	dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
		if err != nil || writer == nil {
			return
		}
		defer writer.Close()
		// Serialize PR to JSON
		data, err := json.Marshal(g.pr)
		if err != nil {
			dialog.NewError(err, g.window).Show()
			return
		}
		_, err = writer.Write(data)
		if err != nil {
			dialog.NewError(err, g.window).Show()
		}
	}, g.window).Show()
}

func (g *gui) LoadState() {
	if g.pr == nil {
		return
	}
	// Open file dialog to load state
	dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil || reader == nil {
			return
		}
		defer reader.Close()
		// Deserialize JSON to PR
		data, err := io.ReadAll(reader)
		if err != nil {
			dialog.NewError(err, g.window).Show()
			return
		}
		var newPR pr.PR
		if err := json.Unmarshal(data, &newPR); err != nil {
			dialog.NewError(err, g.window).Show()
			return
		}
		// Update current PR
		*g.pr = newPR
		// Refresh UI
		g.window.Content().Refresh()
	}, g.window).Show()
}

func (g *gui) QuickSaveState(slot int) {
	if g.pr == nil {
		return
	}
	filename := fmt.Sprintf("quickstate%d.json", slot)
	filepath := filepath.Join(config.SaveDir(), filename)
	data, err := json.Marshal(g.pr)
	if err != nil {
		dialog.NewError(err, g.window).Show()
		return
	}
	if err := os.WriteFile(filepath, data, 0644); err != nil {
		dialog.NewError(err, g.window).Show()
	}
}

func (g *gui) QuickLoadState(slot int) {
	if g.pr == nil {
		return
	}
	filename := fmt.Sprintf("quickstate%d.json", slot)
	filepath := filepath.Join(config.SaveDir(), filename)
	data, err := os.ReadFile(filepath)
	if err != nil {
		if os.IsNotExist(err) {
			dialog.NewInformation("No State", "No quick state saved for this slot.", g.window).Show()
		} else {
			dialog.NewError(err, g.window).Show()
		}
		return
	}
	var newPR pr.PR
	if err := json.Unmarshal(data, &newPR); err != nil {
		dialog.NewError(err, g.window).Show()
		return
	}
	// Update current PR
	*g.pr = newPR
	// Refresh UI
	g.window.Content().Refresh()
}

func (m menuItem) Item() *fyne.MenuItem {
	return m.MenuItem
}

func (m menuItem) Disable() {
	m.MenuItem.Disabled = true
}

func (m menuItem) Enable() {
	m.MenuItem.Disabled = false
}

func (m menuItem) OnSelected(f func()) {
	m.onSelected = f
}
