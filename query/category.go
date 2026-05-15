package query

import "strings"

type Category string

const (
	CategoryGeneral Category = "general"
	CategoryNews    Category = "news"
	CategoryImages  Category = "images"
	CategoryVideos  Category = "videos"
	CategoryMap     Category = "map"
	CategoryMusic   Category = "music"
	CategoryIT      Category = "it"
	CategoryScience Category = "science"
	CategoryFiles   Category = "files"
	CategorySocial  Category = "social"
)

func (c Category) String() string { return string(c) }

var knownCategories = map[string]Category{
	"general": CategoryGeneral,
	"news":    CategoryNews,
	"images":  CategoryImages,
	"videos":  CategoryVideos,
	"map":     CategoryMap,
	"music":   CategoryMusic,
	"it":      CategoryIT,
	"science": CategoryScience,
	"files":   CategoryFiles,
	"social":  CategorySocial,
}

func ParseCategory(s string) (Category, bool) {
	c, ok := knownCategories[strings.ToLower(s)]
	return c, ok
}
