package tiled

import "encoding/json"

// Map представляет корневой объект карты Tiled.
type Map struct {
	BackgroundColor  string      `json:"backgroundcolor,omitempty"`  // Hex-color (#RRGGBB or #AARRGGBB)
	Class            string      `json:"class,omitempty"`            // (since 1.9)
	CompressionLevel int         `json:"compressionlevel,omitempty"` // Default: -1
	Height           int         `json:"height"`                     // Number of tile rows
	HexSideLength    int         `json:"hexsidelength,omitempty"`    // Hexagonal maps only
	Infinite         bool        `json:"infinite"`
	Layers           []Layer     `json:"layers"`
	NextLayerID      int         `json:"nextlayerid"`
	NextObjectID     int         `json:"nextobjectid"`
	Orientation      string      `json:"orientation"` // orthogonal, isometric, staggered, hexagonal
	ParallaxOriginX  float64     `json:"parallaxoriginx,omitempty"`
	ParallaxOriginY  float64     `json:"parallaxoriginy,omitempty"`
	Properties       []Property  `json:"properties,omitempty"`
	RenderOrder      string      `json:"renderorder,omitempty"`  // right-down, right-up, left-down, left-up
	StaggerAxis      string      `json:"staggeraxis,omitempty"`  // x or y
	StaggerIndex     string      `json:"staggerindex,omitempty"` // odd or even
	TiledVersion     string      `json:"tiledversion,omitempty"`
	TileHeight       int         `json:"tileheight"`
	Tilesets         []Tileset   `json:"tilesets"`
	TileWidth        int         `json:"tilewidth"`
	Type             string      `json:"type,omitempty"`    // "map"
	Version          interface{} `json:"version,omitempty"` // String (since 1.6) or Number (older)
	Width            int         `json:"width"`             // Number of tile columns
}

// Property представляет пользовательское свойство (Key-Value).
type Property struct {
	Name         string      `json:"name"`
	Type         string      `json:"type,omitempty"` // string, int, float, bool, color, file, object, class
	PropertyType string      `json:"propertytype,omitempty"`
	Value        interface{} `json:"value"`
}

// Layer объединяет поля для всех типов слоев (tilelayer, objectgroup, imagelayer, group).
// Некоторые поля специфичны только для определенных типов.
type Layer struct {
	// Common fields
	ID         int        `json:"id"`
	Class      string     `json:"class,omitempty"`
	Name       string     `json:"name"`
	Opacity    float64    `json:"opacity"`
	ParallaxX  float64    `json:"parallaxx,omitempty"` // Default: 1
	ParallaxY  float64    `json:"parallaxy,omitempty"` // Default: 1
	Properties []Property `json:"properties,omitempty"`
	TintColor  string     `json:"tintcolor,omitempty"`
	Type       string     `json:"type"` // tilelayer, objectgroup, imagelayer, group
	Visible    bool       `json:"visible"`
	X          int        `json:"x"`
	Y          int        `json:"y"`
	OffsetX    float64    `json:"offsetx,omitempty"`
	OffsetY    float64    `json:"offsety,omitempty"`
	Locked     bool       `json:"locked,omitempty"`

	// Tile Layer specific
	Chunks      []Chunk         `json:"chunks,omitempty"`
	Compression string          `json:"compression,omitempty"` // zlib, gzip, zstd or empty
	Data        json.RawMessage `json:"data,omitempty"`        // []uint32 (array) or string (base64)
	Encoding    string          `json:"encoding,omitempty"`    // csv or base64
	Height      int             `json:"height,omitempty"`
	Width       int             `json:"width,omitempty"`
	StartX      int             `json:"startx,omitempty"`
	StartY      int             `json:"starty,omitempty"`

	// Object Group specific
	DrawOrder string   `json:"draworder,omitempty"` // topdown, index
	Objects   []Object `json:"objects,omitempty"`

	// Image Layer specific
	Image            string `json:"image,omitempty"`
	ImageHeight      int    `json:"imageheight,omitempty"`
	ImageWidth       int    `json:"imagewidth,omitempty"`
	RepeatX          bool   `json:"repeatx,omitempty"`
	RepeatY          bool   `json:"repeaty,omitempty"`
	TransparentColor string `json:"transparentcolor,omitempty"`

	// Group Layer specific
	Layers []Layer `json:"layers,omitempty"` // Recursive definition
}

// Chunk используется для бесконечных карт.
type Chunk struct {
	Data   json.RawMessage `json:"data"` // []uint32 or string (base64)
	Height int             `json:"height"`
	Width  int             `json:"width"`
	X      int             `json:"x"`
	Y      int             `json:"y"`
}

// Object представляет объект на слое объектов.
type Object struct {
	ID         int        `json:"id"`
	GID        uint32     `json:"gid,omitempty"` // Global Tile ID (if tile object)
	Name       string     `json:"name"`
	Type       string     `json:"type,omitempty"`
	Class      string     `json:"class,omitempty"` // Same as Type in newer versions
	X          float64    `json:"x"`
	Y          float64    `json:"y"`
	Width      float64    `json:"width,omitempty"`
	Height     float64    `json:"height,omitempty"`
	Rotation   float64    `json:"rotation"`
	Visible    bool       `json:"visible"`
	Properties []Property `json:"properties,omitempty"`

	// Shape markers
	Ellipse  bool    `json:"ellipse,omitempty"`
	Point    bool    `json:"point,omitempty"`
	Polygon  []Point `json:"polygon,omitempty"`
	Polyline []Point `json:"polyline,omitempty"`

	// Text object
	Text *Text `json:"text,omitempty"`

	// Template reference
	Template string `json:"template,omitempty"`
}

// Point координаты точки (для Polygon/Polyline).
type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// Text свойства текстового объекта.
type Text struct {
	Text       string `json:"text"`
	Bold       bool   `json:"bold,omitempty"`
	Italic     bool   `json:"italic,omitempty"`
	Underline  bool   `json:"underline,omitempty"`
	Strikeout  bool   `json:"strikeout,omitempty"`
	Kerning    bool   `json:"kerning,omitempty"` // Default: true
	Wrap       bool   `json:"wrap,omitempty"`
	Color      string `json:"color,omitempty"`
	FontFamily string `json:"fontfamily,omitempty"`
	PixelSize  int    `json:"pixelsize,omitempty"` // Default: 16
	HAlign     string `json:"halign,omitempty"`    // left, center, right, justify
	VAlign     string `json:"valign,omitempty"`    // top, center, bottom
}

// Tileset описывает набор тайлов.
type Tileset struct {
	BackgroundColor  string          `json:"backgroundcolor,omitempty"`
	Class            string          `json:"class,omitempty"`
	Columns          int             `json:"columns"`
	FillMode         string          `json:"fillmode,omitempty"` // stretch, preserve-aspect-fit
	FirstGID         uint32          `json:"firstgid"`
	Grid             *Grid           `json:"grid,omitempty"`
	Image            string          `json:"image,omitempty"`
	ImageHeight      int             `json:"imageheight,omitempty"`
	ImageWidth       int             `json:"imagewidth,omitempty"`
	Margin           int             `json:"margin"`
	Name             string          `json:"name"`
	ObjectAlignment  string          `json:"objectalignment,omitempty"`
	Properties       []Property      `json:"properties,omitempty"`
	Source           string          `json:"source,omitempty"` // If external .tsx
	Spacing          int             `json:"spacing"`
	Terrains         []Terrain       `json:"terrains,omitempty"`
	TileCount        int             `json:"tilecount"`
	TiledVersion     string          `json:"tiledversion,omitempty"`
	TileHeight       int             `json:"tileheight"`
	TileOffset       *TileOffset     `json:"tileoffset,omitempty"`
	TileRenderSize   string          `json:"tilerendersize,omitempty"` // tile, grid
	Tiles            []Tile          `json:"tiles,omitempty"`
	TileWidth        int             `json:"tilewidth"`
	Transformations  *Transformation `json:"transformations,omitempty"`
	TransparentColor string          `json:"transparentcolor,omitempty"`
	Type             string          `json:"type,omitempty"`
	Version          json.RawMessage `json:"version,omitempty"`
	WangSets         []WangSet       `json:"wangsets,omitempty"`
}

// Grid настройки сетки для тайлсета.
type Grid struct {
	Height      int    `json:"height"`
	Width       int    `json:"width"`
	Orientation string `json:"orientation"`
}

// TileOffset смещение отрисовки тайлов.
type TileOffset struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// Transformation разрешенные трансформации.
type Transformation struct {
	HFlip               bool `json:"hflip"`
	VFlip               bool `json:"vflip"`
	Rotate              bool `json:"rotate"`
	PreferUntransformed bool `json:"preferuntransformed"`
}

// Tile определение отдельного тайла (если у него есть свойства/анимация).
type Tile struct {
	ID          int        `json:"id"`
	Type        string     `json:"type,omitempty"`
	Image       string     `json:"image,omitempty"`
	ImageHeight int        `json:"imageheight,omitempty"`
	ImageWidth  int        `json:"imagewidth,omitempty"`
	X           int        `json:"x,omitempty"`
	Y           int        `json:"y,omitempty"`
	Width       int        `json:"width,omitempty"`
	Height      int        `json:"height,omitempty"`
	ObjectGroup *Layer     `json:"objectgroup,omitempty"` // Collision shapes
	Probability float64    `json:"probability,omitempty"`
	Properties  []Property `json:"properties,omitempty"`
	Terrain     []int      `json:"terrain,omitempty"` // Deprecated in favor of WangSets
	Animation   []Frame    `json:"animation,omitempty"`
}

// Frame кадр анимации тайла.
type Frame struct {
	Duration int `json:"duration"`
	TileID   int `json:"tileid"`
}

// Terrain тип местности (deprecated, see WangSets).
type Terrain struct {
	Name       string     `json:"name"`
	Properties []Property `json:"properties,omitempty"`
	Tile       int        `json:"tile"`
}

// WangSet набор Wang для автотайлинга.
type WangSet struct {
	Class      string      `json:"class,omitempty"`
	Colors     []WangColor `json:"colors,omitempty"`
	Name       string      `json:"name"`
	Properties []Property  `json:"properties,omitempty"`
	Tile       int         `json:"tile"`
	Type       string      `json:"type"` // corner, edge, mixed
	WangTiles  []WangTile  `json:"wangtiles,omitempty"`
}

// WangColor цвет/тип в наборе Wang.
type WangColor struct {
	Class       string     `json:"class,omitempty"`
	Color       string     `json:"color"`
	Name        string     `json:"name"`
	Probability float64    `json:"probability"`
	Properties  []Property `json:"properties,omitempty"`
	Tile        int        `json:"tile"`
}

// WangTile тайл, размеченный цветами Wang.
type WangTile struct {
	TileID int   `json:"tileid"`
	WangID []int `json:"wangid"` // Usually 8 unsigned bytes/ints
}

// ObjectTemplate структура для внешних файлов шаблонов (.tx).
type ObjectTemplate struct {
	Type    string   `json:"type"` // "template"
	Tileset *Tileset `json:"tileset,omitempty"`
	Object  Object   `json:"object"`
}
