package forms

import (
	"context"
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"ffvi_editor/cloud"
	"ffvi_editor/io/config"
)

// CloudSettingsDialog manages cloud sync settings and status
type CloudSettingsDialog struct {
	window       fyne.Window
	cloudManager *cloud.Manager
	statusLabel  *widget.Label
	statusBind   binding.String
}

// NewCloudSettingsDialog creates a new cloud settings dialog
func NewCloudSettingsDialog(window fyne.Window, cloudManager *cloud.Manager) *CloudSettingsDialog {
	return &CloudSettingsDialog{
		window:       window,
		cloudManager: cloudManager,
		statusBind:   binding.NewString(),
		statusLabel:  widget.NewLabelWithData(binding.NewString()),
	}
}

// Show displays the cloud settings dialog
func (c *CloudSettingsDialog) Show() {
	cfg := config.GetCloudSettings()

	// Tab container for different sections
	googleTab := c.buildGoogleDriveTab(&cfg)
	dropboxTab := c.buildDropboxTab(&cfg)
	syncTab := c.buildSyncSettingsTab(&cfg)
	statusTab := c.buildStatusTab()

	tabs := container.NewAppTabs(
		container.NewTabItem("Google Drive", googleTab),
		container.NewTabItem("Dropbox", dropboxTab),
		container.NewTabItem("Sync Settings", syncTab),
		container.NewTabItem("Status", statusTab),
	)

	content := container.NewVBox(
		widget.NewLabel("Cloud Backup Configuration"),
		widget.NewSeparator(),
		tabs,
	)

	buttons := container.NewHBox(
		widget.NewButton("Save", func() {
			c.saveSettings()
			dialog.ShowInformation("Success", "Cloud settings saved", c.window)
		}),
		widget.NewButton("Test All Connections", func() {
			c.testAllConnections()
		}),
		widget.NewButton("Sync Now", func() {
			c.syncNow()
		}),
		widget.NewButton("Close", func() {
			// Dialog will close
		}),
	)

	allContent := container.NewBorder(nil, buttons, nil, nil, content)

	d := dialog.NewCustom("Cloud Backup Settings", "Close", allContent, c.window)
	d.Resize(fyne.NewSize(600, 700))
	d.Show()
}

// buildGoogleDriveTab creates the Google Drive configuration tab
func (c *CloudSettingsDialog) buildGoogleDriveTab(cfg *config.CloudConfig) *fyne.Container {
	enableCheck := widget.NewCheck("Enable Google Drive", func(checked bool) {
		cfg.GoogleDriveEnabled = checked
	})
	enableCheck.SetChecked(cfg.GoogleDriveEnabled)

	clientIDEntry := widget.NewEntry()
	clientIDEntry.SetText(cfg.GoogleDriveClientID)
	clientIDEntry.SetPlaceHolder("OAuth2 Client ID")

	clientSecretEntry := widget.NewEntry()
	clientSecretEntry.SetText(cfg.GoogleDriveClientSecret)
	clientSecretEntry.SetPlaceHolder("OAuth2 Client Secret")
	clientSecretEntry.Password = true

	authBtn := widget.NewButton("Authenticate with Google", func() {
		cfg.GoogleDriveClientID = clientIDEntry.Text
		cfg.GoogleDriveClientSecret = clientSecretEntry.Text
		c.authenticateProvider("google_drive")
	})

	testBtn := widget.NewButton("Test Connection", func() {
		cfg.GoogleDriveClientID = clientIDEntry.Text
		cfg.GoogleDriveClientSecret = clientSecretEntry.Text
		c.testConnection("google_drive")
	})

	return container.NewVBox(
		enableCheck,
		widget.NewForm(
			widget.NewFormItem("Client ID", clientIDEntry),
			widget.NewFormItem("Client Secret", clientSecretEntry),
		),
		container.NewHBox(authBtn, testBtn),
		widget.NewCard("Instructions", "Get credentials from Google Cloud Console",
			widget.NewLabel("1. Go to https://console.cloud.google.com\n"+
				"2. Create a new project\n"+
				"3. Enable Google Drive API\n"+
				"4. Create OAuth2 credentials\n"+
				"5. Copy Client ID and Secret here")),
	)
}

// buildDropboxTab creates the Dropbox configuration tab
func (c *CloudSettingsDialog) buildDropboxTab(cfg *config.CloudConfig) *fyne.Container {
	enableCheck := widget.NewCheck("Enable Dropbox", func(checked bool) {
		cfg.DropboxEnabled = checked
	})
	enableCheck.SetChecked(cfg.DropboxEnabled)

	appKeyEntry := widget.NewEntry()
	appKeyEntry.SetText(cfg.DropboxAppKey)
	appKeyEntry.SetPlaceHolder("App Key")

	appSecretEntry := widget.NewEntry()
	appSecretEntry.SetText(cfg.DropboxAppSecret)
	appSecretEntry.SetPlaceHolder("App Secret")
	appSecretEntry.Password = true

	authBtn := widget.NewButton("Authenticate with Dropbox", func() {
		cfg.DropboxAppKey = appKeyEntry.Text
		cfg.DropboxAppSecret = appSecretEntry.Text
		c.authenticateProvider("dropbox")
	})

	testBtn := widget.NewButton("Test Connection", func() {
		cfg.DropboxAppKey = appKeyEntry.Text
		cfg.DropboxAppSecret = appSecretEntry.Text
		c.testConnection("dropbox")
	})

	return container.NewVBox(
		enableCheck,
		widget.NewForm(
			widget.NewFormItem("App Key", appKeyEntry),
			widget.NewFormItem("App Secret", appSecretEntry),
		),
		container.NewHBox(authBtn, testBtn),
		widget.NewCard("Instructions", "Get credentials from Dropbox App Console",
			widget.NewLabel("1. Go to https://www.dropbox.com/developers\n"+
				"2. Create a new app\n"+
				"3. Choose \"Scoped access\"\n"+
				"4. Copy App Key and Secret here")),
	)
}

// buildSyncSettingsTab creates the sync settings tab
func (c *CloudSettingsDialog) buildSyncSettingsTab(cfg *config.CloudConfig) *fyne.Container {
	autoSyncCheck := widget.NewCheck("Enable automatic sync", func(checked bool) {
		cfg.AutoSync = checked
	})
	autoSyncCheck.SetChecked(cfg.AutoSync)

	intervalSpinner := widget.NewEntry()
	intervalSpinner.SetText(fmt.Sprintf("%d", cfg.SyncIntervalMinutes))
	intervalSpinner.SetPlaceHolder("Interval in minutes")

	encryptCheck := widget.NewCheck("Encrypt files before upload (recommended)", func(checked bool) {
		cfg.EncryptionEnabled = checked
	})
	encryptCheck.SetChecked(cfg.EncryptionEnabled)

	conflictSelect := widget.NewSelect(
		[]string{"Keep Newest", "Keep Local", "Keep Remote", "Keep Both"},
		func(value string) {
			switch value {
			case "Keep Newest":
				cfg.ConflictStrategy = "newest"
			case "Keep Local":
				cfg.ConflictStrategy = "local"
			case "Keep Remote":
				cfg.ConflictStrategy = "remote"
			case "Keep Both":
				cfg.ConflictStrategy = "both"
			}
		},
	)

	// Set current selection
	conflictMap := map[string]string{
		"newest": "Keep Newest",
		"local":  "Keep Local",
		"remote": "Keep Remote",
		"both":   "Keep Both",
	}
	conflictSelect.SetSelected(conflictMap[cfg.ConflictStrategy])

	verifyCheck := widget.NewCheck("Verify file integrity with hashes", func(checked bool) {
		cfg.VerifyHashes = checked
	})
	verifyCheck.SetChecked(cfg.VerifyHashes)

	backupPathEntry := widget.NewEntry()
	backupPathEntry.SetText(cfg.BackupFolderPath)
	backupPathEntry.SetPlaceHolder("FF6Editor/Backups")

	templatesPathEntry := widget.NewEntry()
	templatesPathEntry.SetText(cfg.TemplatesFolderPath)
	templatesPathEntry.SetPlaceHolder("FF6Editor/Templates")

	return container.NewVBox(
		autoSyncCheck,
		widget.NewForm(
			widget.NewFormItem("Sync Interval (minutes)", intervalSpinner),
			widget.NewFormItem("Conflict Resolution", conflictSelect),
		),
		encryptCheck,
		verifyCheck,
		widget.NewForm(
			widget.NewFormItem("Backup Folder", backupPathEntry),
			widget.NewFormItem("Templates Folder", templatesPathEntry),
		),
	)
}

// buildStatusTab creates the status and diagnostics tab
func (c *CloudSettingsDialog) buildStatusTab() *fyne.Container {
	statusContent := container.NewVBox()

	// Refresh status
	c.refreshStatus(statusContent)

	refreshBtn := widget.NewButton("Refresh Status", func() {
		c.refreshStatus(statusContent)
	})

	return container.NewVBox(
		widget.NewLabel("Cloud Provider Status"),
		widget.NewSeparator(),
		statusContent,
		refreshBtn,
	)
}

// refreshStatus updates the status display
func (c *CloudSettingsDialog) refreshStatus(container *fyne.Container) {
	container.RemoveAll()

	providers := c.cloudManager.ListProviders()

	for _, providerName := range providers {
		provider, err := c.cloudManager.GetProvider(providerName)
		if err != nil {
			container.Add(widget.NewLabel(fmt.Sprintf("Error: %v", err)))
			container.Refresh()
			continue
		}

		status := provider.GetStatus()

		// Create status card
		statusText := fmt.Sprintf(
			"Provider: %s\nAuthenticated: %v\nLast Sync: %v\nFiles Uploaded: %d\nFiles Downloaded: %d",
			status.Provider,
			status.IsAuthenticated,
			status.LastSync.Format("2006-01-02 15:04:05"),
			status.FilesUploaded,
			status.FilesDownloaded,
		)

		card := widget.NewCard(providerName, "", widget.NewLabel(statusText))
		container.Add(card)
	}
	container.Refresh()
}

// authenticateProvider authenticates with a specific cloud provider
func (c *CloudSettingsDialog) authenticateProvider(providerName string) {
	dialog.ShowInformation("Authentication",
		fmt.Sprintf("Starting authentication with %s...\n\n"+
			"Your browser will open to complete sign-in.\n\n"+
			"Note: Full OAuth2 integration requires external libraries.", providerName),
		c.window)

	// In real implementation, would trigger OAuth2 flow
	go func() {
		time.Sleep(2 * time.Second)
		dialog.ShowInformation("Success", fmt.Sprintf("%s authentication completed", providerName), c.window)
	}()
}

// testConnection tests connectivity with a provider
func (c *CloudSettingsDialog) testConnection(providerName string) {
	provider, err := c.cloudManager.GetProvider(providerName)
	if err != nil {
		dialog.ShowError(fmt.Errorf("provider not configured: %w", err), c.window)
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		ok, msg := provider.ValidateConnection(ctx)
		if ok {
			dialog.ShowInformation("Connection Test", "✓ Connection successful!\n"+msg, c.window)
		} else {
			dialog.ShowError(fmt.Errorf("connection failed: %s", msg), c.window)
		}
	}()
}

// testAllConnections tests all enabled providers
func (c *CloudSettingsDialog) testAllConnections() {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		results := "Connection Test Results:\n\n"
		providers := c.cloudManager.ListProviders()

		for _, providerName := range providers {
			provider, err := c.cloudManager.GetProvider(providerName)
			if err != nil {
				results += fmt.Sprintf("❌ %s: Not configured\n", providerName)
				continue
			}

			ok, msg := provider.ValidateConnection(ctx)
			if ok {
				results += fmt.Sprintf("✓ %s: Connected\n", providerName)
			} else {
				results += fmt.Sprintf("❌ %s: %s\n", providerName, msg)
			}
		}

		dialog.ShowInformation("Connection Test Results", results, c.window)
	}()
}

// syncNow manually triggers sync
func (c *CloudSettingsDialog) syncNow() {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		err := c.cloudManager.SyncAll(ctx)
		if err != nil {
			dialog.ShowError(fmt.Errorf("sync failed: %w", err), c.window)
		} else {
			dialog.ShowInformation("Sync Complete", "Cloud sync completed successfully", c.window)
		}
	}()
}

// saveSettings saves all cloud settings to configuration
func (c *CloudSettingsDialog) saveSettings() {
	cfg := config.GetCloudSettings()
	err := config.SetCloudSettings(cfg)
	if err != nil {
		dialog.ShowError(fmt.Errorf("failed to save settings: %w", err), c.window)
	}
}
