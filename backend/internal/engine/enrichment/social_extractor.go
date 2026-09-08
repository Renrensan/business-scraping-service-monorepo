package enrichment

import (
	"regexp"
	"strings"
)

var (
	facebookRegex  = regexp.MustCompile(`(?i)https?://(?:www\.)?facebook\.com/(?:pages/[^/]+/\d+|groups/[^/]+|[a-zA-Z0-9._-]+)/?`)
	instagramRegex = regexp.MustCompile(`(?i)https?://(?:www\.)?instagram\.com/([a-zA-Z0-9._-]+)/?`)
	linkedinRegex  = regexp.MustCompile(`(?i)https?://(?:[a-z]{2,3}\.)?linkedin\.com/(?:company/[a-zA-Z0-9._-]+|in/[a-zA-Z0-9._-]+)/?`)
	twitterRegex   = regexp.MustCompile(`(?i)https?://(?:www\.)?(?:twitter\.com|x\.com)/([a-zA-Z0-9_]{1,15})/?`)
	youtubeRegex   = regexp.MustCompile(`(?i)https?://(?:www\.)?youtube\.com/(?:@|c/|channel/|user/)?([a-zA-Z0-9._-]+)/?`)
	tiktokRegex    = regexp.MustCompile(`(?i)https?://(?:www\.)?tiktok\.com/@([a-zA-Z0-9._-]+)/?`)
)

type SocialLinks struct {
	Facebook  string
	Instagram string
	LinkedIn  string
	Twitter   string
	YouTube   string
	TikTok    string
}

func ExtractSocials(htmlContent string) SocialLinks {
	var s SocialLinks

	if match := facebookRegex.FindString(htmlContent); match != "" && !isJunkSocial(match, "facebook.com") {
		s.Facebook = cleanSocialURL(match)
	}
	if match := instagramRegex.FindString(htmlContent); match != "" && !isJunkSocial(match, "instagram.com") {
		s.Instagram = cleanSocialURL(match)
	}
	if match := linkedinRegex.FindString(htmlContent); match != "" && !isJunkSocial(match, "linkedin.com") {
		s.LinkedIn = cleanSocialURL(match)
	}
	if match := twitterRegex.FindString(htmlContent); match != "" && !isJunkSocial(match, "twitter.com") && !isJunkSocial(match, "x.com") {
		s.Twitter = cleanSocialURL(match)
	}
	if match := youtubeRegex.FindString(htmlContent); match != "" && !isJunkSocial(match, "youtube.com") {
		s.YouTube = cleanSocialURL(match)
	}
	if match := tiktokRegex.FindString(htmlContent); match != "" && !isJunkSocial(match, "tiktok.com") {
		s.TikTok = cleanSocialURL(match)
	}

	return s
}

func cleanSocialURL(u string) string {
	u = strings.TrimSpace(u)
	u = strings.Trim(u, `"'`)
	if idx := strings.Index(u, "?"); idx != -1 {
		u = u[:idx]
	}
	return u
}

func isJunkSocial(u, platform string) bool {
	lower := strings.ToLower(u)
	junkWords := []string{"share", "intent", "sharer", "home", "search", "policies", "terms", "privacy", "about", "help", "login", "signup"}
	for _, w := range junkWords {
		if strings.Contains(lower, "/"+w) || strings.Contains(lower, "="+w) {
			return true
		}
	}
	return false
}
