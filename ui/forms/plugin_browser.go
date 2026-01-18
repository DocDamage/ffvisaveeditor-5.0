package forms

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"ffvi_editor/marketplace"
	"ffvi_editor/plugins"
)

// PluginBrowserDialog provides a marketplace browser for discovering and installing plugins
type PluginBrowserDialog struct {
	window        fyne.Window
	dialog        dialog.Dialog
	pluginManager *plugins.Manager
	marketplace   *marketplace.Client
	registry      *marketplace.Registry

	// UI Elements - Header
	searchEntry    *widget.Entry
	categorySelect *widget.Select
	sortSelect     *widget.Select
	refreshBtn     *widget.Button

	// UI Elements - Main Content
	pluginList       *widget.List
	pluginListData   []marketplace.RemotePlugin
	detailsPanel     *fyne.Container
	selectedPluginID string

	// UI Elements - Footer
	statusLabel       *widget.Label
	progressBar       *widget.ProgressBar
	progressContainer *fyne.Container

	// State
	allPlugins      []marketplace.RemotePlugin
	filteredPlugins []marketplace.RemotePlugin
	downloading     bool
}

// NewPluginBrowserDialog creates a new marketplace browser dialog
func NewPluginBrowserDialog(w fyne.Window, mgr *plugins.Manager, marketplaceClient *marketplace.Client, reg *marketplace.Registry) *PluginBrowserDialog {
	d := &PluginBrowserDialog{
		window:        w,
		pluginManager: mgr,
		marketplace:   marketplaceClient,
		registry:      reg,
		statusLabel:   widget.NewLabel("Ready"),
		progressBar:   widget.NewProgressBar(),
		downloading:   false,
	}

	d.progressBar.Hide()
	d.progressContainer = container.NewVBox(d.progressBar)

	return d
}

// Show displays the plugin browser dialog
func (d *PluginBrowserDialog) Show() {
	// Build UI components FIRST before any goroutines
	header := d.buildHeader()
	content := d.buildContent()
	footer := d.buildFooter()

	// Assemble dialog content
	dialogContent := container.NewBorder(
		header,  // top
		footer,  // bottom
		nil,     // left
		nil,     // right
		content, // center
	)

	// Create dialog
	d.dialog = dialog.NewCustom("Plugin Marketplace", "Close", dialogContent, d.window)
	d.dialog.Resize(fyne.NewSize(900, 600))

	// Load initial data AFTER UI is built
	go d.refreshPluginList()

	d.dialog.Show()
}

// buildHeader creates the search and filter controls
func (d *PluginBrowserDialog) buildHeader() *fyne.Container {
	// Search entry
	d.searchEntry = widget.NewEntry()
	d.searchEntry.SetPlaceHolder("Search plugins...")
	d.searchEntry.OnChanged = func(query string) {
		d.filterAndSortPlugins()
	}

	// Category selector
	categories := []string{
		"All Categories",
		"Editor Tools",
		"Speedrun",
		"Analytics",
		"Automation",
		"Utilities",
	}
	d.categorySelect = widget.NewSelect(categories, func(selected string) {
		d.filterAndSortPlugins()
	})
	d.categorySelect.SetSelected("All Categories")

	// Sort selector
	sortOptions := []string{
		"Newest",
		"Popular",
		"Highest Rated",
		"Name (A-Z)",
	}
	d.sortSelect = widget.NewSelect(sortOptions, func(selected string) {
		d.filterAndSortPlugins()
	})
	d.sortSelect.SetSelected("Popular")

	// Refresh button
	d.refreshBtn = widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), func() {
		go d.refreshPluginList()
	})

	// Layout header
	searchRow := container.NewBorder(
		nil,
		nil,
		widget.NewLabel("Search:"),
		d.refreshBtn,
		d.searchEntry,
	)

	filterRow := container.NewHBox(
		widget.NewLabel("Category:"),
		d.categorySelect,
		layout.NewSpacer(),
		widget.NewLabel("Sort:"),
		d.sortSelect,
	)

	return container.NewVBox(
		searchRow,
		filterRow,
		widget.NewSeparator(),
	)
}

// buildContent creates the main content area with plugin list and details panel
func (d *PluginBrowserDialog) buildContent() *fyne.Container {
	// Plugin list
	d.pluginListData = []marketplace.RemotePlugin{}

	d.pluginList = widget.NewList(
		func() int {
			return len(d.pluginListData)
		},
		func() fyne.CanvasObject {
			return d.createPluginListItem()
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			if id < 0 || id >= len(d.pluginListData) {
				return
			}
			d.updatePluginListItem(item, d.pluginListData[id])
		},
	)

	d.pluginList.OnSelected = func(id widget.ListItemID) {
		if id >= 0 && id < len(d.pluginListData) {
			plugin := d.pluginListData[id]
			d.selectedPluginID = plugin.ID
			d.displayPluginDetails(&plugin)
		}
	}

	listScroll := container.NewVScroll(d.pluginList)
	listScroll.SetMinSize(fyne.NewSize(350, 400))

	// Details panel (right side)
	d.detailsPanel = container.NewVBox(
		widget.NewLabel("Select a plugin to view details"),
	)
	detailsScroll := container.NewVScroll(d.detailsPanel)

	// Split layout
	split := container.NewHSplit(
		listScroll,
		detailsScroll,
	)
	split.SetOffset(0.4)

	return container.NewMax(split)
}

// buildFooter creates the status bar and progress indicator
func (d *PluginBrowserDialog) buildFooter() *fyne.Container {
	return container.NewVBox(
		widget.NewSeparator(),
		d.progressContainer,
		d.statusLabel,
	)
}

// createPluginListItem creates a template for a plugin list item
func (d *PluginBrowserDialog) createPluginListItem() fyne.CanvasObject {
	nameLabel := widget.NewLabelWithStyle("Plugin Name", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	authorLabel := widget.NewLabel("by Author")
	versionLabel := widget.NewLabel("v1.0.0")
	downloadsLabel := widget.NewLabel("★ 4.5 • 1,234 downloads")

	return container.NewVBox(
		nameLabel,
		authorLabel,
		container.NewHBox(
			versionLabel,
			layout.NewSpacer(),
			downloadsLabel,
		),
		widget.NewSeparator(),
	)
}

// updatePluginListItem updates a plugin list item with actual data
func (d *PluginBrowserDialog) updatePluginListItem(item fyne.CanvasObject, plugin marketplace.RemotePlugin) {
	container := item.(*fyne.Container)
	if len(container.Objects) < 4 {
		return
	}

	nameLabel := container.Objects[0].(*widget.Label)
	authorLabel := container.Objects[1].(*widget.Label)
	bottomRow := container.Objects[2].(*fyne.Container)

	nameLabel.SetText(plugin.Name)
	authorLabel.SetText(fmt.Sprintf("by %s", plugin.Author))

	if len(bottomRow.Objects) >= 3 {
		versionLabel := bottomRow.Objects[0].(*widget.Label)
		downloadsLabel := bottomRow.Objects[2].(*widget.Label)

		versionLabel.SetText(fmt.Sprintf("v%s", plugin.Version))

		// Format rating with stars
		stars := strings.Repeat("★", int(plugin.Rating))
		stars += strings.Repeat("☆", 5-int(plugin.Rating))
		downloadsLabel.SetText(fmt.Sprintf("%s %.1f • %s downloads",
			stars, plugin.Rating, formatNumber(plugin.Downloads)))
	}
}

// displayPluginDetails shows detailed information for a selected plugin
func (d *PluginBrowserDialog) displayPluginDetails(plugin *marketplace.RemotePlugin) {
	if plugin == nil {
		d.detailsPanel.Objects = []fyne.CanvasObject{
			widget.NewLabel("Select a plugin to view details"),
		}
		d.detailsPanel.Refresh()
		return
	}

	// Check if plugin is already installed
	installed, _ := d.registry.GetPlugin(plugin.ID)
	isInstalled := installed != nil

	// Plugin header
	titleLabel := widget.NewLabelWithStyle(plugin.Name, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	titleLabel.TextStyle.Bold = true
	titleLabel.Resize(fyne.NewSize(400, 30))

	authorLabel := widget.NewLabel(fmt.Sprintf("by %s", plugin.Author))
	versionLabel := widget.NewLabel(fmt.Sprintf("Version: %s", plugin.Version))

	// Rating display
	stars := strings.Repeat("★", int(plugin.Rating))
	stars += strings.Repeat("☆", 5-int(plugin.Rating))
	ratingLabel := widget.NewLabel(fmt.Sprintf("Rating: %s (%.1f/5.0)", stars, plugin.Rating))

	downloadsLabel := widget.NewLabel(fmt.Sprintf("Downloads: %s", formatNumber(plugin.Downloads)))
	categoryLabel := widget.NewLabel(fmt.Sprintf("Category: %s", plugin.Category))
	updatedLabel := widget.NewLabel(fmt.Sprintf("Last Updated: %s", plugin.UpdatedAt.Format("Jan 2, 2006")))

	// Description
	descLabel := widget.NewLabel("Description:")
	descLabel.TextStyle.Bold = true
	descText := widget.NewLabel(plugin.Description)
	descText.Wrapping = fyne.TextWrapWord

	// Tags
	var tagLabels []fyne.CanvasObject
	if len(plugin.Tags) > 0 {
		tagLabels = append(tagLabels, widget.NewLabel("Tags: "))
		for _, tag := range plugin.Tags {
			tagLabel := widget.NewLabel(tag)
			tagLabel.TextStyle.Italic = true
			tagLabels = append(tagLabels, tagLabel)
		}
	}

	// Action buttons
	var actionButtons []fyne.CanvasObject

	if isInstalled {
		installedBtn := widget.NewButton("✓ Installed", func() {})
		installedBtn.Importance = widget.LowImportance
		installedBtn.Disable()
		actionButtons = append(actionButtons, installedBtn)

		if installed.Version != plugin.Version {
			updateBtn := widget.NewButtonWithIcon("Update Available", theme.DownloadIcon(), func() {
				d.updatePlugin(plugin)
			})
			updateBtn.Importance = widget.HighImportance
			actionButtons = append(actionButtons, updateBtn)
		}

		uninstallBtn := widget.NewButtonWithIcon("Uninstall", theme.DeleteIcon(), func() {
			d.uninstallPlugin(plugin)
		})
		actionButtons = append(actionButtons, uninstallBtn)
	} else {
		installBtn := widget.NewButtonWithIcon("Install Plugin", theme.DownloadIcon(), func() {
			d.installPlugin(plugin)
		})
		installBtn.Importance = widget.HighImportance
		actionButtons = append(actionButtons, installBtn)
	}

	// Rate button
	rateBtn := widget.NewButtonWithIcon("Rate & Review", theme.ContentAddIcon(), func() {
		d.showRatingDialog(plugin)
	})
	actionButtons = append(actionButtons, rateBtn)

	// Plugin repository link (if available)
	if plugin.RepositoryURL != "" {
		homepageBtn := widget.NewButtonWithIcon("Repository", theme.InfoIcon(), func() {
			// Open URL in browser
			fmt.Printf("Opening repository: %s\n", plugin.RepositoryURL)
		})
		actionButtons = append(actionButtons, homepageBtn)
	}

	// Assemble details panel
	details := []fyne.CanvasObject{
		titleLabel,
		authorLabel,
		widget.NewSeparator(),
		container.NewVBox(
			versionLabel,
			ratingLabel,
			downloadsLabel,
			categoryLabel,
			updatedLabel,
		),
		widget.NewSeparator(),
		container.NewHBox(actionButtons...),
		widget.NewSeparator(),
		descLabel,
		descText,
	}

	if len(tagLabels) > 0 {
		details = append(details, widget.NewSeparator())
		details = append(details, container.NewHBox(tagLabels...))
	}

	// Version history (if available)
	if len(plugin.VersionHistory) > 1 {
		details = append(details, widget.NewSeparator())
		details = append(details, widget.NewLabel("Version History:"))
		for _, ver := range plugin.VersionHistory {
			versionItem := widget.NewLabel(fmt.Sprintf("  • v%s - %s", ver.Version, ver.ReleaseDate.Format("Jan 2, 2006")))
			details = append(details, versionItem)
		}
	}

	d.detailsPanel.Objects = details
	d.detailsPanel.Refresh()
}

// refreshPluginList fetches the latest plugin list from the marketplace
func (d *PluginBrowserDialog) refreshPluginList() {
	// Safety check
	if d.marketplace == nil {
		d.setStatus("Error: Marketplace client not available")
		return
	}

	d.setStatus("Fetching plugins from marketplace...")
	if d.refreshBtn != nil {
		d.refreshBtn.Disable()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	plugins, err := d.marketplace.ListPlugins(ctx, &marketplace.ListOptions{
		Category: "",
		SortBy:   "downloads",
		Limit:    100,
		Offset:   0,
	})

	if d.refreshBtn != nil {
		d.refreshBtn.Enable()
	}

	if err != nil {
		d.setStatus(fmt.Sprintf("Error: %v", err))
		dialog.ShowError(fmt.Errorf("failed to fetch plugins: %w", err), d.window)
		return
	}

	d.allPlugins = plugins
	d.filterAndSortPlugins()
	d.setStatus(fmt.Sprintf("Loaded %d plugins", len(plugins)))
}

// filterAndSortPlugins applies current filters and sorting
func (d *PluginBrowserDialog) filterAndSortPlugins() {
	// Safety checks
	if d.searchEntry == nil || d.categorySelect == nil || d.sortSelect == nil {
		return
	}

	query := strings.ToLower(strings.TrimSpace(d.searchEntry.Text))
	category := d.categorySelect.Selected
	sortBy := d.sortSelect.Selected

	// Filter plugins
	filtered := []marketplace.RemotePlugin{}
	for _, plugin := range d.allPlugins {
		// Category filter
		if category != "All Categories" && plugin.Category != category {
			continue
		}

		// Search filter
		if query != "" {
			nameMatch := strings.Contains(strings.ToLower(plugin.Name), query)
			authorMatch := strings.Contains(strings.ToLower(plugin.Author), query)
			descMatch := strings.Contains(strings.ToLower(plugin.Description), query)

			if !nameMatch && !authorMatch && !descMatch {
				continue
			}
		}

		filtered = append(filtered, plugin)
	}

	// Sort plugins
	switch sortBy {
	case "Newest":
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].UpdatedAt.After(filtered[j].UpdatedAt)
		})
	case "Popular":
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].Downloads > filtered[j].Downloads
		})
	case "Highest Rated":
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].Rating > filtered[j].Rating
		})
	case "Name (A-Z)":
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].Name < filtered[j].Name
		})
	}

	d.filteredPlugins = filtered
	d.pluginListData = filtered
	d.pluginList.Refresh()

	// Update status
	if len(filtered) == len(d.allPlugins) {
		d.setStatus(fmt.Sprintf("%d plugins", len(filtered)))
	} else {
		d.setStatus(fmt.Sprintf("%d of %d plugins", len(filtered), len(d.allPlugins)))
	}
}

// installPlugin downloads and installs a plugin
func (d *PluginBrowserDialog) installPlugin(plugin *marketplace.RemotePlugin) {
	if d.downloading {
		dialog.ShowInformation("Busy", "Another download is in progress", d.window)
		return
	}

	d.downloading = true
	d.progressBar.Show()
	d.progressBar.SetValue(0)
	d.progressContainer.Refresh()
	d.setStatus(fmt.Sprintf("Installing %s...", plugin.Name))

	go func() {
		defer func() {
			d.downloading = false
			d.progressBar.Hide()
			d.progressContainer.Refresh()
		}()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		d.progressBar.SetValue(0.3)
		err := d.marketplace.InstallPlugin(ctx, plugin.ID, plugin.Version)
		if err != nil {
			d.setStatus(fmt.Sprintf("Error: %v", err))
			dialog.ShowError(fmt.Errorf("failed to install plugin: %w", err), d.window)
			return
		}

		// Track installation in registry (if available)
		d.progressBar.SetValue(0.7)
		if d.registry != nil {
			if err := d.registry.TrackInstallation(plugin.ID, plugin.Version); err != nil {
				d.setStatus(fmt.Sprintf("Warning: %v", err))
			}
		}

		d.progressBar.SetValue(1.0)
		d.setStatus(fmt.Sprintf("Successfully installed %s", plugin.Name))

		// Refresh plugin details to show "Installed" status
		d.displayPluginDetails(plugin)

		// Show success message
		dialog.ShowInformation("Success",
			fmt.Sprintf("Plugin '%s' installed successfully!\n\nRestart the editor to activate the plugin.", plugin.Name),
			d.window)
	}()
}

// updatePlugin updates an installed plugin to a newer version
func (d *PluginBrowserDialog) updatePlugin(plugin *marketplace.RemotePlugin) {
	if d.registry == nil {
		dialog.ShowError(fmt.Errorf("registry not configured"), d.window)
		return
	}

	installed, _ := d.registry.GetPlugin(plugin.ID)
	if installed == nil {
		dialog.ShowError(fmt.Errorf("plugin not installed"), d.window)
		return
	}

	msg := fmt.Sprintf("Update %s from v%s to v%s?", plugin.Name, installed.Version, plugin.Version)
	dialog.ShowConfirm("Update Plugin", msg, func(confirmed bool) {
		if confirmed {
			// Uninstall old version first (best-effort)
			if d.registry != nil {
				_ = d.registry.UninstallPlugin(plugin.ID)
			}

			// Install new version
			d.installPlugin(plugin)
		}
	}, d.window)
}

// uninstallPlugin removes an installed plugin
func (d *PluginBrowserDialog) uninstallPlugin(plugin *marketplace.RemotePlugin) {
	msg := fmt.Sprintf("Uninstall plugin '%s'?\n\nThis will remove all plugin files.", plugin.Name)
	dialog.ShowConfirm("Uninstall Plugin", msg, func(confirmed bool) {
		if confirmed {
			var err error
			if d.registry != nil {
				err = d.registry.UninstallPlugin(plugin.ID)
			}
			if err != nil {
				dialog.ShowError(fmt.Errorf("failed to uninstall: %w", err), d.window)
				return
			}

			d.setStatus(fmt.Sprintf("Uninstalled %s", plugin.Name))
			d.displayPluginDetails(plugin) // Refresh to show "Install" button

			dialog.ShowInformation("Success",
				fmt.Sprintf("Plugin '%s' uninstalled successfully!", plugin.Name),
				d.window)
		}
	}, d.window)
}

// showRatingDialog displays a dialog for rating and reviewing a plugin
func (d *PluginBrowserDialog) showRatingDialog(plugin *marketplace.RemotePlugin) {
	// Rating slider
	ratingValue := 5.0
	ratingLabel := widget.NewLabel("Rating: 5.0 / 5.0")
	ratingSlider := widget.NewSlider(1.0, 5.0)
	ratingSlider.Step = 0.5
	ratingSlider.Value = 5.0
	ratingSlider.OnChanged = func(value float64) {
		ratingValue = value
		stars := int(value)
		halfStar := value-float64(stars) >= 0.5
		starStr := strings.Repeat("★", stars)
		if halfStar {
			starStr += "⯪"
		}
		starStr += strings.Repeat("☆", 5-stars)
		if halfStar {
			starStr = starStr[:len(starStr)-1]
		}
		ratingLabel.SetText(fmt.Sprintf("Rating: %.1f / 5.0  %s", value, starStr))
	}

	// Review entry
	reviewEntry := widget.NewMultiLineEntry()
	reviewEntry.SetPlaceHolder("Write your review (optional)...")
	reviewEntry.SetMinRowsVisible(4)

	// Dialog content
	content := container.NewVBox(
		widget.NewLabel(fmt.Sprintf("Rate '%s'", plugin.Name)),
		widget.NewSeparator(),
		ratingLabel,
		ratingSlider,
		widget.NewLabel("Review:"),
		reviewEntry,
	)

	// Submit dialog
	ratingDialog := dialog.NewCustomConfirm("Submit Rating", "Submit", "Cancel", content, func(confirmed bool) {
		if confirmed {
			d.submitRating(plugin, ratingValue, reviewEntry.Text)
		}
	}, d.window)

	ratingDialog.Resize(fyne.NewSize(400, 300))
	ratingDialog.Show()
}

// submitRating submits a user rating to the marketplace
func (d *PluginBrowserDialog) submitRating(plugin *marketplace.RemotePlugin, rating float64, review string) {
	d.setStatus("Submitting rating...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pluginRating := &marketplace.PluginRating{
		PluginID:  plugin.ID,
		Rating:    float32(rating),
		Review:    review,
		UserID:    "Anonymous", // TODO: Get actual username from settings
		Timestamp: time.Now(),
	}

	err := d.marketplace.SubmitRating(ctx, pluginRating)
	if err != nil {
		d.setStatus(fmt.Sprintf("Error: %v", err))
		dialog.ShowError(fmt.Errorf("failed to submit rating: %w", err), d.window)
		return
	}

	// Save rating locally as well
	d.registry.SaveRatings([]*marketplace.PluginRating{pluginRating})

	d.setStatus("Rating submitted successfully")
	dialog.ShowInformation("Success", "Thank you for your rating!", d.window)
}

// setStatus updates the status label
func (d *PluginBrowserDialog) setStatus(status string) {
	if d.statusLabel != nil {
		d.statusLabel.SetText(status)
	}
}

// formatNumber formats large numbers with commas (e.g., 1234 -> "1,234")
func formatNumber(n int) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}

	var result string
	for i, digit := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result += ","
		}
		result += string(digit)
	}
	return result
}
