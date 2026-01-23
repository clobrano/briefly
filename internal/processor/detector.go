package processor

import (
	"net/url"
	"strings"

	"github.com/clobrano/briefly/internal/models"
)

func DetectContentType(rawURL string) models.ContentType {
	u, err := url.Parse(rawURL)
	if err != nil {
		return models.ContentTypeUnknown
	}

	host := strings.ToLower(u.Host)

	// YouTube detection
	if strings.Contains(host, "youtube.com") || strings.Contains(host, "youtu.be") {
		return models.ContentTypeYouTube
	}

	// Default to text for any other HTTP(S) URL
	if u.Scheme == "http" || u.Scheme == "https" {
		return models.ContentTypeText
	}

	return models.ContentTypeUnknown
}
