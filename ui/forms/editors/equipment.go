package editors

import (
	"fmt"
	"strings"

	"ffvi_editor/io/pr"
	"ffvi_editor/models"
	"ffvi_editor/ui/forms/inputs"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type (
	Equipment struct {
		widget.BaseWidget
		c           *models.Character
		search      *widget.Entry
		results     *widget.TextGrid
		weaponEntry *inputs.IntEntry
		shieldEntry *inputs.IntEntry
		helmetEntry *inputs.IntEntry
		armorEntry  *inputs.IntEntry
		relic1Entry *inputs.IntEntry
		relic2Entry *inputs.IntEntry
	}
)

func NewEquipment(c *models.Character) *Equipment {
	e := &Equipment{
		c:       c,
		search:  widget.NewEntry(),
		results: widget.NewTextGrid(),
	}
	e.ExtendBaseWidget(e)
	e.search.OnChanged = func(s string) {
		s = strings.ToLower(s)
		if len(s) < 2 {
			e.results.SetText("")
			return
		}
		var sb strings.Builder
		count := 0
		const maxResults = 20
		if s != "" {
			for k, v := range pr.AllNormalItems {
				if count >= maxResults {
					break
				}
				if strings.Contains(strings.ToLower(v), s) {
					sb.WriteString(fmt.Sprintf("%d - %s\n", k, v))
					count++
				}
			}
		}
		e.results.SetText(sb.String())
	}
	return e
}

func (e *Equipment) CreateRenderer() fyne.WidgetRenderer {
	e.weaponEntry = inputs.NewIntEntryWithData(&e.c.Equipment.WeaponID)
	e.shieldEntry = inputs.NewIntEntryWithData(&e.c.Equipment.ShieldID)
	e.helmetEntry = inputs.NewIntEntryWithData(&e.c.Equipment.HelmetID)
	e.armorEntry = inputs.NewIntEntryWithData(&e.c.Equipment.ArmorID)
	e.relic1Entry = inputs.NewIntEntryWithData(&e.c.Equipment.Relic1ID)
	e.relic2Entry = inputs.NewIntEntryWithData(&e.c.Equipment.Relic2ID)

	// Add validation
	validateItemID := func(entry *inputs.IntEntry) {
		id := entry.Int()
		if id != 0 {
			if _, ok := pr.AllNormalItems[id]; !ok {
				entry.SetInt(0)
			}
		}
	}
	e.weaponEntry.OnChanged = func(s string) { validateItemID(e.weaponEntry) }
	e.shieldEntry.OnChanged = func(s string) { validateItemID(e.shieldEntry) }
	e.helmetEntry.OnChanged = func(s string) { validateItemID(e.helmetEntry) }
	e.armorEntry.OnChanged = func(s string) { validateItemID(e.armorEntry) }
	e.relic1Entry.OnChanged = func(s string) { validateItemID(e.relic1Entry) }
	e.relic2Entry.OnChanged = func(s string) { validateItemID(e.relic2Entry) }

	return widget.NewSimpleRenderer(
		container.NewGridWithColumns(4,
			container.NewGridWithRows(3,
				container.NewVBox(
					inputs.NewLabeledIntEntryWithHint("Weapon:", e.weaponEntry),
					inputs.NewLabeledIntEntryWithHint("Shield:", e.shieldEntry),
				),
				container.NewVBox(
					inputs.NewLabeledIntEntryWithHint("Helmet:", e.helmetEntry),
					inputs.NewLabeledIntEntryWithHint("Armor:", e.armorEntry),
				),
				container.NewVBox(
					inputs.NewLabeledIntEntryWithHint("Relic 1:", e.relic1Entry),
					inputs.NewLabeledIntEntryWithHint("Relic 2:", e.relic2Entry),
				),
			),
			container.NewGridWithRows(3,
				weaponsTextBox,
				helmetTextBox,
				relic1TextBox),
			container.NewGridWithRows(3,
				shieldsTextBox,
				armorTextBox,
				relic2TextBox),
			container.NewBorder(
				inputs.NewLabeledEntry("Find By Name:", e.search), nil, nil, nil,
				container.NewVScroll(e.results))))
}
