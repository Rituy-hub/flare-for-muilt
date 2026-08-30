package home

import (
	"html/template"
	"net"
	"strings"

	"github.com/soulteary/flare/config/data"
	"github.com/soulteary/flare/config/model"
	"github.com/soulteary/flare/internal/fn"
	"github.com/soulteary/flare/internal/resources/mdi"
)

// isLanEnvironment 判断当前请求主机名是否属于内网环境
func isLanEnvironment(hostname string) bool {
	if hostname == "" {
		return false
	}
	if hostname == "localhost" {
		return true
	}
	if strings.HasSuffix(hostname, ".local") {
		return true
	}
	ip := net.ParseIP(hostname)
	if ip == nil {
		return false
	}
	if ip4 := ip.To4(); ip4 != nil {
		// 10.0.0.0/8
		if ip4[0] == 10 {
			return true
		}
		// 172.16.0.0/12
		if ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31 {
			return true
		}
		// 192.168.0.0/16
		if ip4[0] == 192 && ip4[1] == 168 {
			return true
		}
		// 127.0.0.0/8
		if ip4[0] == 127 {
			return true
		}
	}
	return false
}

// normalizeURL 确保URL包含协议前缀，避免被浏览器当成相对路径
func normalizeURL(url string) string {
	if url == "" {
		return ""
	}
	// 已有协议前缀的直接返回
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") ||
		strings.HasPrefix(url, "ftp://") || strings.HasPrefix(url, "chrome-extension://") ||
		strings.HasPrefix(url, "mailto:") || strings.HasPrefix(url, "tel:") {
		return url
	}
	// 否则自动加上 http:// 前缀
	return "http://" + url
}

// getEffectiveURL 根据内外网环境返回书签应使用的URL
// 内网环境且配置了 LanURL 时优先使用 LanURL，否则使用 URL
func getEffectiveURL(bookmark model.Bookmark) string {
	if bookmark.LanURL != "" && isLanEnvironment(fn.RequestURL.Hostname) {
		return normalizeURL(bookmark.LanURL)
	}
	return normalizeURL(bookmark.URL)
}

func GenerateBookmarkTemplate(filter string, options *model.Application) template.HTML {
	if options == nil {
		op, err := data.GetAllSettingsOptions()
		if err != nil {
			op = model.Application{}
		}
		options = &op
	}
	bookmarksData, err := data.LoadNormalBookmarks()
	if err != nil {
		return template.HTML("")
	}
	b, ok := builderPool.Get().(*strings.Builder)
	if !ok {
		b = &strings.Builder{}
	}
	b.Reset()
	defer builderPool.Put(b)

	n := len(bookmarksData.Items)
	parseBookmarks := make([]model.Bookmark, 0, n)
	for _, bookmark := range bookmarksData.Items {
		bookmark.URL = fn.ParseDynamicUrl(bookmark.URL)
		bookmark.LanURL = fn.ParseDynamicUrl(bookmark.LanURL)
		parseBookmarks = append(parseBookmarks, bookmark)
	}

	bookmarks := parseBookmarks
	if filter != "" {
		bookmarks = make([]model.Bookmark, 0, n)
	}

	if filter != "" {
		filterLower := strings.ToLower(filter)
		for _, bookmark := range parseBookmarks {
			if strings.Contains(strings.ToLower(bookmark.Name), filterLower) || strings.Contains(strings.ToLower(bookmark.URL), filterLower) || strings.Contains(strings.ToLower(bookmark.LanURL), filterLower) {
				bookmarks = append(bookmarks, bookmark)
			}
		}
	}

	if len(bookmarksData.Categories) > 0 {
		defaultCategory := bookmarksData.Categories[0]
		for _, category := range bookmarksData.Categories {
			categoryCopy := category
			renderBookmarksWithCategories(b, &bookmarks, &categoryCopy, &defaultCategory, options.OpenBookmarkNewTab, options.EnableEncryptedLink, options.IconMode)
		}
	} else {
		renderBookmarksWithoutCategories(b, &bookmarks, options.OpenBookmarkNewTab, options.EnableEncryptedLink, options.IconMode)
	}

	return template.HTML(b.String())
}

func renderBookmarksWithoutCategories(b *strings.Builder, bookmarks *[]model.Bookmark, OpenBookmarkNewTab bool, EnableEncryptedLink bool, IconMode string) {
	b.WriteString(`<div class="bookmark-group-container pull-left"><ul class="bookmark-list">`)
	for _, bookmark := range *bookmarks {
		templateURL := getEffectiveURL(bookmark)
		if strings.HasPrefix(templateURL, "chrome-extension://") || EnableEncryptedLink {
			templateURL = "/redir/url?go=" + data.Base64EncodeUrl(templateURL)
		}
		templateIcon := mdi.GetIconByName(bookmark.Icon)
		if strings.HasPrefix(bookmark.Icon, "http://") || strings.HasPrefix(bookmark.Icon, "https://") {
			templateIcon = `<img src="` + bookmark.Icon + `"/>`
		} else if bookmark.Icon != "" {
			templateIcon = mdi.GetIconByName(bookmark.Icon)
		} else if IconMode == "FILLING" {
			templateIcon = fn.GetYandexFavicon(getEffectiveURL(bookmark), mdi.GetIconByName(bookmark.Icon))
		}
		if OpenBookmarkNewTab {
			b.WriteString(`<li><a target="_blank" rel="noopener" href="`)
			b.WriteString(templateURL)
			b.WriteString(`" class="bookmark">`)
			b.WriteString(templateIcon)
			b.WriteString(`<span>`)
			b.WriteString(bookmark.Name)
			b.WriteString(`</span></a></li>`)
		} else {
			b.WriteString(`<li><a rel="noopener" href="`)
			b.WriteString(templateURL)
			b.WriteString(`" class="bookmark">`)
			b.WriteString(templateIcon)
			b.WriteString(`<span>`)
			b.WriteString(bookmark.Name)
			b.WriteString(`</span></a></li>`)
		}
	}
	b.WriteString(`</ul></div>`)
}

func renderBookmarksWithCategories(b *strings.Builder, bookmarks *[]model.Bookmark, category *model.Category, defaultCategory *model.Category, OpenBookmarkNewTab bool, EnableEncryptedLink bool, IconMode string) {
	var itemBuf strings.Builder
	for _, bookmark := range *bookmarks {
		templateURL := getEffectiveURL(bookmark)
		if strings.HasPrefix(templateURL, "chrome-extension://") || EnableEncryptedLink {
			templateURL = "/redir/url?go=" + data.Base64EncodeUrl(templateURL)
		}
		templateIcon := mdi.GetIconByName(bookmark.Icon)
		if strings.HasPrefix(bookmark.Icon, "http://") || strings.HasPrefix(bookmark.Icon, "https://") {
			templateIcon = `<img src="` + bookmark.Icon + `"/>`
		} else if bookmark.Icon != "" {
			templateIcon = mdi.GetIconByName(bookmark.Icon)
		} else if IconMode == "FILLING" {
			templateIcon = fn.GetYandexFavicon(getEffectiveURL(bookmark), mdi.GetIconByName(bookmark.Icon))
		}
		matched := false
		if bookmark.Category != "" {
			matched = bookmark.Category == category.ID
		} else {
			matched = category.ID == defaultCategory.ID
		}
		if !matched {
			continue
		}
		if OpenBookmarkNewTab {
			itemBuf.WriteString(`<li><a target="_blank" rel="noopener" href="`)
		} else {
			itemBuf.WriteString(`<li><a rel="noopener" href="`)
		}
		itemBuf.WriteString(templateURL)
		itemBuf.WriteString(`" class="bookmark">`)
		itemBuf.WriteString(templateIcon)
		itemBuf.WriteString(`<span>`)
		itemBuf.WriteString(bookmark.Name)
		itemBuf.WriteString(`</span></a></li>`)
	}
	if itemBuf.Len() == 0 {
		return
	}
	b.WriteString(`<div class="bookmark-group-container pull-left"><h3 class="bookmark-group-title">`)
	b.WriteString(category.Name)
	b.WriteString(`</h3><ul class="bookmark-list">`)
	b.WriteString(itemBuf.String())
	b.WriteString(`</ul></div>`)
}
