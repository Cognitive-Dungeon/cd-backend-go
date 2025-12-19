package enums

import "strings"

type ItemCategory uint8

const (
	ItemCategoryUnknown   ItemCategory = iota // 0
	ItemCategoryWeapon                        // 1
	ItemCategoryArmor                         // 2
	ItemCategoryPotion                        // 3
	ItemCategoryFood                          // 4
	ItemCategoryMisc                          // 5
	ItemCategoryContainer                     // 6
)

var itemCategoryToString = map[ItemCategory]string{
	ItemCategoryWeapon:    "WEAPON",
	ItemCategoryArmor:     "ARMOR",
	ItemCategoryPotion:    "POTION",
	ItemCategoryFood:      "FOOD",
	ItemCategoryMisc:      "MISC",
	ItemCategoryContainer: "CONTAINER",
}

var itemCategoryStringToType = map[string]ItemCategory{
	"WEAPON":    ItemCategoryWeapon,
	"ARMOR":     ItemCategoryArmor,
	"POTION":    ItemCategoryPotion,
	"FOOD":      ItemCategoryFood,
	"MISC":      ItemCategoryMisc,
	"CONTAINER": ItemCategoryContainer,
}

func (c ItemCategory) String() string {
	if val, ok := itemCategoryToString[c]; ok {
		return val
	}
	return "UNKNOWN"
}

func ParseItemCategory(s string) ItemCategory {
	upper := strings.ToUpper(s)
	if val, ok := itemCategoryStringToType[upper]; ok {
		return val
	}
	return ItemCategoryUnknown
}
