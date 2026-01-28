package editors

import (
	"strings"

	"ffvi_editor/models"
	"ffvi_editor/models/consts/pr"
	"ffvi_editor/ui/forms/inputs"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
)

type (
	Character struct {
		widget.BaseWidget
		c              *models.Character
		name           binding.String
		isEnabled      binding.Bool
		level          inputs.IntEntryBinding
		exp            inputs.IntEntryBinding
		currentHP      inputs.IntEntryBinding
		maxHP          inputs.IntEntryBinding
		currentMP      inputs.IntEntryBinding
		maxMP          inputs.IntEntryBinding
		strength       inputs.IntEntryBinding
		stamina        inputs.IntEntryBinding
		agility        inputs.IntEntryBinding
		magic          inputs.IntEntryBinding
		currentHPEntry *inputs.IntEntry
		maxHPEntry     *inputs.IntEntry
		currentMPEntry *inputs.IntEntry
		maxMPEntry     *inputs.IntEntry
		warningLabel   *widget.Label
		statusEffects  fyne.CanvasObject
		// Initial values for undo
		initialName      string
		initialLevel     int
		initialExp       int
		initialCurrentHP int
		initialMaxHP     int
		initialCurrentMP int
		initialMaxMP     int
		initialStrength  int
		initialStamina   int
		initialAgility   int
		initialMagic     int
	}
)

func NewCharacter(c *models.Character) *Character {
	e := &Character{
		BaseWidget: widget.BaseWidget{}, c: c, name: binding.BindString(&c.Name),
		isEnabled:     binding.BindBool(&c.IsEnabled),
		level:         inputs.NewIntEntryBinding(&c.Level),
		exp:           inputs.NewIntEntryBinding(&c.Exp),
		currentHP:     inputs.NewIntEntryBinding(&c.HP.Current),
		maxHP:         inputs.NewIntEntryBinding(&c.HP.Max),
		currentMP:     inputs.NewIntEntryBinding(&c.MP.Current),
		maxMP:         inputs.NewIntEntryBinding(&c.MP.Max),
		strength:      inputs.NewIntEntryBinding(&c.Vigor),
		stamina:       inputs.NewIntEntryBinding(&c.Stamina),
		agility:       inputs.NewIntEntryBinding(&c.Speed),
		magic:         inputs.NewIntEntryBinding(&c.Magic),
		warningLabel:  widget.NewLabel(""),
		statusEffects: nil, // Will be set after initialization
		// Backup initial values
		initialName:      c.Name,
		initialLevel:     c.Level,
		initialExp:       c.Exp,
		initialCurrentHP: c.HP.Current,
		initialMaxHP:     c.HP.Max,
		initialCurrentMP: c.MP.Current,
		initialMaxMP:     c.MP.Max,
		initialStrength:  c.Vigor,
		initialStamina:   c.Stamina,
		initialAgility:   c.Speed,
		initialMagic:     c.Magic}
	e.ExtendBaseWidget(e)
	e.statusEffects = e.createStatusEffects()
	return e
}

func (e *Character) CreateRenderer() fyne.WidgetRenderer {
	name := widget.NewEntryWithData(e.name)
	name.Validator = nil
	e.currentHPEntry = inputs.NewIntEntryWithBinding(e.currentHP)
	e.maxHPEntry = inputs.NewIntEntryWithBinding(e.maxHP)
	e.currentMPEntry = inputs.NewIntEntryWithBinding(e.currentMP)
	e.maxMPEntry = inputs.NewIntEntryWithBinding(e.maxMP)

	// Add validation
	e.currentHPEntry.OnChanged = func(s string) { e.validateHPMP() }
	e.maxHPEntry.OnChanged = func(s string) { e.validateHPMP() }
	e.currentMPEntry.OnChanged = func(s string) { e.validateHPMP() }
	e.maxMPEntry.OnChanged = func(s string) { e.validateHPMP() }

	left := container.NewVBox(
		inputs.NewLabeledEntry("Name:", name),
		inputs.NewLabeledEntry("Experience:", inputs.NewIntEntryWithBinding(e.exp)),
		inputs.NewLabeledEntry("Level:", inputs.NewIntEntryWithBinding(e.level)),
		inputs.NewLabeledEntry("HP Current/Max:", container.NewGridWithColumns(2,
			e.currentHPEntry,
			e.maxHPEntry)),
		inputs.NewLabeledEntry("MP Current/Max:", container.NewGridWithColumns(2,
			e.currentMPEntry,
			e.maxMPEntry)),
		inputs.NewLabeledEntry("Strength:", inputs.NewIntEntryWithBinding(e.strength)),
		inputs.NewLabeledEntry("Agility:", inputs.NewIntEntryWithBinding(e.agility)),
		inputs.NewLabeledEntry("Cheats:", container.NewVBox(
			container.NewHBox(
				widget.NewButton("Max Stats", func() {
					e.maxHP.Set(9999)
					e.currentHP.Set(9999)
					e.maxMP.Set(999)
					e.currentMP.Set(999)
					e.strength.Set(255)
					e.stamina.Set(255)
					e.agility.Set(255)
					e.magic.Set(255)
					e.level.Set(99)
					e.exp.Set(1848184) // Max exp for level 99
				}),
				widget.NewButton("Starter Kit", func() {
					e.maxHP.Set(1000)
					e.currentHP.Set(1000)
					e.maxMP.Set(200)
					e.currentMP.Set(200)
					e.strength.Set(50)
					e.stamina.Set(50)
					e.agility.Set(50)
					e.magic.Set(50)
					e.level.Set(20)
					e.exp.Set(20000)
				}),
				widget.NewButton("Heal", func() {
					maxHP := e.maxHPEntry.Int()
					e.currentHPEntry.SetInt(maxHP)
					maxMP := e.maxMPEntry.Int()
					e.currentMPEntry.SetInt(maxMP)
				}),
			),
			container.NewHBox(
				widget.NewButton("High Stats", func() {
					e.maxHP.Set(5000)
					e.currentHP.Set(5000)
					e.maxMP.Set(500)
					e.currentMP.Set(500)
					e.strength.Set(150)
					e.stamina.Set(150)
					e.agility.Set(150)
					e.magic.Set(150)
					e.level.Set(50)
					e.exp.Set(500000)
				}),
				widget.NewButton("Best Equip", func() {
					// Best equipment IDs
					e.c.Equipment.WeaponID = 122 // Ultima Weapon
					e.c.Equipment.ShieldID = 211 // Genji Shield
					e.c.Equipment.HelmetID = 237 // Crystal Helm
					e.c.Equipment.ArmorID = 266  // Genji Armor
					e.c.Equipment.Relic1ID = 301 // Ribbon
					e.c.Equipment.Relic2ID = 305 // Celestriad
					// Items will be added to inventory by saver
				}),
				widget.NewButton("Magitek", func() {
					// Set command to Magitek
					if len(e.c.Commands) > 0 {
						e.c.Commands[0] = pr.CommandLookupByValue[7] // Magitek
					}
					// Learn Magitek spells
					magitekSpells := []int{40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50, 51} // Fire to Thundaga
					for _, id := range magitekSpells {
						if spell, exists := e.c.SpellsByID[id]; exists {
							spell.Value = 100
						}
					}
				}),
				widget.NewButton("Dance Fix", func() {
					// Set speed to max to make Mog's dances never fail (only for Mog)
					if e.c.ID == 16 { // Mog
						e.agility.Set(255)
					}
				}),
			),
			container.NewHBox(
				widget.NewButton("Reset Character", func() {
					e.name.Set(e.initialName)
					e.level.Set(e.initialLevel)
					e.exp.Set(e.initialExp)
					e.currentHP.Set(e.initialCurrentHP)
					e.maxHP.Set(e.initialMaxHP)
					e.currentMP.Set(e.initialCurrentMP)
					e.maxMP.Set(e.initialMaxMP)
					e.strength.Set(e.initialStrength)
					e.stamina.Set(e.initialStamina)
					e.agility.Set(e.initialAgility)
					e.magic.Set(e.initialMagic)
					// Equipment not reset, as it's separate
				}),
			),
			container.NewHBox(
				widget.NewButton("Basic Equip", func() {
					// Decent equipment IDs
					e.c.Equipment.WeaponID = 104 // Mythril Sword
					e.c.Equipment.ShieldID = 203 // Mythril Shield
					e.c.Equipment.HelmetID = 227 // Mythril Helm
					e.c.Equipment.ArmorID = 252  // Mythril Mail
					e.c.Equipment.Relic1ID = 200 // Empty
					e.c.Equipment.Relic2ID = 200
				}),
				widget.NewButton("Reset Exp", func() {
					e.exp.Set(0)
				}),
				widget.NewButton("Reset HP/MP", func() {
					e.maxHP.Set(0)
					e.currentHP.Set(0)
					e.maxMP.Set(0)
					e.currentMP.Set(0)
				}),
			),
		)),
		inputs.NewLabeledEntry("Stamina:", inputs.NewIntEntryWithBinding(e.stamina)),
		inputs.NewLabeledEntry("Magic:", inputs.NewIntEntryWithBinding(e.magic)),
		e.warningLabel,
		e.statusEffects,
	)
	right := container.NewGridWithRows(2, container.NewVScroll(widget.NewRichTextWithText(lvlToExp)))
	return widget.NewSimpleRenderer(container.NewBorder(nil, nil, left, right))
}

func (e *Character) validateHPMP() {
	var warnings []string
	currentHP := e.currentHPEntry.Int()
	maxHP := e.maxHPEntry.Int()
	currentMP := e.currentMPEntry.Int()
	maxMP := e.maxMPEntry.Int()

	if currentHP > maxHP && maxHP > 0 {
		warnings = append(warnings, "Current HP > Max HP")
	}
	if currentMP > maxMP && maxMP > 0 {
		warnings = append(warnings, "Current MP > Max MP")
	}
	if len(warnings) > 0 {
		e.warningLabel.SetText("Warning: " + strings.Join(warnings, ", "))
	} else {
		e.warningLabel.SetText("")
	}
}

func (e *Character) createStatusEffects() fyne.CanvasObject {
	container := container.NewVBox()
	container.Add(widget.NewLabel("Status Effects:"))

	for _, se := range e.c.StatusEffects {
		check := widget.NewCheck(se.Name, func(checked bool) {
			se.Checked = checked
		})
		check.SetChecked(se.Checked)
		container.Add(check)
	}

	return container
}

const (
	lvlToExp = `Level - Experience    
01 - 0
02 - 32
03 - 96
04 - 208
05 - 400
06 - 672
07 - 1056
08 - 1552
09 - 2184
10 - 2976
11 - 3936
12 - 5080
13 - 6432
14 - 7992
15 - 9784
16 - 11840
17 - 14152
18 - 16736
19 - 19616
20 - 22832
21 - 26360
22 - 30232
23 - 24456
24 - 39056
25 - 44072
26 - 49464
27 - 55288
28 - 61568
29 - 68304
30 - 75496
31 - 93184
32 - 91384
33 - 100083
34 - 108344
35 - 119136
36 - 129504
37 - 140464
38 - 152008
39 - 164184
40 - 176976
41 - 190416
42 - 204520
43 - 219320
44 - 234808
45 - 251000
46 - 267936
47 - 285600
48 - 304040
49 - 323248
50 - 343248
51 - 364064
52 - 385696
53 - 408160
54 - 431488
55 - 455680
56 - 480776
57 - 506760
58 - 533680
59 - 561528
60 - 590320
61 - 620096
62 - 650840
63 - 682600
64 - 715368
65 - 749160
66 - 784016
67 - 819920
68 - 856920
69 - 895016
70 - 934208
71 - 974536
72 - 1016000
73 - 1058640
74 - 1102456
75 - 1147456
76 - 1193648
77 - 1241080
78 - 1289744
79 - 1339672
80 - 1390872
81 - 1443368
82 - 1497160
83 - 1553364
84 - 1608712
85 - 1666512
86 - 1725688
87 - 1786240
88 - 1848184
89 - 1911552
90 - 1976352
91 - 2042608
92 - 2110320
93 - 2179504
94 - 2250192
95 - 2322392
96 - 2396128
97 - 2471400
98 - 2548224
99 - 2637112`
)
