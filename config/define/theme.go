package define

// TODO：挪到合适地方，拆分 assets

import (
	"html/template"

	"github.com/soulteary/flare/config/data"
	"github.com/soulteary/flare/config/model"
)

// 程序运行默认使用内置的主题配色
var ThemePalettes = getDefaultThemePalettes()
var ThemeCurrent = ""
var ThemePrimaryColor = ""

func Init() {
	initPageInlineStyle()
	UpdatePagePalettes()
	initPagePrimaryColorCache()
}

// 页面内缓存
var _CACHE_PAGE_INLINE_STYLE template.CSS

// 用于mdi
var CACHE_APP_CURRENT_THEME_PRIMARY_COLOR string
var _CACHE_PREV_THEME_NAME string

func GetPageInlineStyle() template.CSS {
	return _CACHE_PAGE_INLINE_STYLE
}

func initPageInlineStyle() {
	if AppFlags.DebugMode {
		return
	}

	_CACHE_PAGE_INLINE_STYLE = template.CSS(PAGE_INLINE_STYLE)
}

func initPagePrimaryColorCache() {
	theme := data.GetThemeName()
	ThemeCurrent = theme
	ThemePrimaryColor = GetThemePrimaryColor(theme)
}

// 页面 body cssvar 样式
var _CACHE_PAGE_BODY_THEME_NAME template.CSS

func GetAppBodyStyle() template.CSS {
	return _CACHE_PAGE_BODY_THEME_NAME
}

func GetThemePrimaryColor(theme string) string {
	if _CACHE_PREV_THEME_NAME == theme {
		return CACHE_APP_CURRENT_THEME_PRIMARY_COLOR
	}
	for _, themePresent := range ThemePalettes {
		if themePresent.Name == theme {
			CACHE_APP_CURRENT_THEME_PRIMARY_COLOR = themePresent.Colors.Primary
			return CACHE_APP_CURRENT_THEME_PRIMARY_COLOR
		}
	}
	return CACHE_APP_CURRENT_THEME_PRIMARY_COLOR
}

const emptyPageBodyStyle = template.CSS(``)

func UpdatePagePalettes() {
	theme := data.GetThemeName()
	for _, themePresent := range ThemePalettes {
		if themePresent.Name == theme {
			_CACHE_PAGE_BODY_THEME_NAME = template.CSS(`--color-background:` + themePresent.Colors.Background + `;--color-primary:` + themePresent.Colors.Primary + `;--color-accent:` + themePresent.Colors.Accent + `;`)
			return
		}
	}
	_CACHE_PAGE_BODY_THEME_NAME = emptyPageBodyStyle
}

func getDefaultThemePalettes() []model.Theme {
	return []model.Theme{
		{
			Name:   "blackboard",
			Colors: model.Palette{Background: "#1a1a1a", Primary: "#FFFDEA", Accent: "#5c5c5c"},
		},
		{
			Name:   "gazette",
			Colors: model.Palette{Background: "#F2F7FF", Primary: "#000000", Accent: "#5c5c5c"},
		},
		{
			Name:   "espresso",
			Colors: model.Palette{Background: "#21211F", Primary: "#D1B59A", Accent: "#4E4E4E"},
		},
		{
			Name:   "cab",
			Colors: model.Palette{Background: "#F6D305", Primary: "#1F1F1F", Accent: "#424242"},
		},
		{
			Name:   "cloud",
			Colors: model.Palette{Background: "#f1f2f0", Primary: "#35342f", Accent: "#37bbe4"},
		},
		{
			Name:   "lime",
			Colors: model.Palette{Background: "#263238", Primary: "#AABBC3", Accent: "#aeea00"},
		},
		{
			Name:   "white",
			Colors: model.Palette{Background: "#ffffff", Primary: "#222222", Accent: "#dddddd"},
		},
		{
			Name:   "tron",
			Colors: model.Palette{Background: "#242B33", Primary: "#EFFBFF", Accent: "#6EE2FF"},
		},
		{
			Name:   "blues",
			Colors: model.Palette{Background: "#2B2C56", Primary: "#EFF1FC", Accent: "#6677EB"},
		},
		{
			Name:   "passion",
			Colors: model.Palette{Background: "#f5f5f5", Primary: "#12005e", Accent: "#8e24aa"},
		},
		{
			Name:   "chalk",
			Colors: model.Palette{Background: "#263238", Primary: "#AABBC3", Accent: "#FF869A"},
		},
		{
			Name:   "paper",
			Colors: model.Palette{Background: "#F8F6F1", Primary: "#4C432E", Accent: "#AA9A73"},
		},
		{
			Name:   "neon",
			Colors: model.Palette{Background: "#091833", Primary: "#EFFBFF", Accent: "#ea00d9"},
		},
		{
			Name:   "pumpkin",
			Colors: model.Palette{Background: "#2d3436", Primary: "#EFFBFF", Accent: "#ffa500"},
		},
		{
			Name:   "onedark",
			Colors: model.Palette{Background: "#282c34", Primary: "#dfd9d6", Accent: "#98c379"},
		},
		{
			Name:   "sunset",
			Colors: model.Palette{Background: "#2D1B3D", Primary: "#FFD89B", Accent: "#FF6B35"},
		},
		{
			Name:   "ocean",
			Colors: model.Palette{Background: "#0A2540", Primary: "#B8E0FF", Accent: "#00B4D8"},
		},
		{
			Name:   "forest",
			Colors: model.Palette{Background: "#1A2F1A", Primary: "#C8E6C9", Accent: "#7CB342"},
		},
		{
			Name:   "rose",
			Colors: model.Palette{Background: "#FCE4EC", Primary: "#880E4F", Accent: "#E91E63"},
		},
		{
			Name:   "slate",
			Colors: model.Palette{Background: "#37474F", Primary: "#CFD8DC", Accent: "#546E7A"},
		},
		{
			Name:   "amber",
			Colors: model.Palette{Background: "#3E2723", Primary: "#FFE0B2", Accent: "#FFB300"},
		},
		{
			Name:   "violet",
			Colors: model.Palette{Background: "#311B92", Primary: "#E1BEE7", Accent: "#7C4DFF"},
		},
		{
			Name:   "mint",
			Colors: model.Palette{Background: "#E0F2F1", Primary: "#004D40", Accent: "#26A69A"},
		},
		{
			Name:   "coral",
			Colors: model.Palette{Background: "#FFF3E0", Primary: "#BF360C", Accent: "#FF5722"},
		},
		{
			Name:   "graphite",
			Colors: model.Palette{Background: "#212121", Primary: "#BDBDBD", Accent: "#616161"},
		},
	}
}
