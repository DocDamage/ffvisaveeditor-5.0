package consts

// Landmark represents a key location on the world map for snapping teleport.
type Landmark struct {
	Name  string
	World int     // 1 = Balance, 2 = Ruin
	X     float64 // in-game coordinates (0-256)
	Y     float64 // in-game coordinates (0-256)
}

// Landmarks for both worlds (add more as needed)
var Landmarks = []Landmark{
	{"Narshe", 1, 83.5, 34.4},
	{"Narshe", 2, 83.5, 34.4},
	{"Figaro Castle", 1, 64.6, 79.3},
	{"Figaro Castle", 2, 24, 56},
	{"South Figaro", 1, 84.14, 112.8},
	{"South Figaro", 2, 32, 80},
	{"Returners' Hideout", 1, 104.8, 66.0},
	{"Returners' Hideout", 2, 20, 40},
	{"Kohlingen", 1, 26.3, 39.6},
	{"Kohlingen", 2, 48, 32},
	{"Jidoor", 1, 27.3, 130.8},
	{"Jidoor", 2, 64, 96},
	{"Zozo", 1, 21.0, 93.0},
	{"Zozo", 2, 60, 60},
	{"Thamasa", 1, 249.4, 127.9},
	{"Thamasa", 2, 112, 120},
	{"Vector", 1, 119.9, 187.1},
	{"Vector", 2, 96, 32},
	{"Doma Castle", 1, 155.8, 84.0},
	{"Doma Castle", 2, 104, 24},
	{"Crescent Mountain", 1, 212.6, 148.8},
	{"Crescent Mountain", 2, 120, 80},
	{"Solitary Island", 2, 126, 126},
}
