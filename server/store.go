package main

// Rarity levels for cosmetic items
const (
	RarityCommon    = 0
	RarityRare      = 1
	RarityEpic      = 2
	RarityLegendary = 3
)

// ItemType distinguishes different cosmetic categories
const (
	ItemSkin  = "skin"
	ItemTrail = "trail"
)

// StoreItem represents a purchasable cosmetic item
type StoreItem struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`     // "skin" or "trail"
	Rarity   int    `json:"rarity"`   // 0=common, 1=rare, 2=epic, 3=legendary
	Price    int    `json:"price"`    // in credits
	Color1   string `json:"color1"`   // primary color (hex)
	Color2   string `json:"color2"`   // secondary color (hex), used for trail/accent
	Preview  string `json:"preview"`  // description for UI
}

// StoreCatalog is the full list of purchasable items
var StoreCatalog = []StoreItem{
	// Ship skins - Common (50-100 credits)
	{ID: "skin_crimson", Name: "Crimson", Type: ItemSkin, Rarity: RarityCommon, Price: 50, Color1: "#ff3333", Color2: "#cc0000", Preview: "Deep red hull plating"},
	{ID: "skin_forest", Name: "Forest", Type: ItemSkin, Rarity: RarityCommon, Price: 50, Color1: "#33cc33", Color2: "#006600", Preview: "Jungle green camo"},
	{ID: "skin_ocean", Name: "Ocean", Type: ItemSkin, Rarity: RarityCommon, Price: 50, Color1: "#3399ff", Color2: "#0044aa", Preview: "Deep sea blue"},
	{ID: "skin_sunset", Name: "Sunset", Type: ItemSkin, Rarity: RarityCommon, Price: 75, Color1: "#ff8833", Color2: "#cc4400", Preview: "Warm orange tones"},
	{ID: "skin_purple", Name: "Amethyst", Type: ItemSkin, Rarity: RarityCommon, Price: 75, Color1: "#aa44ff", Color2: "#6600cc", Preview: "Royal purple finish"},

	// Ship skins - Rare (150-250 credits)
	{ID: "skin_gold", Name: "Golden", Type: ItemSkin, Rarity: RarityRare, Price: 150, Color1: "#ffcc00", Color2: "#aa8800", Preview: "Gleaming gold plating"},
	{ID: "skin_ice", Name: "Ice", Type: ItemSkin, Rarity: RarityRare, Price: 150, Color1: "#88ddff", Color2: "#44aacc", Preview: "Frozen crystal coating"},
	{ID: "skin_toxic", Name: "Toxic", Type: ItemSkin, Rarity: RarityRare, Price: 200, Color1: "#88ff00", Color2: "#44aa00", Preview: "Radioactive green glow"},
	{ID: "skin_rose", Name: "Rose", Type: ItemSkin, Rarity: RarityRare, Price: 200, Color1: "#ff66aa", Color2: "#cc3377", Preview: "Pink rose tinted armor"},

	// Ship skins - Epic (400-600 credits)
	{ID: "skin_phantom", Name: "Phantom", Type: ItemSkin, Rarity: RarityEpic, Price: 400, Color1: "#333344", Color2: "#111122", Preview: "Nearly invisible dark hull"},
	{ID: "skin_inferno", Name: "Inferno", Type: ItemSkin, Rarity: RarityEpic, Price: 500, Color1: "#ff4400", Color2: "#ff8800", Preview: "Burning flame pattern"},
	{ID: "skin_arctic", Name: "Arctic", Type: ItemSkin, Rarity: RarityEpic, Price: 500, Color1: "#ffffff", Color2: "#aaddff", Preview: "Pure white ice armor"},

	// Ship skins - Legendary (1000+ credits)
	{ID: "skin_nebula", Name: "Nebula", Type: ItemSkin, Rarity: RarityLegendary, Price: 1000, Color1: "#ff44ff", Color2: "#4444ff", Preview: "Swirling cosmic colors"},
	{ID: "skin_void", Name: "Void", Type: ItemSkin, Rarity: RarityLegendary, Price: 1200, Color1: "#000000", Color2: "#440088", Preview: "Absorbs all light"},

	// Trail effects - Common (50-100 credits)
	{ID: "trail_fire", Name: "Fire Trail", Type: ItemTrail, Rarity: RarityCommon, Price: 75, Color1: "#ff4400", Color2: "#ffaa00", Preview: "Leaves a fiery wake"},
	{ID: "trail_ice", Name: "Ice Trail", Type: ItemTrail, Rarity: RarityCommon, Price: 75, Color1: "#44aaff", Color2: "#88ddff", Preview: "Crystalline ice particles"},

	// Trail effects - Rare (150-250 credits)
	{ID: "trail_neon", Name: "Neon Trail", Type: ItemTrail, Rarity: RarityRare, Price: 200, Color1: "#00ff88", Color2: "#00ffcc", Preview: "Bright neon glow"},
	{ID: "trail_plasma", Name: "Plasma Trail", Type: ItemTrail, Rarity: RarityRare, Price: 200, Color1: "#aa44ff", Color2: "#ff44aa", Preview: "Crackling plasma energy"},

	// Trail effects - Epic (400-600 credits)
	{ID: "trail_rainbow", Name: "Rainbow Trail", Type: ItemTrail, Rarity: RarityEpic, Price: 500, Color1: "#ff0000", Color2: "#0000ff", Preview: "Shifts through all colors"},
	{ID: "trail_star", Name: "Stardust Trail", Type: ItemTrail, Rarity: RarityEpic, Price: 500, Color1: "#ffcc00", Color2: "#ffffff", Preview: "Sparkling star particles"},

	// Trail effects - Legendary (1000+ credits)
	{ID: "trail_void", Name: "Void Trail", Type: ItemTrail, Rarity: RarityLegendary, Price: 1000, Color1: "#220044", Color2: "#000000", Preview: "Dark matter distortion"},
}

// StoreCatalogMap provides O(1) lookup by item ID
var StoreCatalogMap map[string]StoreItem

func init() {
	StoreCatalogMap = make(map[string]StoreItem, len(StoreCatalog))
	for _, item := range StoreCatalog {
		StoreCatalogMap[item.ID] = item
	}
}

// CreditsPerMatch returns the base credits earned for a match
func CreditsPerMatch(kills, assists int, won bool) int {
	credits := 30 + kills*5 + assists*2
	if won {
		credits += 25
	}
	return credits
}
