package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ffvi_editor/achievements"
	"ffvi_editor/browser"
	"ffvi_editor/cloud"
	"ffvi_editor/docs"
	"ffvi_editor/global"
	"ffvi_editor/io"
	"ffvi_editor/io/backup"
	"ffvi_editor/io/config"
	"ffvi_editor/io/pr"
	"ffvi_editor/io/validation"
	"ffvi_editor/marketplace"
	"ffvi_editor/models"
	pri "ffvi_editor/models/pr"
	"ffvi_editor/plugins"
	"ffvi_editor/settings"
	"ffvi_editor/ui/forms"
	"ffvi_editor/ui/forms/dialogs"
	"ffvi_editor/ui/forms/selections"
	"ffvi_editor/ui/state"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// splitSpriteFrames breaks a multi-frame FF6Sprite into per-frame sprites using tile data.
// Assumes sprite.Data stores frames sequentially, 4bpp (2 pixels per byte), width*height/2 bytes each.
func splitSpriteFrames(sprite *models.FF6Sprite) ([]*models.FF6Sprite, []int) {
	if sprite == nil || sprite.Width == 0 || sprite.Height == 0 || sprite.Frames <= 0 {
		return nil, nil
	}

	bytesPerFrame := (sprite.Width * sprite.Height) / 2
	if bytesPerFrame <= 0 {
		return nil, nil
	}

	totalBytes := len(sprite.Data)
	maxFrames := totalBytes / bytesPerFrame
	frameCount := sprite.Frames
	if maxFrames > 0 && frameCount > maxFrames {
		frameCount = maxFrames
	}
	if frameCount <= 0 {
		return nil, nil
	}

	frames := make([]*models.FF6Sprite, 0, frameCount)
	timings := make([]int, 0, frameCount)

	for i := 0; i < frameCount; i++ {
		start := i * bytesPerFrame
		end := start + bytesPerFrame
		if end > totalBytes {
			break
		}

		frame := &models.FF6Sprite{
			ID:             fmt.Sprintf("%s_frame_%d", sprite.ID, i),
			Name:           sprite.Name,
			Type:           sprite.Type,
			Description:    sprite.Description,
			Data:           append([]byte(nil), sprite.Data[start:end]...),
			Palette:        sprite.Palette,
			Frames:         1,
			FrameRate:      sprite.FrameRate,
			FrameDurations: nil,
			SourceFile:     sprite.SourceFile,
			ImportedFrom:   sprite.ImportedFrom,
			ImportDate:     sprite.ImportDate,
			Width:          sprite.Width,
			Height:         sprite.Height,
			IsCompressed:   sprite.IsCompressed,
			Checksum:       sprite.Checksum,
			Author:         sprite.Author,
			License:        sprite.License,
			Tags:           sprite.Tags,
			CreatedDate:    sprite.CreatedDate,
			ModifiedDate:   sprite.ModifiedDate,
		}

		frames = append(frames, frame)

		dur := 100
		if len(sprite.FrameDurations) > i {
			dur = sprite.FrameDurations[i]
			if dur <= 0 {
				dur = 100
			}
		}
		timings = append(timings, dur)
	}

	return frames, timings
}

// getCharacterWithSprite retrieves a character from the global character list by CharacterID.
// Returns a character with placeholder sprite if needed for animation/export features.
// Supports both field sprites (16x24, 3 frames) and battle sprites (32x32, 6 frames).
func (g *gui) getCharacterWithSprite(characterID int) *models.Character {
	char := pri.GetCharacterByID(characterID)
	if char == nil {
		return nil
	}

	// Try to load sprite from ROM if available
	if char.Sprite == nil && g.romExtractor != nil && g.romExtractor.IsROMLoaded() {
		sprite, err := g.romExtractor.ExtractCharacterSprite(characterID)
		if err == nil && sprite != nil {
			char.Sprite = sprite
			fmt.Printf("Loaded field sprite for character %d from ROM\n", characterID)
		} else {
			fmt.Printf("Warning: Failed to load field sprite for character %d from ROM: %v\n", characterID, err)
		}
	}

	// Try to extract palette from ROM if character doesn't have one
	if char.Sprite != nil && (char.Sprite.Palette == nil || len(char.Sprite.Palette.Colors) == 0) {
		if g.romExtractor != nil && g.romExtractor.IsROMLoaded() {
			palette, err := g.romExtractor.ExtractCharacterPalette(characterID)
			if err == nil && palette != nil {
				char.Sprite.Palette = palette
				fmt.Printf("Extracted palette for character %d from ROM\n", characterID)
			}
		}
	}

	// If sprite is still not loaded, create a placeholder 16x24 character sprite
	if char.Sprite == nil {
		char.Sprite = &models.FF6Sprite{
			Width:  16,
			Height: 24,
			Frames: 3,
			Data:   make([]byte, 16*24*3/2), // 3 frames, 4bpp (0.5 bytes per pixel)
			Palette: &models.Palette{
				Colors: [16]models.RGB555{},
			},
		}
	}

	return char
}

// getCharacterWithBattleSprite retrieves a character with battle sprite data (32x32, 6 frames)
func (g *gui) getCharacterWithBattleSprite(characterID int) *models.Character {
	char := pri.GetCharacterByID(characterID)
	if char == nil {
		return nil
	}

	// Try to load battle sprite from ROM if available
	if g.romExtractor != nil && g.romExtractor.IsROMLoaded() {
		battleSprite, err := g.romExtractor.ExtractBattleSprite(characterID)
		if err == nil && battleSprite != nil {
			// Swap sprite temporarily (don't modify original)
			char.Sprite = battleSprite // 32x32, 6 frames
			fmt.Printf("Loaded battle sprite for character %d from ROM\n", characterID)
			return char
		} else {
			fmt.Printf("Warning: Failed to load battle sprite for character %d from ROM: %v\n", characterID, err)
		}
	}

	// Fall back to current sprite if battle sprite not available
	return char
}

// buildAnimationFromSprite constructs AnimationData using split frames and timings.
func buildAnimationFromSprite(sprite *models.FF6Sprite, name string) *models.AnimationData {
	frames, timings := splitSpriteFrames(sprite)
	if len(frames) == 0 {
		return nil
	}

	total := int64(0)
	for _, d := range timings {
		total += int64(d)
	}

	anim := &models.AnimationData{
		Frames:       frames,
		FrameTimings: timings,
		PlaybackMode: models.PlayContinuous,
		DefaultSpeed: 1.0,
		Metadata: models.AnimationMetadata{
			Name:          name,
			Description:   "",
			Author:        "",
			Created:       sprite.CreatedDate,
			Modified:      sprite.ModifiedDate,
			TotalDuration: total,
			FrameCount:    len(frames),
		},
	}

	return anim
}

func sanitizeFileName(name string) string {
	s := strings.TrimSpace(name)
	if s == "" {
		s = "sprite"
	}
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "\t", "_")
	s = strings.ReplaceAll(s, "\\", "_")
	s = strings.ReplaceAll(s, "/", "_")
	return s
}

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
		app                 fyne.App
		window              fyne.Window
		canvas              *fyne.Container
		open                *fyne.MenuItem
		save                *fyne.MenuItem
		prev                fyne.CanvasObject
		pr                  *pr.PR
		backupManager       *backup.Manager
		undoStack           *state.UndoStack
		themeSwitcher       *ThemeSwitcher
		settingsManager     *settings.Manager
		achievementTracker  *achievements.Tracker
		cloudManager        *cloud.Manager
		pluginManager       *plugins.Manager
		helpSystem          *docs.HelpSystem
		marketplaceClient   *marketplace.Client
		marketplaceRegistry *marketplace.Registry
		validationStatus    *widget.Label
		romExtractor        *io.ROMSpriteExtractor
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

	// Initialize backup manager
	backupDir := filepath.Join(config.SaveDir(), "backups")
	backupMgr, err := backup.NewManager(backupDir, 10)
	if err != nil {
		// If backup manager fails to initialize, create a nil placeholder
		// The user can still use the app without backup functionality
		backupMgr = nil
	}

	var (
		a = app.NewWithID("com.ff6editor.app")
		g = &gui{
			app:                a,
			window:             a.NewWindow(fmt.Sprintf("Final Fantasy VI Save Editor - v%s", browser.Version)),
			canvas:             container.NewMax(),
			backupManager:      backupMgr,
			undoStack:          state.NewUndoStack(100),
			themeSwitcher:      NewThemeSwitcher(a.Preferences()),
			settingsManager:    settings.New(),
			achievementTracker: achievements.NewTracker(),
			cloudManager:       cloud.New(),
			pluginManager:      plugins.NewManager(filepath.Join(config.SaveDir(), "plugins"), nil),
			helpSystem:         docs.NewHelpSystem(),
			marketplaceClient:  marketplace.NewClient("https://api.ff6-marketplace.local", "demo-key"),
		}
	)

	// Apply FF6 custom theme
	a.Settings().SetTheme(NewFF6Theme(theme.VariantDark))
	// Initialize marketplace registry
	reg, err := marketplace.NewRegistry(filepath.Join(config.SaveDir(), "marketplace", "registry.json"))
	if err != nil {
		fmt.Printf("Warning: Failed to initialize marketplace registry: %v\n", err)
	}
	g.marketplaceRegistry = reg

	g.window.SetIcon(fyne.NewStaticResource("icon", resourceIconIco.StaticContent))
	g.window.SetContent(g.canvas)

	// Register window for theme updates
	g.themeSwitcher.RegisterWindow(g.window)

	// Initialize settings from persistence
	settingsPath := filepath.Join(config.SaveDir(), "settings.json")
	settingsMgr := settings.NewManager(settingsPath)
	if err := settingsMgr.Load(); err != nil {
		// If load fails, use defaults
		settingsMgr = settings.New()
	}
	g.settingsManager = settingsMgr

	// Initialize ROM sprite extractor if ROM path is configured, or try default locations
	s := g.settingsManager.Get()
	var romExtractor *io.ROMSpriteExtractor
	var romErr error

	if s != nil && s.ROMPath != "" {
		// Use configured ROM path
		romExtractor, romErr = io.NewROMSpriteExtractor(s.ROMPath)
	} else {
		// Try to find ROM from default locations
		romExtractor, romErr = io.NewROMSpriteExtractor("")
	}

	if romErr == nil && romExtractor.IsROMLoaded() {
		g.romExtractor = romExtractor
		fmt.Println("✅ ROM loaded successfully for sprite extraction")

		// Load all character palettes into cache in background for fast access
		go func() {
			if err := romExtractor.LoadAllPalettesCached(); err != nil {
				fmt.Printf("Note: Background palette caching encountered an issue: %v\n", err)
			}
		}()
	} else {
		fmt.Printf("⚠️  ROM not found, sprites will use placeholders: %v\n", romErr)
	}

	// Apply theme setting from settings
	if s != nil && s.Theme == "light" {
		// TODO: Apply custom light theme to Fyne app
		// g.app.Settings().SetTheme(theme.GetLightTheme())
	}

	// Initialize achievement tracker with unlock callbacks
	g.achievementTracker.SetUnlockCallback(func(achievement *achievements.Achievement) {
		dialog.ShowInformation(
			"Achievement Unlocked!",
			fmt.Sprintf("%s\n%s\n+%d points", achievement.Name, achievement.Description, achievement.Points),
			g.window,
		)
	})

	// Register window resize listener for saving window size
	// Integrate undo/redo keyboard handling
	undoCtrl := NewUndoRedoController(g.undoStack)
	// Update validation status after undo/redo
	undoCtrl.SetOnUndo(func() {
		if g.pr != nil {
			res := validation.NewValidator().Validate(g.pr)
			g.validationStatus.SetText(fmt.Sprintf("Validation: %d errors, %d warnings", len(res.Errors), len(res.Warnings)))
		}
	})
	undoCtrl.SetOnRedo(func() {
		if g.pr != nil {
			res := validation.NewValidator().Validate(g.pr)
			g.validationStatus.SetText(fmt.Sprintf("Validation: %d errors, %d warnings", len(res.Errors), len(res.Warnings)))
		}
	})
	// Create status bar labels
	g.validationStatus = widget.NewLabel("Validation: n/a")
	statusBar := container.NewHBox(
		layout.NewSpacer(),
		undoCtrl.GetStatusLabel(),
		widget.NewSeparator(),
		g.validationStatus,
		layout.NewSpacer(),
	)
	// Wrap main canvas with status bar at bottom
	g.window.SetContent(container.NewBorder(nil, statusBar, nil, nil, g.canvas))
	g.window.Canvas().SetOnTypedKey(func(ke *fyne.KeyEvent) {
		// Handle undo/redo shortcuts
		undoCtrl.HandleKeyboardShortcut(ke)
		if ke.Name == fyne.KeyEscape {
			// Could add escape key handling if needed
		}
	})

	g.open = fyne.NewMenuItem("Open", func() {
		g.Load()
	})
	g.save = fyne.NewMenuItem("Save", func() {
		g.Save()
	})
	g.save.Disabled = true
	x, y := config.WindowSize()
	// Set minimum window size for better UX
	if x < 1000 {
		x = 1000
	}
	if y < 700 {
		y = 700
	}
	g.window.Resize(fyne.NewSize(x, y))
	g.window.SetMaster()

	// Show welcome screen on startup
	g.showWelcomeScreen()
	// Build Edit menu from controller
	undoItem, redoItem := undoCtrl.BuildMenuItems()
	g.window.SetMainMenu(fyne.NewMainMenu(
		fyne.NewMenu("File",
			g.open,
			fyne.NewMenuItemSeparator(),
			g.save,
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem("Manage Backups", func() {
				if g.backupManager != nil {
					d := dialogs.NewBackupManagerDialog(g.window, g.backupManager)
					d.Show()
				} else {
					dialog.ShowError(fmt.Errorf("backup manager not available"), g.window)
				}
			}),
		),
		fyne.NewMenu("Edit",
			undoItem,
			redoItem,
		),
		fyne.NewMenu("Tools",
			fyne.NewMenuItem("Configure ROM...", func() {
				currentPath := ""
				if s := g.settingsManager.Get(); s != nil {
					currentPath = s.ROMPath
				}

				d := forms.NewROMConfigDialog(g.window, currentPath)
				d.OnSave(func(romPath string) {
					// Update settings
					if s := g.settingsManager.Get(); s != nil {
						s.ROMPath = romPath
						if err := g.settingsManager.Save(); err != nil {
							dialog.ShowError(fmt.Errorf("failed to save ROM path: %w", err), g.window)
						}
					}
				})
				d.Show()
			}),
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem("Cloud Sync...", func() {
				d := forms.NewCloudSettingsDialog(g.window, g.cloudManager)
				d.Show()
			}),
			fyne.NewMenuItem("Combat Depth Pack...", func() {
				forms.NewCombatDepthPackDialog(g.window, g.pr)
			}),
			fyne.NewMenuItem("Plugin Manager...", func() {
				d := forms.NewPluginManagerDialog(g.window, g.pluginManager, g.marketplaceClient, g.marketplaceRegistry)
				d.Show()
			}),
			fyne.NewMenuItem("Lua Scripts...", func() {
				d := forms.NewScriptEditorDialog(g.window)
				d.Show()
			}),
			fyne.NewMenuItem("Batch Operations...", func() {
				if g.pr != nil {
					d := forms.NewBatchOperationsDialog(g.pr, g.window)
					d.Show()
				} else {
					dialog.ShowError(fmt.Errorf("no save file loaded"), g.window)
				}
			}),
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem("Sprite Editor...", func() {
				d := forms.NewSpriteImportDialog(g.window)
				d.OnImportSuccess(func(sprite *models.FF6Sprite) {
					if g.pr != nil {
						// TODO: Apply sprite to current character
						dialog.ShowInformation("Success", "Sprite imported successfully!", g.window)
					}
				})
				d.OnImportError(func(err error) {
					dialog.ShowError(fmt.Errorf("import error: %w", err), g.window)
				})
				d.Show()
			}),
			fyne.NewMenuItem("Palette Editor...", func() {
				party := pri.GetParty()
				if g.pr != nil && party.Enabled && party.Members[0] != nil && party.Members[0] != pri.EmptyPartyMember {
					// Use the first character's palette for now
					// TODO: Make this character-aware
					palette := &models.Palette{
						Colors: [16]models.RGB555{},
					}
					d := forms.NewPaletteEditorDialog(g.window, palette)
					d.OnApply(func(p *models.Palette) {
						dialog.ShowInformation("Success", "Palette updated!", g.window)
					})
					d.Show()
				} else {
					dialog.ShowError(fmt.Errorf("no character loaded"), g.window)
				}
			}),
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem("Animation Player...", func() {
				party := pri.GetParty()
				if g.pr == nil || !party.Enabled || party.Members[0] == nil || party.Members[0] == pri.EmptyPartyMember {
					dialog.ShowError(fmt.Errorf("no character loaded"), g.window)
					return
				}

				character := g.getCharacterWithSprite(party.Members[0].CharacterID)
				if character == nil || character.Sprite == nil {
					dialog.ShowError(fmt.Errorf("no sprite data available"), g.window)
					return
				}

				animationData := buildAnimationFromSprite(character.Sprite, fmt.Sprintf("%s Animation", character.Name))
				if animationData == nil {
					dialog.ShowError(fmt.Errorf("failed to build animation from sprite"), g.window)
					return
				}

				playerDialog := forms.NewAnimationPlayerDialog(animationData)
				if playerDialog != nil {
					playerDialog.Show(g.window)
				} else {
					dialog.ShowError(fmt.Errorf("failed to create animation player"), g.window)
				}
			}),
			fyne.NewMenuItem("Frame Editor...", func() {
				party := pri.GetParty()
				if g.pr == nil || !party.Enabled || party.Members[0] == nil || party.Members[0] == pri.EmptyPartyMember {
					dialog.ShowError(fmt.Errorf("no character loaded"), g.window)
					return
				}

				character := g.getCharacterWithSprite(party.Members[0].CharacterID)
				if character == nil || character.Sprite == nil || character.Sprite.Frames == 0 {
					dialog.ShowError(fmt.Errorf("no sprite frames available"), g.window)
					return
				}

				frames, _ := splitSpriteFrames(character.Sprite)
				if len(frames) == 0 {
					dialog.ShowError(fmt.Errorf("failed to split sprite frames"), g.window)
					return
				}

				editorDialog := forms.NewFrameEditorDialog(frames)
				if editorDialog != nil {
					editorDialog.Show(g.window)
				} else {
					dialog.ShowError(fmt.Errorf("failed to create frame editor"), g.window)
				}
			}),
			fyne.NewMenuItem("Export Animation...", func() {
				party := pri.GetParty()
				if g.pr == nil || !party.Enabled || party.Members[0] == nil || party.Members[0] == pri.EmptyPartyMember {
					dialog.ShowError(fmt.Errorf("no character loaded"), g.window)
					return
				}

				character := g.getCharacterWithSprite(party.Members[0].CharacterID)
				if character == nil || character.Sprite == nil || character.Sprite.Frames == 0 {
					dialog.ShowError(fmt.Errorf("no sprite frames available"), g.window)
					return
				}

				animationData := buildAnimationFromSprite(character.Sprite, fmt.Sprintf("%s Export", character.Name))
				if animationData == nil {
					dialog.ShowError(fmt.Errorf("failed to build animation from sprite"), g.window)
					return
				}

				exportDialog := forms.NewAnimationExportDialog(animationData)
				if exportDialog != nil {
					exportDialog.Show(g.window)
				} else {
					dialog.ShowError(fmt.Errorf("failed to create export dialog"), g.window)
				}
			}),
			fyne.NewMenuItem("Export All Animations...", func() {
				party := pri.GetParty()
				if g.pr == nil || !party.Enabled || party.Members[0] == nil || party.Members[0] == pri.EmptyPartyMember {
					dialog.ShowError(fmt.Errorf("no character loaded"), g.window)
					return
				}

				dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
					if err != nil {
						dialog.ShowError(err, g.window)
						return
					}
					if uri == nil {
						return
					}

					dest := uri.Path()
					exportedCount := 0

					for _, member := range party.Members {
						if member == nil || member == pri.EmptyPartyMember {
							continue
						}

						character := g.getCharacterWithSprite(member.CharacterID)
						if character == nil || character.Sprite == nil || character.Sprite.Frames == 0 {
							continue
						}

						anim := buildAnimationFromSprite(character.Sprite, fmt.Sprintf("%s Animation", character.Name))
						if anim == nil {
							continue
						}

						exporter := io.NewAnimationExporter(anim)
						if exporter == nil {
							continue
						}

						baseName := sanitizeFileName(character.Name)
						gifPath := filepath.Join(dest, fmt.Sprintf("%s_anim.gif", baseName))
						pngDir := filepath.Join(dest, fmt.Sprintf("%s_frames", baseName))

						opts := &io.AnimationExportOptions{Quality: 75, Scale: 1, Dither: false}

						_ = exporter.ExportGIF(gifPath, opts)
						_ = exporter.ExportFramesPNG(pngDir, opts)
						exportedCount++
					}

					dialog.ShowInformation("Export Complete", fmt.Sprintf("Exported %d character animations", exportedCount), g.window)
				}, g.window).Show()
			}),
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem("Validation Panel", func() {
				if g.pr != nil {
					// Show validation panel in main canvas
					g.savePreviousCanvas()
					validator := validation.NewValidator()
					// Update status bar with latest counts
					res := validator.Validate(g.pr)
					g.validationStatus.SetText(fmt.Sprintf("Validation: %d errors, %d warnings", len(res.Errors), len(res.Warnings)))
					panel := forms.NewValidationPanel(validator)
					panel.ValidateSaveData(g.pr)
					g.setCanvasContent(panel.BuildPanel())
				} else {
					dialog.ShowError(fmt.Errorf("no save file loaded"), g.window)
				}
			}),
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem("Palette Viewer...", func() {
				fmt.Println("DEBUG: Palette Viewer clicked")
				if g.romExtractor == nil || !g.romExtractor.IsROMLoaded() {
					dialog.ShowError(fmt.Errorf("ROM not loaded - cannot extract palettes"), g.window)
					return
				}

				party := pri.GetParty()
				if g.pr == nil || !party.Enabled || party.Members[0] == nil || party.Members[0] == pri.EmptyPartyMember {
					dialog.ShowError(fmt.Errorf("no character loaded"), g.window)
					return
				}

				// Extract palette for current character (uses cache if available)
				palette, err := g.romExtractor.ExtractCharacterPaletteWithCache(party.Members[0].CharacterID)
				if err != nil {
					dialog.ShowError(fmt.Errorf("failed to extract palette: %w", err), g.window)
					return
				}

				character := pri.GetCharacterByID(party.Members[0].CharacterID)
				paletteViewer := forms.NewPaletteViewerDialog(g.window, palette, character.Name)
				paletteViewer.Show()
			}),
			fyne.NewMenuItem("Battle Sprite Viewer...", func() {
				if g.romExtractor == nil || !g.romExtractor.IsROMLoaded() {
					dialog.ShowError(fmt.Errorf("ROM not loaded - cannot extract battle sprites"), g.window)
					return
				}

				party := pri.GetParty()
				if g.pr == nil || !party.Enabled || party.Members[0] == nil || party.Members[0] == pri.EmptyPartyMember {
					dialog.ShowError(fmt.Errorf("no character loaded"), g.window)
					return
				}

				// Extract battle sprite for current character (32x32, 6 frames)
				battleSprite, err := g.romExtractor.ExtractBattleSprite(party.Members[0].CharacterID)
				if err != nil {
					dialog.ShowError(fmt.Errorf("failed to extract battle sprite: %w", err), g.window)
					return
				}

				character := pri.GetCharacterByID(party.Members[0].CharacterID)
				animationData := buildAnimationFromSprite(battleSprite, fmt.Sprintf("%s Battle Sprite", character.Name))
				if animationData == nil {
					dialog.ShowError(fmt.Errorf("failed to build battle animation"), g.window)
					return
				}

				playerDialog := forms.NewAnimationPlayerDialog(animationData)
				if playerDialog != nil {
					playerDialog.Show(g.window)
				} else {
					dialog.ShowError(fmt.Errorf("failed to create battle sprite viewer"), g.window)
				}
			}),
		),
		fyne.NewMenu("Community",
			fyne.NewMenuItem("Plugin Marketplace...", func() {
				d := forms.NewPluginBrowserDialog(g.window, g.pluginManager, g.marketplaceClient, g.marketplaceRegistry)
				d.Show()
			}),
			fyne.NewMenuItem("Share Build...", func() {
				if g.pr != nil {
					d := forms.NewShareDialogFromPR(g.window, g.pr)
					d.Show()
				} else {
					dialog.ShowError(fmt.Errorf("no save file loaded"), g.window)
				}
			}),
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem("Achievements", func() {
				panel := forms.NewAchievementsPanel(g.window, g.achievementTracker)
				panel.Show()
			}),
		),
		fyne.NewMenu("Advanced",
			fyne.NewMenuItem("Speedrun Setup Wizard...", func() {
				if g.pr != nil {
					forms.NewSpeedrunSetupWizard(g.pr, g.window)
				} else {
					dialog.ShowError(fmt.Errorf("no save file loaded"), g.window)
				}
			}),
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem("JSON Export/Import...", func() {
				if g.pr != nil {
					forms.NewJSONExportImportDialog(g.pr, g.window)
				} else {
					dialog.ShowError(fmt.Errorf("no save file loaded"), g.window)
				}
			}),
		),
		fyne.NewMenu("View",
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem("Help & Documentation", func() {
				d := forms.NewHelpDialog(g.window)
				d.Show()
			}),
		),
		fyne.NewMenu("Settings",
			fyne.NewMenuItem("Preferences...", func() {
				d := forms.NewPreferencesDialog(g.window, g.settingsManager)
				d.Show()
			}),
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem("Light Theme", func() {
				g.app.Settings().SetTheme(NewFF6ThemeWithType("light"))
			}),
			fyne.NewMenuItem("Dark Theme", func() {
				g.app.Settings().SetTheme(NewFF6ThemeWithType("dark"))
			}),
			fyne.NewMenuItem("Cyberpunk Theme", func() {
				g.app.Settings().SetTheme(NewFF6ThemeWithType("cyberpunk"))
			}),
			fyne.NewMenuItem("Ocean Theme", func() {
				g.app.Settings().SetTheme(NewFF6ThemeWithType("ocean"))
			}),
			fyne.NewMenuItem("Forest Theme", func() {
				g.app.Settings().SetTheme(NewFF6ThemeWithType("forest"))
			}),
			fyne.NewMenuItem("Amethyst Theme", func() {
				g.app.Settings().SetTheme(NewFF6ThemeWithType("amethyst"))
			}),
		),
	))
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
	g.savePreviousCanvas()
	g.open.Disabled = true
	g.canvas.RemoveAll()
	fileIO := forms.NewFileIO(forms.Load, g.window, config.SaveDir(), func(name, dir, file string, _ int, saveType global.SaveFileType) {
		defer func() { g.open.Disabled = false }()
		// Load file
		config.SetSaveDir(dir)
		p := pr.New()
		if err := p.Load(filepath.Join(dir, file), saveType); err != nil {
			g.restorePreviousCanvas()
			dialog.NewError(err, g.window).Show()
		} else {
			// Success
			g.prev = nil
			g.save.Disabled = false
			g.pr = p
			// Update validation status on load
			validator := validation.NewValidator()
			res := validator.Validate(g.pr)
			g.validationStatus.SetText(fmt.Sprintf("Validation: %d errors, %d warnings", len(res.Errors), len(res.Warnings)))
			// Create editor and wire tab-change callback to refresh status
			ed := selections.NewEditor()
			ed.SetOnTabChanged(func(tabTitle string) {
				if g.pr != nil {
					res := validation.NewValidator().Validate(g.pr)
					g.validationStatus.SetText(fmt.Sprintf("Validation: %d errors, %d warnings", len(res.Errors), len(res.Warnings)))
					// If switching to Validation tab, refresh panel with latest data
					if tabTitle == "Validation" && ed.GetValidationPanel() != nil {
						ed.GetValidationPanel().ValidateSaveData(g.pr)
					}
				}
			})
			g.setCanvasContent(ed)
		}
	}, func() {
		defer func() { g.open.Disabled = false }()
		// Cancel
		g.restorePreviousCanvas()
	})
	g.canvas.Add(fileIO)
	g.canvas.Refresh()
	if g.window.Content() != nil {
		g.window.Content().Refresh()
	}
}

func (g *gui) Save() {
	g.savePreviousCanvas()
	g.open.Disabled = true
	g.save.Disabled = true
	g.canvas.RemoveAll()
	fileIO := forms.NewFileIO(forms.Save, g.window, config.SaveDir(), func(name, dir, file string, slot int, saveType global.SaveFileType) {
		defer func() {
			g.open.Disabled = false
			g.save.Disabled = false
		}()
		config.SetSaveDir(dir)
		// Pre-save validation and optional auto-fix
		validator := validation.NewValidator()
		result := validator.Validate(g.pr)
		// Update status bar with latest counts
		g.validationStatus.SetText(fmt.Sprintf("Validation: %d errors, %d warnings", len(result.Errors), len(result.Warnings)))
		proceedSave := func() {
			if err := g.pr.Save(slot, filepath.Join(dir, file), saveType); err != nil {
				g.restorePreviousCanvas()
				dialog.NewError(err, g.window).Show()
			} else {
				// Success
				g.restorePreviousCanvas()
			}
		}
		if !result.Valid {
			msg := fmt.Sprintf("Validation found %d errors and %d warnings. Auto-fix and continue?", len(result.Errors), len(result.Warnings))
			dialog.NewConfirm("Validation Issues", msg, func(ok bool) {
				if ok {
					// Attempt auto-fix
					_, _ = validator.AutoFixIssues(g.pr)
					// Re-validate (optional)
					newRes := validator.Validate(g.pr)
					g.validationStatus.SetText(fmt.Sprintf("Validation: %d errors, %d warnings", len(newRes.Errors), len(newRes.Warnings)))
					proceedSave()
				} else {
					// Secondary confirmation: proceed without fixes?
					dialog.NewConfirm("Proceed Without Fixes?", "Save anyway without fixing validation issues?", func(confirm bool) {
						if confirm {
							proceedSave()
						}
					}, g.window).Show()
				}
			}, g.window).Show()
		} else {
			proceedSave()
		}
	}, func() {
		defer func() {
			g.open.Disabled = false
			g.save.Disabled = false
		}()
		// Cancel
		g.restorePreviousCanvas()
	})
	g.canvas.Add(fileIO)
	g.canvas.Refresh()
	if g.window.Content() != nil {
		g.window.Content().Refresh()
	}
}

func (g *gui) Run() {
	g.window.ShowAndRun()
}

// setCanvasContent safely replaces canvas content and refreshes layout
func (g *gui) setCanvasContent(obj fyne.CanvasObject) {
	g.canvas.RemoveAll()
	g.canvas.Add(obj)
	g.canvas.Refresh()
	// Also refresh the window content to ensure proper layout propagation
	if g.window.Content() != nil {
		g.window.Content().Refresh()
	}
}

// restorePreviousCanvas restores the previous canvas content and refreshes
func (g *gui) restorePreviousCanvas() {
	if g.prev != nil {
		g.canvas.RemoveAll()
		g.canvas.Add(g.prev)
		g.canvas.Refresh()
		// Also refresh the window content to ensure proper layout propagation
		if g.window.Content() != nil {
			g.window.Content().Refresh()
		}
	}
}

// savePreviousCanvas saves current canvas as previous before changing it
func (g *gui) savePreviousCanvas() {
	if len(g.canvas.Objects) > 0 {
		g.prev = g.canvas.Objects[0]
	}
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

// showWelcomeScreen displays an attractive welcome/start screen
func (g *gui) showWelcomeScreen() {
	// Create welcome title
	title := canvas.NewText("Final Fantasy VI", theme.ForegroundColor())
	title.TextSize = 32
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Alignment = fyne.TextAlignCenter

	subtitle := canvas.NewText("Save Editor", theme.ForegroundColor())
	subtitle.TextSize = 24
	subtitle.Alignment = fyne.TextAlignCenter

	version := widget.NewLabel(fmt.Sprintf("Version %s", browser.Version))
	version.Alignment = fyne.TextAlignCenter

	// Quick action buttons with icons
	openBtn := widget.NewButtonWithIcon("Open Save File", theme.FolderOpenIcon(), func() {
		g.Load()
	})
	openBtn.Importance = widget.HighImportance

	recentBtn := widget.NewButtonWithIcon("Recent Files", theme.HistoryIcon(), func() {
		dialog.ShowInformation("Recent Files", "Recent files feature coming soon!", g.window)
	})

	helpBtn := widget.NewButtonWithIcon("Help & Documentation", theme.HelpIcon(), func() {
		d := forms.NewHelpDialog(g.window)
		d.Show()
	})

	// Feature cards in grid
	featuresGrid := container.NewGridWithColumns(2,
		g.createFeatureCard("⚔️ Combat Pack", "Advanced combat tuning with Lua scripting"),
		g.createFeatureCard("☁️ Cloud Backup", "Sync saves across devices"),
		g.createFeatureCard("🔌 Plugin System", "Extend functionality with plugins"),
		g.createFeatureCard("✓ Validation", "Automatic save file validation"),
	)

	// Assemble welcome screen
	welcomeContent := container.NewVBox(
		layout.NewSpacer(),
		container.NewCenter(container.NewVBox(
			title,
			subtitle,
			version,
		)),
		layout.NewSpacer(),
		container.NewCenter(container.NewVBox(
			openBtn,
			recentBtn,
			helpBtn,
		)),
		layout.NewSpacer(),
		widget.NewCard("", "Quick Features", featuresGrid),
		layout.NewSpacer(),
	)

	g.canvas.Add(welcomeContent)
	g.canvas.Refresh()
	g.prev = welcomeContent
}

// createFeatureCard creates a styled feature card
func (g *gui) createFeatureCard(title, description string) fyne.CanvasObject {
	titleLabel := widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	descLabel := widget.NewLabel(description)
	descLabel.Wrapping = fyne.TextWrapWord

	return container.NewVBox(titleLabel, descLabel)
}
