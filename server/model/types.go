package model

type CategoryChild struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

type CategoryConfig struct {
	Key       string          `json:"key"`
	Label     string          `json:"label"`
	Color     string          `json:"color"`
	SortOrder int             `json:"sort_order"`
	IsSystem  bool            `json:"is_system"`
	Children  []CategoryChild `json:"children"`
}

type CategoryConfigFile struct {
	Categories []CategoryConfig `json:"categories"`
}

type Location struct {
	HID                  string  `json:"hid,omitempty"`
	Name                 string  `json:"name"`
	Address              string  `json:"address,omitempty"`
	Lat                  float64 `json:"lat"`
	Lng                  float64 `json:"lng"`
	Category             string  `json:"category"`
	ChildCategory        string  `json:"child_category,omitempty"`
	CategoryDisplay      string  `json:"category_display,omitempty"`
	ChildCategoryDisplay string  `json:"child_category_display,omitempty"`
	Type                 string  `json:"type,omitempty"`
	TypeDisplay          string  `json:"type_display,omitempty"`
	Manager              string  `json:"manager,omitempty"`
	IsPartnerships       bool    `json:"is_partnerships"`
}

type LocationsFile struct {
	Locations []Location `json:"locations"`
}

type GeoItem struct {
	HID        string  `json:"hid"`
	Title      string  `json:"title"`
	Population int     `json:"population"`
	Lat        float64 `json:"lat"`
	Lng        float64 `json:"lng"`
}
