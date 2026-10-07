package sets

// PieceSetInfo describes an available piece set.
type PieceSetInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	IsScidNative bool   `json:"isScidNative"`
	Description string `json:"description"`
}

// BoardThemeInfo describes an available board theme.
type BoardThemeInfo struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Light     string `json:"light"`
	Dark      string `json:"dark"`
	Highlight string `json:"highlight"`
}

// GetAvailablePieceSets returns all supported piece sets (Scid native + online vector sets).
func GetAvailablePieceSets() []PieceSetInfo {
	return []PieceSetInfo{
		{ID: "cburnett", Name: "Colin Burnett (Standard)", IsScidNative: true, Description: "Standard clean modern tournament pieces"},
		{ID: "merida", Name: "Merida", IsScidNative: true, Description: "Classic Shane's Chess Database default piece set"},
		{ID: "alpha", Name: "Alpha", IsScidNative: true, Description: "Clean bold outline style from Scid vs PC"},
		{ID: "leipzig", Name: "Leipzig", IsScidNative: true, Description: "Traditional European book typography set"},
		{ID: "spatial", Name: "Spatial", IsScidNative: true, Description: "Geometric modernist set from Scid vs PC"},
		{ID: "fantasy", Name: "Fantasy", IsScidNative: true, Description: "Playful gothic design from Scid"},
		{ID: "staunty", Name: "Staunty", IsScidNative: false, Description: "High-contrast wooden tournament set"},
		{ID: "cardinal", Name: "Cardinal", IsScidNative: false, Description: "Vibrant and expressive pieces"},
		{ID: "caliente", Name: "Caliente", IsScidNative: false, Description: "Sleek and sharp contours"},
		{ID: "dubrovny", Name: "Dubrovny", IsScidNative: false, Description: "Historic 1950 Dubrovnik Olympiad style"},
		{ID: "governor", Name: "Governor", IsScidNative: false, Description: "Stately classical lines"},
		{ID: "tatiana", Name: "Tatiana", IsScidNative: false, Description: "Graceful curves and ornate knights"},
		{ID: "fresca", Name: "Fresca", IsScidNative: false, Description: "Casual and modern aesthetic"},
		{ID: "riohacha", Name: "Riohacha", IsScidNative: false, Description: "South American chess heritage set"},
		{ID: "kosal", Name: "Kosal", IsScidNative: false, Description: "Minimalist bold shapes"},
		{ID: "shapes", Name: "Shapes", IsScidNative: false, Description: "Geometric abstract representation"},
		{ID: "letter", Name: "Letter / Text", IsScidNative: true, Description: "Pure letter glyph representation"},
		{ID: "mono", Name: "Monochrome", IsScidNative: false, Description: "High-contrast stencil pieces"},
	}
}

// GetAvailableBoardThemes returns all supported board themes from Scid vs PC and modern themes.
func GetAvailableBoardThemes() []BoardThemeInfo {
	return []BoardThemeInfo{
		{ID: "classic-wood", Name: "Classic Wood", Light: "#e3c896", Dark: "#ab7a4e", Highlight: "#f7e597"},
		{ID: "tournament-green", Name: "Tournament Green", Light: "#eeeed2", Dark: "#769656", Highlight: "#baca44"},
		{ID: "blue-white", Name: "Blue-White (Scid)", Light: "#f3f3f3", Dark: "#7389b6", Highlight: "#f3f484"},
		{ID: "brown", Name: "Brown (Scid)", Light: "#d0c0a0", Dark: "#a08050", Highlight: "#e0d873"},
		{ID: "green-sa", Name: "Green-SA (Scid)", Light: "#46a631", Dark: "#006400", Highlight: "#8bb869"},
		{ID: "m-thomas", Name: "M.Thomas (Scid)", Light: "#d3d9a8", Dark: "#51a068", Highlight: "#e0d873"},
		{ID: "grey", Name: "Grey (Scid)", Light: "#9d9d9b", Dark: "#7a7977", Highlight: "#b0c0d0"},
		{ID: "rosy", Name: "Rosy (Scid)", Light: "#f8dbcc", Dark: "#bc8f8f", Highlight: "#e8c0b0"},
		{ID: "steelblue", Name: "SteelBlue (Scid)", Light: "#b0c4de", Dark: "#4682b4", Highlight: "#98fb98"},
		{ID: "walnut", Name: "Walnut", Light: "#ebd3a8", Dark: "#b88b4a", Highlight: "#f5d77f"},
		{ID: "dark-wood", Name: "Dark Wood", Light: "#c7a982", Dark: "#7c5835", Highlight: "#e8bc6a"},
		{ID: "slate", Name: "Slate", Light: "#c2c6cd", Dark: "#69798e", Highlight: "#94b0c2"},
	}
}
