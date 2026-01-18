package forms

import (
	"context"
	"ffvi_editor/io/pr"
	"ffvi_editor/models/consts"
	"fmt"
	"image/color"
	"sort"
	"strings"

	"ffvi_editor/scripting"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// CombatDepthPackDialog provides a comprehensive interface for combat customization
type CombatDepthPackDialog struct {
	window    fyne.Window
	save      *pr.PR
	dialog    dialog.Dialog
	statusMsg *widget.Label
}

// NewCombatDepthPackDialog creates and shows the Combat Depth Pack dialog
func NewCombatDepthPackDialog(win fyne.Window, save *pr.PR) *CombatDepthPackDialog {
	cdpd := &CombatDepthPackDialog{
		window: win,
		save:   save,
	}
	cdpd.buildUI()
	cdpd.Show()
	return cdpd
}

// Show displays the dialog
func (cdpd *CombatDepthPackDialog) Show() {
	if cdpd.dialog != nil {
		cdpd.dialog.Show()
	}
}

// Hide hides the dialog
func (cdpd *CombatDepthPackDialog) Hide() {
	if cdpd.dialog != nil {
		cdpd.dialog.Hide()
	}
}

// getUniqueZones returns all unique zone names from the maps database
func getUniqueZones() []string {
	zoneMap := make(map[string]bool)
	for _, m := range consts.Maps {
		if m != nil {
			// Extract zone name (remove additional info in parentheses)
			zoneName := m.Name
			if idx := strings.Index(zoneName, " ("); idx >= 0 {
				zoneName = zoneName[:idx]
			}
			zoneMap[zoneName] = true
		}
	}
	
	// Convert to sorted slice
	zones := make([]string, 0, len(zoneMap))
	for zone := range zoneMap {
		zones = append(zones, zone)
	}
	sort.Strings(zones)
	return zones
}

// buildUI constructs the complete dialog UI with modern styling
func (cdpd *CombatDepthPackDialog) buildUI() {
	// Status message display with modern styling
	cdpd.statusMsg = CreateStatusLabel("Ready to apply combat customizations")
	cdpd.statusMsg.Alignment = fyne.TextAlignCenter

	// ===== ENCOUNTER TUNER SECTION =====
	zoneSelect := widget.NewSelect(getUniqueZones(), func(s string) {})
	zoneSelect.PlaceHolder = "Select a zone..."

	rateEntry := widget.NewEntry()
	rateEntry.SetText("1.0")
	rateEntry.SetPlaceHolder("0.5 - 2.0")

	eliteEntry := widget.NewEntry()
	eliteEntry.SetText("0.10")
	eliteEntry.SetPlaceHolder("0.0 - 1.0")

	applyEncounterBtn := CreateCardButton("⚙ Apply Tuning", func() {
		if zoneSelect.Selected == "" {
			dialog.ShowError(fmt.Errorf("zone cannot be empty"), cdpd.window)
			return
		}
		code := scripting.BuildEncounterScript(zoneSelect.Selected, rateEntry.Text, eliteEntry.Text)
		_, err := scripting.RunSnippetWithSave(context.Background(), code, cdpd.save)
		if err != nil {
			dialog.ShowError(err, cdpd.window)
			return
		}
		cdpd.statusMsg.SetText(fmt.Sprintf("✓ Applied to %s", zoneSelect.Selected))
		dialog.ShowInformation("Success", fmt.Sprintf("Zone: %s\nRate: %s\nElite: %s", zoneSelect.Selected, rateEntry.Text, eliteEntry.Text), cdpd.window)
	})

	encounterCard := CreateModernCard(
		"🎲",
		"Dynamic Encounter Tuner",
		"Customize encounter rates and elite chances",
		container.NewVBox(
			CreateFormRow("Zone", zoneSelect),
			CreateFormRow("Spawn Rate", rateEntry),
			CreateFormRow("Elite Chance", eliteEntry),
		),
		applyEncounterBtn,
	)

	// ===== BOSS REMIX & AFFIXES SECTION =====
	affixEntry := widget.NewEntry()
	affixEntry.SetPlaceHolder("enraged, arcane_shield, ...")

	applyBossBtn := CreateCardButton("👹 Generate", func() {
		if affixEntry.Text == "" {
			dialog.ShowError(fmt.Errorf("affixes cannot be empty"), cdpd.window)
			return
		}
		code := scripting.BuildBossScript(affixEntry.Text)
		_, err := scripting.RunSnippetWithSave(context.Background(), code, cdpd.save)
		if err != nil {
			dialog.ShowError(err, cdpd.window)
			return
		}
		cdpd.statusMsg.SetText("✓ Boss remix applied")
		dialog.ShowInformation("Success", fmt.Sprintf("Affixes applied:\n%s", affixEntry.Text), cdpd.window)
	})

	bossCard := CreateModernCard(
		"👹",
		"Boss Remix & Affixes",
		"Apply special effects to boss encounters",
		container.NewVBox(
			CreateFormRow("Affixes", affixEntry),
		),
		applyBossBtn,
	)

	// ===== AI COMPANION DIRECTOR SECTION =====
	profileEntry := widget.NewEntry()
	profileEntry.SetPlaceHolder("aggressive, support, balanced")

	riskEntry := widget.NewSelect([]string{"low", "normal", "high"}, func(s string) {})
	riskEntry.SetSelected("normal")

	applyCompanionBtn := CreateCardButton("🤖 Save", func() {
		if profileEntry.Text == "" {
			dialog.ShowError(fmt.Errorf("profile cannot be empty"), cdpd.window)
			return
		}
		code := scripting.BuildCompanionScript(profileEntry.Text, riskEntry.Selected)
		_, err := scripting.RunSnippetWithSave(context.Background(), code, cdpd.save)
		if err != nil {
			dialog.ShowError(err, cdpd.window)
			return
		}
		cdpd.statusMsg.SetText(fmt.Sprintf("✓ Profile saved: %s", profileEntry.Text))
		dialog.ShowInformation("Success", fmt.Sprintf("Profile: %s\nRisk: %s", profileEntry.Text, riskEntry.Selected), cdpd.window)
	})

	companionCard := CreateModernCard(
		"🤖",
		"AI Companion Director",
		"Configure AI behavior profiles",
		container.NewVBox(
			CreateFormRow("Profile", profileEntry),
			CreateFormRow("Risk Level", riskEntry),
		),
		applyCompanionBtn,
	)

	// ===== SMOKE TESTS SECTION =====
	smokeBtn := CreateCardButton("🧪 Run Tests", func() {
		_, err := scripting.RunSnippetWithSave(context.Background(), scripting.BuildSmokeScript(), cdpd.save)
		if err != nil {
			dialog.ShowError(err, cdpd.window)
			return
		}
		cdpd.statusMsg.SetText("✓ Tests passed")
		dialog.ShowInformation("Success", "All smoke tests completed successfully", cdpd.window)
	})

	smokeCard := container.NewVBox(
		CreateDivider(),
		container.NewCenter(
			container.NewVBox(
				widget.NewLabel("Quality Assurance"),
				smokeBtn,
			),
		),
	)

	// ===== MAIN LAYOUT WITH MODERN DESIGN =====
	mainContent := container.NewVBox(
		// Header
		CreateHeader("⚔ Combat Configuration"),
		CreateStatusBar(cdpd.statusMsg),

		// Cards
		encounterCard,
		bossCard,
		companionCard,
		smokeCard,

		// Padding
		layout.NewSpacer(),
	)

	// Wrap in scroll for long content with dark background
	scrollContent := CreateScrollableContent(mainContent, 550, 650)

	// Modern button row
	closeBtn := widget.NewButton("Close", func() { 
		if cdpd.dialog != nil {
			cdpd.dialog.Hide()
		}
	})
	closeBtn.Importance = widget.LowImportance

	buttonRow := CreateDialogButtonRow(closeBtn)

	// Create dark glass wrapper for entire dialog
	darkBg := canvas.NewRectangle(color.NRGBA{R: 18, G: 18, B: 24, A: 255})

	// Create dialog with border layout wrapped in dark background
	contentWithBg := container.NewMax(
		darkBg,
		container.NewBorder(
			nil,           // Top
			buttonRow,     // Bottom
			nil,           // Left
			nil,           // Right
			scrollContent, // Center
		),
	)

	cdpd.dialog = dialog.NewCustom(
		"Combat Depth Pack",
		"",
		contentWithBg,
		cdpd.window,
	)
	cdpd.dialog.Resize(fyne.NewSize(650, 750))
}
