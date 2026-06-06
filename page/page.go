// Package page fetches a single web page and turns it into LLM-friendly input,
// modelled on how Firecrawl scrapes: page metadata, the readable content as
// Markdown (boilerplate removed, GitHub-flavored), and discrete lists of the
// real image and link URLs on the page (so callers cite or display URLs that
// exist rather than guessing them). It reuses scour's uTLS HTTP client via
// engines.FetchURL.
package page

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/base"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/commonmark"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/strikethrough"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/table"
	"github.com/PuerkitoBio/goquery"

	"github.com/JoakimCarlsson/scour/engines"
)

const (
	defaultMaxMarkdown = 12000
	maxImages          = 25
	maxLinks           = 50
)

var blankLinesRe = regexp.MustCompile(`\n{3,}`)
var bgImageRe = regexp.MustCompile(`url\(['"]?([^'")]+)['"]?\)`)

var conv = converter.NewConverter(converter.WithPlugins(
	base.NewBasePlugin(),
	commonmark.NewCommonmarkPlugin(),
	table.NewTablePlugin(),
	strikethrough.NewStrikethroughPlugin(),
))

var excludeNonMainSelectors = strings.Join([]string{
	"header", "footer", "nav", "aside",
	".header", ".top", ".navbar", "#header",
	".footer", ".bottom", "#footer",
	".sidebar", ".side", ".aside", "#sidebar",
	".modal", ".popup", "#modal", ".overlay",
	".ad", ".ads", ".advert", "#ad",
	".lang-selector", ".language", "#language-selector",
	".social", ".social-media", ".social-links", "#social",
	".menu", ".navigation", "#nav",
	".breadcrumbs", "#breadcrumbs",
	".share", "#share",
	".widget", "#widget",
	".cookie", "#cookie",
}, ", ")

// Result is the readable content of a fetched page, in the Firecrawl shape:
// metadata, a Markdown body, and discrete image and link lists.
type Result struct {
	URL         string   `json:"url"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Markdown    string   `json:"markdown"`
	Images      []string `json:"images,omitempty"`
	Links       []string `json:"links,omitempty"`
}

// Fetch retrieves rawURL and extracts its title, description, the main content
// as Markdown (boilerplate stripped), and the image and link URLs on the page.
// maxMarkdown limits the Markdown length (<= 0 uses a default).
func Fetch(ctx context.Context, rawURL string, maxMarkdown int) (*Result, error) {
	if maxMarkdown <= 0 {
		maxMarkdown = defaultMaxMarkdown
	}
	body, finalURL, contentType, err := engines.FetchURL(ctx, rawURL)
	if err != nil && len(body) == 0 {
		return nil, fmt.Errorf("page: fetch %s: %w", rawURL, err)
	}
	if contentType != "" &&
		!strings.Contains(contentType, "html") &&
		!strings.Contains(contentType, "text/") {
		return nil, fmt.Errorf("page: unsupported content-type %q for %s", contentType, finalURL)
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("page: parse %s: %w", finalURL, err)
	}

	baseURL := resolveBase(doc, finalURL)
	res := &Result{
		URL:         finalURL,
		Title:       firstNonEmpty(metaProperty(doc, "og:title"), strings.TrimSpace(doc.Find("title").First().Text())),
		Description: firstNonEmpty(metaProperty(doc, "og:description"), metaName(doc, "description")),
		Images:      extractImages(doc, baseURL),
		Links:       extractLinks(doc, baseURL),
	}

	doc.Find("script, style, noscript, meta, head, svg, iframe, form").Remove()
	doc.Find(excludeNonMainSelectors).Remove()
	doc.Find(`img[src^="data:"]`).Remove()

	html, err := doc.Find("body").Html()
	if err != nil {
		return nil, fmt.Errorf("page: read content %s: %w", finalURL, err)
	}
	md, err := conv.ConvertString(html)
	if err != nil {
		return nil, fmt.Errorf("page: convert %s: %w", finalURL, err)
	}
	res.Markdown = truncateRunes(tidyMarkdown(md), maxMarkdown)
	return res, nil
}

// extractImages collects image URLs the way Firecrawl does: meta images first
// (the canonical hero), then <img> src/data-src/srcset, <picture> sources, icon
// links, inline background-images, and video posters — resolved absolute,
// de-duplicated, and capped. data:/blob: URIs are skipped.
func extractImages(doc *goquery.Document, base *url.URL) []string {
	var out []string
	seen := map[string]bool{}
	add := func(raw string) {
		raw = strings.TrimSpace(raw)
		if raw == "" || strings.HasPrefix(raw, "data:") || strings.HasPrefix(raw, "blob:") || len(out) >= maxImages {
			return
		}
		abs := resolve(base, raw)
		if abs == "" || seen[abs] {
			return
		}
		seen[abs] = true
		out = append(out, abs)
	}
	addSrcset := func(srcset string) {
		for _, part := range strings.Split(srcset, ",") {
			fields := strings.Fields(strings.TrimSpace(part))
			if len(fields) > 0 {
				add(fields[0])
			}
		}
	}

	for _, p := range []string{"og:image", "og:image:url", "og:image:secure_url"} {
		add(metaProperty(doc, p))
	}
	add(metaName(doc, "twitter:image"))
	add(metaName(doc, "twitter:image:src"))
	if v, ok := doc.Find(`meta[itemprop="image"]`).First().Attr("content"); ok {
		add(v)
	}

	doc.Find("img").Each(func(_ int, s *goquery.Selection) {
		if src, ok := s.Attr("src"); ok {
			add(src)
		}
		if ds, ok := s.Attr("data-src"); ok {
			add(ds)
		}
		if ss, ok := s.Attr("srcset"); ok {
			addSrcset(ss)
		}
	})
	doc.Find("picture source").Each(func(_ int, s *goquery.Selection) {
		if ss, ok := s.Attr("srcset"); ok {
			addSrcset(ss)
		}
	})
	doc.Find(`link[rel*="icon"], link[rel*="apple-touch-icon"], link[rel*="image_src"]`).Each(func(_ int, s *goquery.Selection) {
		if href, ok := s.Attr("href"); ok {
			add(href)
		}
	})
	doc.Find(`[style*="background-image"]`).Each(func(_ int, s *goquery.Selection) {
		style, _ := s.Attr("style")
		for _, m := range bgImageRe.FindAllStringSubmatch(style, -1) {
			if len(m) > 1 {
				add(m[1])
			}
		}
	})
	doc.Find("video[poster]").Each(func(_ int, s *goquery.Selection) {
		if p, ok := s.Attr("poster"); ok {
			add(p)
		}
	})
	return out
}

// extractLinks returns absolute http(s)/mailto link URLs on the page, honoring
// <base href>, skipping fragment-only links, de-duplicated and capped.
func extractLinks(doc *goquery.Document, base *url.URL) []string {
	var out []string
	seen := map[string]bool{}
	doc.Find("a[href]").EachWithBreak(func(_ int, s *goquery.Selection) bool {
		href, _ := s.Attr("href")
		href = strings.TrimSpace(href)
		if href == "" || strings.HasPrefix(href, "#") {
			return true
		}
		if strings.HasPrefix(href, "mailto:") {
			if !seen[href] {
				seen[href] = true
				out = append(out, href)
			}
			return len(out) < maxLinks
		}
		abs := resolve(base, href)
		if abs == "" || seen[abs] {
			return true
		}
		seen[abs] = true
		out = append(out, abs)
		return len(out) < maxLinks
	})
	return out
}

// resolveBase returns the URL relative references resolve against: the page's
// <base href> when present, otherwise the final (post-redirect) URL.
func resolveBase(doc *goquery.Document, finalURL string) *url.URL {
	base, _ := url.Parse(finalURL)
	if href, ok := doc.Find("base[href]").First().Attr("href"); ok {
		if u, err := url.Parse(strings.TrimSpace(href)); err == nil {
			if base != nil {
				return base.ResolveReference(u)
			}
			return u
		}
	}
	return base
}

func metaProperty(doc *goquery.Document, property string) string {
	v, _ := doc.Find(`meta[property="` + property + `"]`).First().Attr("content")
	return strings.TrimSpace(v)
}

func metaName(doc *goquery.Document, name string) string {
	v, _ := doc.Find(`meta[name="` + name + `"]`).First().Attr("content")
	return strings.TrimSpace(v)
}

func resolve(base *url.URL, raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	if base != nil {
		u = base.ResolveReference(u)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return ""
	}
	return u.String()
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// tidyMarkdown normalizes newlines and collapses runs of blank lines the
// conversion can leave behind once boilerplate is stripped.
func tidyMarkdown(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = blankLinesRe.ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
