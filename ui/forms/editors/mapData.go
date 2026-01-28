package editors

import (
	"fmt"
	"image/color"
	"sort"
	"slices"
	"strings"

	"ffvi_editor/io/config"
	"ffvi_editor/models/consts"
	"ffvi_editor/models/pr"
	"ffvi_editor/ui/forms/inputs"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

type (
	MapData struct {
		widget.BaseWidget
		search  *widget.Entry
		results *widget.TextGrid
	}
)

type mapLocationItem struct {
	Name        string
	HasFallback bool
	X           float64
	Y           float64
}

func defaultWorldLocationNames(world int) []string {
	if world == 2 {
		return []string{
			"Solitary Island",
			"Albrook",
			"Tzen",
			"Mobliz",
			"Nikeah",
			"South Figaro",
			"Figaro Cave",
			"Figaro Castle",
			"Kohlingen",
			"Darill's Tomb",
			"Maranda",
			"Zozo",
			"Cave on the Veldt",
			"Thamasa",
			"Colosseum",
			"Jidoor",
			"Phoenix Cave",
			"Narshe",
			"Triangle Island",
			"Fanatics' Tower",
			"Doma Castle",
			"Duncan's House",
			"The Ancient Castle",
			"Gau's Father's House",
			"Opera House",
			"Kefka's Tower",
			"Ebot's Rock",
			"The Veldt",
		}
	}
	return []string{
		"Narshe",
		"Kohlingen",
		"Zozo",
		"Jidoor",
		"Opera House",
		"Maranda",
		"Vector",
		"Albrook",
		"Sealed Gate",
		"Tzen",
		"South Figaro",
		"Figaro Castle",
		"Figaro Cave",
		"Sabin's Cabin",
		"Mt. Koltz",
		"Returners' Hideout",
		"Nikea",
		"Doma Castle",
		"Imperial Base",
		"Phantom Forest",
		"Barren Falls",
		"House in the Veldt",
		"The Veldt",
		"Mobliz",
		"Crescent Mountain",
		"Espers' Gathering Place",
		"Thamasa",
	}
}

func NewMapData() *MapData {
	e := &MapData{
		search:  widget.NewEntry(),
		results: widget.NewTextGrid(),
	}

	e.search.OnChanged = func(s string) {
		s = strings.ToLower(s)
		if len(s) < 2 {
			e.results.SetText("")
			return
		}
		var sb strings.Builder
		if len(s) > 2 {
			var found []string
			for k, v := range mapLookup {
				if strings.Contains(strings.ToLower(v), s) {
					found = append(found, fmt.Sprintf("%3d - %s\n", k, v))
				}
			}
			slices.Sort(found)
			for _, v := range found {
				sb.WriteString(v)
			}
		}
		e.results.SetText(sb.String())
	}
	e.ExtendBaseWidget(e)
	return e
}

func (e *MapData) CreateRenderer() fyne.WidgetRenderer {
	data := pr.GetMapData()
	transport := pr.Transportations

	// --- Interactive Map Section ---
	worldSelect := widget.NewSelect([]string{"World of Balance", "World of Ruin"}, nil)
	worldSelect.SetSelected("World of Balance")
	mapImagePath := "resources/ff6-world-map_world-of-balance.png"
	defaultWorldWidth, defaultWorldHeight := 256, 256

	// --- Zoomable Map Section ---
	// Add a visible background to help debug layout
	bg := canvas.NewRectangle(&color.NRGBA{R: 220, G: 220, B: 255, A: 255})
	bg.SetMinSize(fyne.NewSize(1, 1))
	// --- Coordinate display label ---
	coordLabel := widget.NewLabel("")
	snapToLandmark := widget.NewCheck("Snap to nearest landmark", nil)
	snapToLandmark.SetChecked(false)

	locationSearch := widget.NewEntry()
	locationSearch.SetPlaceHolder("Filter locations...")
	var filteredLocations []mapLocationItem
	selectedLocationName := ""
	calibratingName := ""
	calibLabel := widget.NewLabel("")

	mapWidget := NewMapClickWidget(mapImagePath, nil, nil)
	mapContainer := container.NewMax(bg, mapWidget)

	getWorld := func() int {
		if worldSelect.Selected == "World of Ruin" {
			return 2
		}
		return 1
	}

	getGameSize := func() (int, int) {
		gameWidth, gameHeight := data.Gps.Width, data.Gps.Height
		if gameWidth <= 0 {
			gameWidth = defaultWorldWidth
		}
		if gameHeight <= 0 {
			gameHeight = defaultWorldHeight
		}
		return gameWidth, gameHeight
	}

	toImageMarker := func(gameX, gameY float64) (float64, float64, bool) {
		gameWidth, gameHeight := getGameSize()
		if gameWidth <= 1 || gameHeight <= 1 || mapWidget.imgWidth <= 1 || mapWidget.imgHeight <= 1 {
			return -1, -1, false
		}
		mx := float64(mapWidget.imgWidth-1) * gameX / float64(gameWidth-1)
		my := float64(mapWidget.imgHeight-1) * gameY / float64(gameHeight-1)
		return mx, my, true
	}

	updateMarker := func() {
		gameWidth, gameHeight := getGameSize()
		world := getWorld()

		mapWidget.showMarker = false
		mapWidget.markerX, mapWidget.markerY = -1, -1
		if mapWidget.imgWidth <= 0 || mapWidget.imgHeight <= 0 {
			mapWidget.Refresh()
			return
		}
		validPlayer := data.Player.X >= 0 && data.Player.Y >= 0 && data.Player.X <= float64(gameWidth-1) && data.Player.Y <= float64(gameHeight-1)
		if !validPlayer {
			mapWidget.Refresh()
			return
		}
		if (world == 1 && data.MapID != 1) || (world == 2 && data.MapID != 2) {
			mapWidget.Refresh()
			return
		}

		mx, my, ok := toImageMarker(data.Player.X, data.Player.Y)
		if !ok {
			mapWidget.Refresh()
			return
		}
		mapWidget.markerX = mx
		mapWidget.markerY = my
		mapWidget.showMarker = true
		mapWidget.Refresh()
	}

	rebuildLocations := func() {
		world := getWorld()
		q := strings.ToLower(strings.TrimSpace(locationSearch.Text))

		byName := map[string]mapLocationItem{}

		// Add defaults from the map artwork.
		for _, name := range defaultWorldLocationNames(world) {
			byName[name] = mapLocationItem{Name: name}
		}

		// Start with any user-added location names.
		for _, name := range config.MapLocations(world) {
			byName[name] = mapLocationItem{Name: name}
		}

		// Include any names already calibrated.
		for name := range config.AllMapPoints(world) {
			if name == "" {
				continue
			}
			if _, ok := byName[name]; !ok {
				byName[name] = mapLocationItem{Name: name}
			}
		}

		// Merge in built-in fallbacks (where we have coordinates).
		for _, lm := range consts.Landmarks {
			if lm.World != world {
				continue
			}
			item, ok := byName[lm.Name]
			if !ok {
				item = mapLocationItem{Name: lm.Name}
			}
			item.HasFallback = true
			item.X = lm.X
			item.Y = lm.Y
			byName[lm.Name] = item
		}

		filteredLocations = filteredLocations[:0]
		for _, item := range byName {
			if q != "" && !strings.Contains(strings.ToLower(item.Name), q) {
				continue
			}
			filteredLocations = append(filteredLocations, item)
		}
		sort.Slice(filteredLocations, func(i, j int) bool {
			return strings.ToLower(filteredLocations[i].Name) < strings.ToLower(filteredLocations[j].Name)
		})
	}

	locationsList := widget.NewList(
		func() int { return len(filteredLocations) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(id widget.ListItemID, o fyne.CanvasObject) {
			if id < 0 || id >= len(filteredLocations) {
				return
			}
			o.(*widget.Label).SetText(filteredLocations[id].Name)
		},
	)

	teleportTo := func(name string, x, y float64) {
		world := getWorld()
		if world == 2 {
			data.MapID = 2
		} else {
			data.MapID = 1
		}
		data.Player.X = x
		data.Player.Y = y
		data.Player.Z = 0
		data.PlayerDirection = 0
		updateMarker()
		if name != "" {
			coordLabel.SetText(fmt.Sprintf("%s (%.1f, %.1f) on %s", name, x, y, worldSelect.Selected))
		} else {
			coordLabel.SetText(fmt.Sprintf("(%.1f, %.1f) on %s", x, y, worldSelect.Selected))
		}
	}

	locationsList.OnSelected = func(id widget.ListItemID) {
		if id < 0 || id >= len(filteredLocations) {
			return
		}
		loc := filteredLocations[id]
		selectedLocationName = loc.Name

		world := getWorld()
		if p, ok := config.GetMapPoint(world, loc.Name); ok && mapWidget.imgWidth > 1 && mapWidget.imgHeight > 1 {
			calibLabel.SetText("Using saved map point (image-calibrated).")
			gameWidth, gameHeight := getGameSize()
			gx := float64(p.X) * float64(gameWidth-1) / float64(mapWidget.imgWidth-1)
			gy := float64(p.Y) * float64(gameHeight-1) / float64(mapWidget.imgHeight-1)
			teleportTo(loc.Name, gx, gy)
			return
		}

		if loc.HasFallback {
			calibLabel.SetText("No point set for this location (using fallback).")
			teleportTo(loc.Name, loc.X, loc.Y)
			return
		}

		calibLabel.SetText("No point set for this location. Click 'Set Point', then click on the map.")
	}

	locationSearch.OnChanged = func(_ string) {
		rebuildLocations()
		locationsList.Refresh()
	}

	var updateMap func()
	currentMapPath := ""
	updateMap = func() {
		nextMapPath := "resources/ff6-world-map_world-of-balance.png"
		if worldSelect.Selected == "World of Ruin" {
			nextMapPath = "resources/ff6-world-map_world-of-ruin.png"
		}
		mapImagePath = nextMapPath

		if currentMapPath != nextMapPath {
			mapWidget.SetImagePath(nextMapPath)
			mapWidget.ResetView()
			currentMapPath = nextMapPath
		}

		rebuildLocations()
		locationsList.Refresh()
		updateMarker()
		mapContainer.Refresh()
	}

	mapWidget.onClick = func(x, y int) {
		gameWidth, gameHeight := getGameSize()
		if mapWidget.imgWidth <= 0 || mapWidget.imgHeight <= 0 {
			return
		}

		world := getWorld()

		// If we're calibrating a named point, store it and teleport there.
		if calibratingName != "" {
			name := calibratingName
			calibratingName = ""
			calibLabel.SetText("")
			config.SetMapPoint(world, name, config.MapPoint{X: x, Y: y})
			if gameWidth > 1 && gameHeight > 1 && mapWidget.imgWidth > 1 && mapWidget.imgHeight > 1 {
				gx := float64(x) * float64(gameWidth-1) / float64(mapWidget.imgWidth-1)
				gy := float64(y) * float64(gameHeight-1) / float64(mapWidget.imgHeight-1)
				teleportTo(name, gx, gy)
			}
			return
		}

		gx := float64(x) * float64(gameWidth-1) / float64(mapWidget.imgWidth-1)
		gy := float64(y) * float64(gameHeight-1) / float64(mapWidget.imgHeight-1)

		if gx < 0 {
			gx = 0
		} else if gx > float64(gameWidth-1) {
			gx = float64(gameWidth - 1)
		}
		if gy < 0 {
			gy = 0
		} else if gy > float64(gameHeight-1) {
			gy = float64(gameHeight - 1)
		}

			targetX, targetY := gx, gy
			targetName := ""
			if snapToLandmark.Checked {
				// Prefer calibrated points (pixel-perfect on the map image).
				points := config.AllMapPoints(world)
				if len(points) > 0 {
				minDist := 1e18
				bestName := ""
				bestPx, bestPy := 0, 0
				for name, p := range points {
					dx := float64(p.X - x)
					dy := float64(p.Y - y)
					dist := dx*dx + dy*dy
					if dist < minDist {
						minDist = dist
						bestName = name
						bestPx = p.X
						bestPy = p.Y
					}
				}
				if bestName != "" && mapWidget.imgWidth > 1 && mapWidget.imgHeight > 1 && gameWidth > 1 && gameHeight > 1 {
					targetName = bestName
					targetX = float64(bestPx) * float64(gameWidth-1) / float64(mapWidget.imgWidth-1)
					targetY = float64(bestPy) * float64(gameHeight-1) / float64(mapWidget.imgHeight-1)
				}
				} else {
				// Fallback: built-in landmark coordinates (may not match custom map art).
					minDist := 1e9
					found := false
					var nearestX, nearestY float64
					nearestName := ""
					for _, lm := range consts.Landmarks {
						if lm.World != world {
							continue
						}
						dx := lm.X - gx
						dy := lm.Y - gy
						dist := dx*dx + dy*dy
						if dist < minDist {
							minDist = dist
							nearestX = lm.X
							nearestY = lm.Y
							nearestName = lm.Name
							found = true
						}
					}
					if found {
						targetX = nearestX
						targetY = nearestY
						targetName = nearestName
					}
				}
			}

		teleportTo(targetName, targetX, targetY)
	}

	updateMap()
	worldSelect.OnChanged = func(_ string) {
		selectedLocationName = ""
		calibratingName = ""
		calibLabel.SetText("")
		locationsList.UnselectAll()
		updateMap()
	}

	resetViewBtn := widget.NewButton("Reset View", func() { mapWidget.ResetView() })
	setPointBtn := widget.NewButton("Set Point", func() {
		if selectedLocationName == "" {
			return
		}
		calibratingName = selectedLocationName
		calibLabel.SetText(fmt.Sprintf("Click on the map to set: %s", calibratingName))
	})
	clearPointBtn := widget.NewButton("Clear Point", func() {
		if selectedLocationName == "" {
			return
		}
		config.ClearMapPoint(getWorld(), selectedLocationName)
		calibLabel.SetText("Cleared saved map point (using fallback).")
	})
	addLocationBtn := widget.NewButton("Add", func() {
		name := strings.TrimSpace(locationSearch.Text)
		if name == "" {
			return
		}
		config.AddMapLocation(getWorld(), name)
		locationSearch.SetText("")
		rebuildLocations()
		locationsList.Refresh()
	})

	locationsScroll := container.NewVScroll(locationsList)
	locationsScroll.SetMinSize(fyne.NewSize(240, 400))
	locationsCard := widget.NewCard(
		"Locations",
		"",
		container.NewBorder(
			container.NewBorder(nil, nil, nil, addLocationBtn, locationSearch),
			container.NewHBox(setPointBtn, clearPointBtn, calibLabel),
			nil,
			nil,
			locationsScroll,
		),
	)

	mapSplit := container.NewHSplit(locationsCard, mapContainer)
	mapSplit.Offset = 0.28

	mapSection := container.NewVBox(
		widget.NewLabel("Click anywhere on the map to teleport (or pick a location on the left)."),
		container.NewHBox(worldSelect, resetViewBtn, snapToLandmark),
		mapSplit,
		coordLabel,
	)

	// --- Existing Data Cards Section ---
	cards := make([]fyne.CanvasObject, 0, 4)
	cards = append(cards, widget.NewCard("Player", "", container.NewVBox(
		inputs.NewLabeledIntEntryWithHint("Map ID", inputs.NewIntEntryWithData(&data.MapID), inputs.HintArgs{
			Align: inputs.NewAlign(fyne.TextAlignTrailing),
			Hints: &mapLookup,
		}),
		inputs.NewLabeledEntry("Position X", inputs.NewFloatEntryWithData(&data.Player.X)),
		inputs.NewLabeledEntry("Position Y", inputs.NewFloatEntryWithData(&data.Player.Y)),
		inputs.NewLabeledEntry("Position Z", inputs.NewFloatEntryWithData(&data.Player.Z)),
		inputs.NewLabeledEntry("Facing Direction", inputs.NewIntEntryWithData(&data.PlayerDirection)),
	)))
	cards = append(cards,
		widget.NewCard("GPS", "", container.NewVBox(
			inputs.NewLabeledEntry("World", e.newWorldSelect(&data.Gps.MapID)),
			inputs.NewLabeledEntry("Area ID", inputs.NewIntEntryWithData(&data.Gps.AreaID)),
			inputs.NewLabeledEntry("ID", inputs.NewIntEntryWithData(&data.Gps.GpsID)),
			inputs.NewLabeledEntry("Width", inputs.NewIntEntryWithData(&data.Gps.Width)),
			inputs.NewLabeledEntry("Height", inputs.NewIntEntryWithData(&data.Gps.Height)),
		)))
	if len(transport) >= 5 {
		bj := transport[3]
		cards = append(cards,
			widget.NewCard("Blackjack", "", container.NewVBox(
				inputs.NewLabeledEntry("Enabled", widget.NewCheckWithData("", binding.BindBool(&bj.Enabled))),
				inputs.NewLabeledEntry("World", e.newWorldSelectUnset(&bj.MapID)),
				inputs.NewLabeledEntry("Position X", inputs.NewFloatEntryWithData(&bj.Position.X)),
				inputs.NewLabeledEntry("Position Y", inputs.NewFloatEntryWithData(&bj.Position.Y)),
				inputs.NewLabeledEntry("Position Z", inputs.NewFloatEntryWithData(&bj.Position.Z)),
				inputs.NewLabeledEntry("Facing Direction", inputs.NewIntEntryWithData(&bj.Direction)),
			)))
		f := transport[4]
		cards = append(cards,
			widget.NewCard("Falcon", "", container.NewVBox(
				inputs.NewLabeledEntry("Enabled", widget.NewCheckWithData("", binding.BindBool(&f.Enabled))),
				inputs.NewLabeledEntry("World", e.newWorldSelectUnset(&f.MapID)),
				inputs.NewLabeledEntry("Position X", inputs.NewFloatEntryWithData(&f.Position.X)),
				inputs.NewLabeledEntry("Position Y", inputs.NewFloatEntryWithData(&f.Position.Y)),
				inputs.NewLabeledEntry("Position Z", inputs.NewFloatEntryWithData(&f.Position.Z)),
				inputs.NewLabeledEntry("Facing Direction", inputs.NewIntEntryWithData(&f.Direction)),
			)))
	}

	// Fix: combine mapSection and cards into a single slice for NewVBox
	allVBoxItems := []fyne.CanvasObject{mapSection}
	allVBoxItems = append(allVBoxItems, cards...)

	leftCol := container.NewVScroll(container.NewVBox(allVBoxItems...))
	mapIDsCol := container.NewBorder(
		widget.NewLabel("Map IDs"), nil, nil, nil,
		container.NewVScroll(mapsTextBox),
	)
	findByNameCol := container.NewBorder(
		inputs.NewLabeledEntry("Find By Name:", e.search), nil, nil, nil,
		container.NewVScroll(e.results),
	)

	rightSplit := container.NewHSplit(mapIDsCol, findByNameCol)
	rightSplit.Offset = 0.40
	mainSplit := container.NewHSplit(leftCol, rightSplit)
	mainSplit.Offset = 0.35

	return widget.NewSimpleRenderer(mainSplit)
}

func (e *MapData) newWorldSelect(i *int) *widget.Select {
	s := widget.NewSelect([]string{"Balance", "Ruin"}, func(s string) {
		if s == "Balance" {
			*i = 1
		} else {
			*i = 2
		}
	})
	if *i == 1 {
		s.SetSelected("Balance")
	} else {
		s.SetSelected("Ruin")
	}
	return s
}

func (e *MapData) newWorldSelectUnset(i *int) *widget.Select {
	s := widget.NewSelect([]string{"-", "Balance", "Ruin"}, func(s string) {
		if s == "Balance" {
			*i = 1
		} else if s == "Ruin" {
			*i = 2
		} else {
			*i = -1
		}
	})
	if *i == 1 {
		s.SetSelected("Balance")
	} else if *i == 2 {
		s.SetSelected("Ruin")
	} else {
		s.SetSelected("-")
	}
	return s
}

func (e *MapData) showTeleportDialog(data *pr.MapData) {
	// Choose which map to show
	worldSelect := widget.NewSelect([]string{"World of Balance", "World of Ruin"}, nil)
	worldSelect.SetSelected("World of Balance")
	mapImagePath := "resources/ff6-world-map_world-of-balance.png"
	defaultWorldWidth, defaultWorldHeight := 256, 256
	snapToLandmark := widget.NewCheck("Snap to nearest landmark", nil)
	snapToLandmark.SetChecked(false)
	coordLabel := widget.NewLabel("")

	getWorld := func() int {
		if worldSelect.Selected == "World of Ruin" {
			return 2
		}
		return 1
	}

	getGameSize := func() (int, int) {
		gameWidth, gameHeight := data.Gps.Width, data.Gps.Height
		if gameWidth <= 0 {
			gameWidth = defaultWorldWidth
		}
		if gameHeight <= 0 {
			gameHeight = defaultWorldHeight
		}
		return gameWidth, gameHeight
	}

	mapWidget := NewMapClickWidget(mapImagePath, nil, nil)

	updateMarker := func() {
		gameWidth, gameHeight := getGameSize()
		world := getWorld()

		mapWidget.showMarker = false
		mapWidget.markerX, mapWidget.markerY = -1, -1
		if mapWidget.imgWidth <= 0 || mapWidget.imgHeight <= 0 {
			mapWidget.Refresh()
			return
		}
			validPlayer := data.Player.X >= 0 && data.Player.Y >= 0 && data.Player.X <= float64(gameWidth-1) && data.Player.Y <= float64(gameHeight-1)
		if !validPlayer {
			mapWidget.Refresh()
			return
		}
		if (world == 1 && data.MapID != 1) || (world == 2 && data.MapID != 2) {
			mapWidget.Refresh()
			return
		}

			if gameWidth <= 1 || gameHeight <= 1 || mapWidget.imgWidth <= 1 || mapWidget.imgHeight <= 1 {
				mapWidget.Refresh()
				return
			}
			mapWidget.markerX = float64(mapWidget.imgWidth-1) * data.Player.X / float64(gameWidth-1)
			mapWidget.markerY = float64(mapWidget.imgHeight-1) * data.Player.Y / float64(gameHeight-1)
			mapWidget.showMarker = true
			mapWidget.Refresh()
		}

	teleportTo := func(name string, x, y float64) {
		world := getWorld()
		if world == 2 {
			data.MapID = 2
		} else {
			data.MapID = 1
		}
		data.Player.X = x
		data.Player.Y = y
		data.Player.Z = 0
		data.PlayerDirection = 0
		updateMarker()
		if name != "" {
			coordLabel.SetText(fmt.Sprintf("%s (%.1f, %.1f) on %s", name, x, y, worldSelect.Selected))
		} else {
			coordLabel.SetText(fmt.Sprintf("(%.1f, %.1f) on %s", x, y, worldSelect.Selected))
		}
	}

		mapWidget.onClick = func(x, y int) {
			gameWidth, gameHeight := getGameSize()
			if mapWidget.imgWidth <= 1 || mapWidget.imgHeight <= 1 || gameWidth <= 1 || gameHeight <= 1 {
				return
			}

			gx := float64(x) * float64(gameWidth-1) / float64(mapWidget.imgWidth-1)
			gy := float64(y) * float64(gameHeight-1) / float64(mapWidget.imgHeight-1)

			if gx < 0 {
				gx = 0
			} else if gx > float64(gameWidth-1) {
				gx = float64(gameWidth - 1)
			}
			if gy < 0 {
				gy = 0
			} else if gy > float64(gameHeight-1) {
				gy = float64(gameHeight - 1)
			}

			targetX, targetY := gx, gy
			targetName := ""
			if snapToLandmark.Checked {
				world := getWorld()
				minDist := 1e9
				found := false
				var nearest consts.Landmark
				var nearestX, nearestY float64
				for _, lm := range consts.Landmarks {
					if lm.World != world {
						continue
					}
					dx := lm.X - gx
					dy := lm.Y - gy
					dist := dx*dx + dy*dy
					if dist < minDist {
						minDist = dist
						nearest = lm
						nearestX = lm.X
						nearestY = lm.Y
						found = true
					}
				}
				if found {
					targetX = nearestX
					targetY = nearestY
					targetName = nearest.Name
				}
			}

			teleportTo(targetName, targetX, targetY)
		}

	updateMap := func() {
		if worldSelect.Selected == "World of Ruin" {
			mapImagePath = "resources/ff6-world-map_world-of-ruin.png"
		} else {
			mapImagePath = "resources/ff6-world-map_world-of-balance.png"
		}
		mapWidget.SetImagePath(mapImagePath)
		updateMarker()
	}

	updateMap()
	worldSelect.OnChanged = func(_ string) { updateMap() }

	d := dialog.NewCustom("Click to Teleport", "Close",
		container.NewVBox(
			widget.NewLabel("Select World and click a location on the map to teleport."),
			worldSelect,
			snapToLandmark,
			mapWidget,
			coordLabel,
		), fyne.CurrentApp().Driver().AllWindows()[0])
	d.Show()
}
