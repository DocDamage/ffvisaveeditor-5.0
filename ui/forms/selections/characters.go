package selections

import (
	"ffvi_editor/models/pr"
	"ffvi_editor/ui/forms/editors"
	"ffvi_editor/ui/forms/inputs"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type (
	Characters struct {
		widget.BaseWidget
		top    *fyne.Container
		middle *fyne.Container
	}
)

func NewCharacters() *Characters {
	s := &Characters{
		top:    container.NewVBox(),
		middle: container.NewStack(),
	}
	s.ExtendBaseWidget(s)

	// Character selector
	charSelect := inputs.NewLabeledEntry("Character:", widget.NewSelect(pr.CharacterNamesHumanSelect(), func(name string) {
		s.middle.RemoveAll()
		c := pr.GetCharacter(name)
		s.middle.Add(container.NewAppTabs(
			container.NewTabItem("Stats", editors.NewCharacter(c)),
			container.NewTabItem("Magic", editors.NewMagic(c)),
			container.NewTabItem("Equipment", editors.NewEquipment(c)),
			container.NewTabItem("Commands", editors.NewCommands(c)),
		))
	}))

	// Global presets
	globalPresets := container.NewHBox(
		widget.NewButton("Max All Characters", func() {
			for _, name := range pr.CharacterNamesHumanSelect() {
				c := pr.GetCharacter(name)
				c.Level = 99
				c.Exp = 1848184
				c.HP.Max = 9999
				c.HP.Current = 9999
				c.MP.Max = 999
				c.MP.Current = 999
				c.Vigor = 255
				c.Stamina = 255
				c.Speed = 255
				c.Magic = 255
			}
		}),
		widget.NewButton("Heal All", func() {
			for _, name := range pr.CharacterNamesHumanSelect() {
				c := pr.GetCharacter(name)
				c.HP.Current = c.HP.Max
				c.MP.Current = c.MP.Max
			}
		}),
		widget.NewButton("Reset All", func() {
			// This would need initial values stored somewhere, for now just basic reset
			for _, name := range pr.CharacterNamesHumanSelect() {
				c := pr.GetCharacter(name)
				c.Level = 1
				c.Exp = 0
				c.HP.Max = 0
				c.HP.Current = 0
				c.MP.Max = 0
				c.MP.Current = 0
				c.Vigor = 0
				c.Stamina = 0
				c.Speed = 0
				c.Magic = 0
			}
		}),
	)

	s.top.Add(container.NewVBox(charSelect, globalPresets))
	return s
}

func (s *Characters) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewBorder(s.top, nil, nil, nil, s.middle))
}
