package flowfile

import (
	"maps"
	"path"
	"slices"
	"strings"
)

// AssetsDir is the folder of a project's pictures, at its top, which an
// image's src names by their path in the project, such as
// assets/latte.jpg, so a design shows the project's own pictures.
const AssetsDir = "assets"

// AssetExts are the extensions of the pictures an image may show from
// the assets folder.
var AssetExts = []string{".png", ".jpg", ".jpeg", ".gif", ".webp"}

// AssetPath is the path in the project, with slashes, of the picture an
// image's src names, when it names a file rather than a URL. Any path
// may name one, so the editor can say what is wrong with it.
func AssetPath(src string) (string, bool) {
	if src == "" || strings.Contains(src, "://") || strings.HasPrefix(src, "data:") {
		return "", false
	}
	return path.Clean(strings.TrimPrefix(src, "./")), true
}

// IsAsset reports whether the file at p, relative to the project's
// folder with slashes, is a picture of the assets folder.
func IsAsset(p string) bool {
	p = path.Clean(p)
	return strings.HasPrefix(p, AssetsDir+"/") && slices.Contains(AssetExts, strings.ToLower(path.Ext(p)))
}

// Images are the srcs of els' images, and of those uses change, each
// once, in the order they are first written.
func Images(els []Element) []string {
	var out []string
	add := func(src string) {
		if src != "" && !slices.Contains(out, src) {
			out = append(out, src)
		}
	}
	var walk func(els []Element)
	walk = func(els []Element) {
		for _, el := range els {
			if el.Kind == "image" {
				add(el.Attr("src"))
			}
			if el.Kind == "use" {
				for _, k := range slices.Sorted(maps.Keys(el.Attrs)) {
					if strings.HasSuffix(k, ".src") {
						add(el.Attrs[k])
					}
				}
			}
			walk(el.Children)
		}
	}
	walk(els)
	return out
}
