// Package scoring ranks leads with admin-defined scoring profiles (for example "GuestChatPro"):
//
//	Score = log10(reviews)×w + (rating − baseline)×w + points per contact channel (capped) + group bonus
//
// and turns the score into a tier (hot / good / low) with per-area cut-offs. The functions here are
// pure; storage and recalculation live in the repository and service layers.
package scoring

import (
	"math"
	"net/url"
	"regexp"
	"sort"
	"strings"
)

// Tiers
const (
	TierHot  = "hot"
	TierGood = "good"
	TierLow  = "low"
	TierSkip = "skip"
)

func ValidTier(t string) bool {
	switch t {
	case TierHot, TierGood, TierLow, TierSkip:
		return true
	}
	return false
}

// Group types (lead-level; the bonus for each comes from the profile)
const (
	GroupIndependent    = "independent"
	GroupGroup          = "group"
	GroupSmallGroup     = "small_group"
	GroupRestaurantStay = "restaurant_stay"
	GroupChain          = "chain"
	GroupSkip           = "skip"
)

var GroupTypes = []string{GroupIndependent, GroupGroup, GroupSmallGroup, GroupRestaurantStay, GroupChain, GroupSkip}

func ValidGroupType(g string) bool {
	for _, t := range GroupTypes {
		if t == g {
			return true
		}
	}
	return false
}

// Channels that can earn points
const (
	ChannelEmail     = "email"
	ChannelWebsite   = "website"
	ChannelInstagram = "instagram"
	ChannelFacebook  = "facebook"
	ChannelTikTok    = "tiktok"
	ChannelLinkedIn  = "linkedin"
	ChannelTwitter   = "twitter"
	ChannelYouTube   = "youtube"
	ChannelPhone     = "phone"
)

var AllChannels = []string{ChannelEmail, ChannelWebsite, ChannelInstagram, ChannelFacebook, ChannelTikTok, ChannelLinkedIn, ChannelTwitter, ChannelYouTube, ChannelPhone}

type Cutoff struct {
	Hot  int `json:"hot"`
	Good int `json:"good"`
}

// Config is a profile's formula. Stored as JSON on the profile.
type Config struct {
	ReviewsWeight      float64            `json:"reviews_weight"`
	RatingBaseline     float64            `json:"rating_baseline"`
	RatingWeight       float64            `json:"rating_weight"`
	ChannelPoints      float64            `json:"channel_points"`
	ChannelCap         float64            `json:"channel_cap"`
	Channels           []string           `json:"channels"`
	GroupBonuses       map[string]float64 `json:"group_bonuses"`
	ChainForcesLow     bool               `json:"chain_forces_low"`
	IgnoreJunkContacts bool               `json:"ignore_junk_contacts"`
	DefaultCutoff      Cutoff             `json:"default_cutoff"`
	AreaCutoffs        map[string]Cutoff  `json:"area_cutoffs"`
}

// GuestChatProConfig is the formula from "GuestChatPro – Approach Priority Score Factors".
func GuestChatProConfig() Config {
	return Config{
		ReviewsWeight:      10,
		RatingBaseline:     4,
		RatingWeight:       10,
		ChannelPoints:      4,
		ChannelCap:         20,
		Channels:           []string{ChannelEmail, ChannelWebsite, ChannelInstagram, ChannelFacebook, ChannelTikTok},
		GroupBonuses:       map[string]float64{GroupGroup: 15, GroupSmallGroup: 8, GroupIndependent: 0, GroupRestaurantStay: -5, GroupChain: -15, GroupSkip: -30},
		ChainForcesLow:     true,
		IgnoreJunkContacts: true,
		DefaultCutoff:      Cutoff{Hot: 50, Good: 38},
		AreaCutoffs: map[string]Cutoff{
			"Ubud": {61, 45}, "Kintamani": {43, 31}, "Jatiluwih": {48, 31}, "Tabanan": {47, 31}, "Pecatu": {50, 38},
			"Kerobokan": {52, 38}, "Umalas": {45, 32}, "Berawa": {52, 38}, "Legian": {64, 44}, "Kuta": {62, 45},
			"Denpasar": {50, 38}, "Sanur": {56, 42}, "Uluwatu": {40, 30}, "Nusa Dua": {47, 34}, "Pererenan": {46, 37},
			"Canggu": {50, 40}, "Seminyak": {59, 45}, "Jimbaran": {53, 42}, "Badung": {49, 31},
		},
	}
}

// Normalize fills missing values so a partly filled config still scores sensibly.
func (c *Config) Normalize() {
	if c.GroupBonuses == nil {
		c.GroupBonuses = map[string]float64{}
	}
	if c.AreaCutoffs == nil {
		c.AreaCutoffs = map[string]Cutoff{}
	}
	if c.Channels == nil {
		c.Channels = []string{}
	}
}

// CutoffFor returns the cut-offs for an area (case-insensitive), or the default.
func (c Config) CutoffFor(area string) Cutoff {
	for name, co := range c.AreaCutoffs {
		if strings.EqualFold(name, area) {
			return co
		}
	}
	return c.DefaultCutoff
}

// Lead is what scoring needs to know about a lead.
type Lead struct {
	Name         string
	Category     string
	Types        []string // Google types, formatted ("Lodging", "Hotel" ...)
	Rating       float64
	ReviewCount  int
	Website      string
	Emails       []string
	Phone        string
	FacebookURL  string
	InstagramURL string
	TikTokURL    string
	LinkedInURL  string
	TwitterURL   string
	YouTubeURL   string
	GroupType    string // effective group type
}

type IgnoredContact struct {
	Channel string `json:"channel"`
	Value   string `json:"value"`
	Reason  string `json:"reason"`
}

// Breakdown explains a score, like the sheet's columns.
type Breakdown struct {
	ReviewPoints  float64          `json:"review_points"`
	RatingPoints  float64          `json:"rating_points"`
	ChannelPoints float64          `json:"channel_points"`
	GroupBonus    float64          `json:"group_bonus"`
	Channels      []string         `json:"channels"`
	Ignored       []IgnoredContact `json:"ignored"`
	NoRating      bool             `json:"no_rating,omitempty"`
	GroupType     string           `json:"group_type"`
	Cutoff        Cutoff           `json:"cutoff"`
	TierReason    string           `json:"tier_reason,omitempty"`
}

type Result struct {
	Raw       float64
	Score     int
	Tier      string
	Breakdown Breakdown
}

// Compute scores a lead with a profile config, using area's cut-offs.
func Compute(l Lead, c Config, area string) Result {
	c.Normalize()
	var b Breakdown

	reviews := l.ReviewCount
	if reviews < 1 {
		reviews = 1
	}
	b.ReviewPoints = math.Log10(float64(reviews)) * c.ReviewsWeight

	// No rating (very new or unrated places): neutral instead of the formula's (0 − 4) × 10 = −40.
	if l.Rating > 0 {
		b.RatingPoints = (l.Rating - c.RatingBaseline) * c.RatingWeight
	} else {
		b.NoRating = true
	}

	channels, ignored := CountChannels(l, c.Channels, c.IgnoreJunkContacts)
	b.Channels = channels
	b.Ignored = ignored
	b.ChannelPoints = float64(len(channels)) * c.ChannelPoints
	if c.ChannelCap > 0 && b.ChannelPoints > c.ChannelCap {
		b.ChannelPoints = c.ChannelCap
	}

	group := l.GroupType
	if !ValidGroupType(group) {
		group = GroupIndependent
	}
	b.GroupType = group
	b.GroupBonus = c.GroupBonuses[group]

	raw := b.ReviewPoints + b.RatingPoints + b.ChannelPoints + b.GroupBonus
	score := int(math.Round(raw))
	cut := c.CutoffFor(area)
	b.Cutoff = cut

	tier := TierLow
	switch {
	case group == GroupSkip:
		tier, b.TierReason = TierSkip, "Not a real target (group type: skip)"
	case group == GroupChain && c.ChainForcesLow:
		tier, b.TierReason = TierLow, "Chain: decisions are made at head office"
	case score >= cut.Hot:
		tier = TierHot
	case score >= cut.Good:
		tier = TierGood
	}

	return Result{Raw: raw, Score: score, Tier: tier, Breakdown: b}
}

// Applies reports whether a profile applies to a lead: it has one of the types (case-insensitive),
// or the profile lists no types.
func Applies(appliesTo []string, l Lead) bool {
	if len(appliesTo) == 0 {
		return true
	}
	for _, want := range appliesTo {
		w := strings.TrimSpace(want)
		if w == "" {
			continue
		}
		if strings.EqualFold(l.Category, w) {
			return true
		}
		for _, t := range l.Types {
			if strings.EqualFold(t, w) {
				return true
			}
		}
	}
	return false
}

// ---------------------------------------------------------------------------------------------
// Contact cleaning

var (
	junkEmailDomains = map[string]bool{
		"example.com": true, "example.org": true, "domain.com": true, "company.com": true, "email.com": true,
		"yourdomain.com": true, "mysite.com": true, "website.com": true, "yourcompany.com": true,
		"wixpress.com": true, "sentry.io": true, "sentry.wixpress.com": true, "sentry-next.wixpress.com": true,
	}
	junkEmailLocals = map[string]bool{
		"yourmail": true, "youremail": true, "your.email": true, "your-email": true, "yourname": true,
		"john": true, "johndoe": true, "john.doe": true, "jane": true, "janedoe": true, "example": true,
		"test": true, "user": true, "username": true, "name": true, "email": true,
	}
	platformDomains = []string{
		"airbnb.", "booking.com", "linktr.ee", "linktree.", "oyorooms.", "oyo.", "whatsapp.com", "wa.me",
		"redbubble.", "mamikos.", "agoda.", "traveloka.", "tripadvisor.", "expedia.", "hotels.com", "trip.com",
		"tiket.com", "pegipegi.", "hostelworld.", "google.com", "goo.gl", "g.page",
	}
	socialDomains   = []string{"facebook.com", "fb.com", "instagram.com", "tiktok.com", "twitter.com", "x.com", "youtube.com", "linkedin.com"}
	platformHandles = map[string]bool{
		"airbnb": true, "booking": true, "bookingcom": true, "booking.com": true, "linktree": true, "linktr.ee": true,
		"oyo": true, "oyorooms": true, "oyo.indonesia": true, "oyoindonesia": true, "whatsapp": true, "redbubble": true,
		"mamikos": true, "wix": true, "wixcom": true, "squarespace": true, "tripadvisor": true, "agoda": true,
		"traveloka": true, "expedia": true, "sharer": true, "share": true, "sharer.php": true, "dialog": true,
	}
	imageFile     = regexp.MustCompile(`(?i)\.(png|jpe?g|gif|webp|svg)$`)
	brokenSocials = []*regexp.Regexp{
		regexp.MustCompile(`(?i)facebook\.com/tr\b`),
		regexp.MustCompile(`(?i)facebook\.com/profile\.php\b`),
		regexp.MustCompile(`(?i)instagram\.com/(p|reel|reels|explore|stories)/`),
		regexp.MustCompile(`(?i)/p/`),
	}
)

func hostOf(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	if !strings.Contains(s, "://") {
		s = "https://" + s
	}
	u, err := url.Parse(s)
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(strings.ToLower(u.Host), "www.")
}

func firstPathSegment(raw string) string {
	s := strings.TrimSpace(raw)
	if !strings.Contains(s, "://") {
		s = "https://" + s
	}
	u, err := url.Parse(s)
	if err != nil {
		return ""
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) == 0 {
		return ""
	}
	return strings.TrimPrefix(strings.ToLower(parts[0]), "@")
}

func hasAnySuffix(host string, list []string) bool {
	for _, d := range list {
		if host == d || strings.HasSuffix(host, "."+d) || strings.HasPrefix(host, d) || strings.Contains(host, "."+d) {
			return true
		}
	}
	return false
}

// JunkEmailReason returns why an email doesn't belong to the property, or "".
func JunkEmailReason(email string) string {
	e := strings.ToLower(strings.TrimSpace(email))
	at := strings.LastIndex(e, "@")
	if at <= 0 || at == len(e)-1 {
		return "not an email address"
	}
	local, domain := e[:at], e[at+1:]
	switch {
	case imageFile.MatchString(e):
		return "image file name, not an email"
	case junkEmailDomains[domain] || strings.HasSuffix(domain, ".wixpress.com") || strings.Contains(domain, "sentry"):
		return "system or template email"
	case junkEmailLocals[local] || strings.Contains(domain, "doe."):
		return "template email"
	case hasAnySuffix(domain, platformDomains):
		return "belongs to a booking or link platform"
	}
	return ""
}

// JunkWebsiteReason returns why a website isn't the property's own site, or "".
func JunkWebsiteReason(site string) string {
	h := hostOf(site)
	if h == "" {
		return "not a valid link"
	}
	if hasAnySuffix(h, platformDomains) {
		return "listing, review or link platform"
	}
	if hasAnySuffix(h, socialDomains) {
		return "social profile, not a website"
	}
	return ""
}

// JunkSocialReason returns why a social link doesn't count, or "".
func JunkSocialReason(link string) string {
	if hostOf(link) == "" {
		return "not a valid link"
	}
	for _, re := range brokenSocials {
		if re.MatchString(link) {
			return "broken or placeholder link"
		}
	}
	if platformHandles[firstPathSegment(link)] {
		return "belongs to a platform, not the property"
	}
	if firstPathSegment(link) == "" {
		return "no profile in link"
	}
	return ""
}

// CountChannels returns the counted channels (in the profile's order) and the contacts ignored as junk.
func CountChannels(l Lead, channels []string, ignoreJunk bool) ([]string, []IgnoredContact) {
	var counted []string
	var ignored []IgnoredContact

	check := func(channel, value string, reason func(string) string) bool {
		if strings.TrimSpace(value) == "" {
			return false
		}
		if ignoreJunk {
			if r := reason(value); r != "" {
				ignored = append(ignored, IgnoredContact{Channel: channel, Value: value, Reason: r})
				return false
			}
		}
		return true
	}

	for _, ch := range channels {
		ok := false
		switch ch {
		case ChannelEmail:
			for _, e := range l.Emails {
				if check(ChannelEmail, e, JunkEmailReason) {
					ok = true
				}
			}
		case ChannelWebsite:
			ok = check(ChannelWebsite, l.Website, JunkWebsiteReason)
		case ChannelInstagram:
			ok = check(ChannelInstagram, l.InstagramURL, JunkSocialReason)
		case ChannelFacebook:
			ok = check(ChannelFacebook, l.FacebookURL, JunkSocialReason)
		case ChannelTikTok:
			ok = check(ChannelTikTok, l.TikTokURL, JunkSocialReason)
		case ChannelLinkedIn:
			ok = check(ChannelLinkedIn, l.LinkedInURL, JunkSocialReason)
		case ChannelTwitter:
			ok = check(ChannelTwitter, l.TwitterURL, JunkSocialReason)
		case ChannelYouTube:
			ok = check(ChannelYouTube, l.YouTubeURL, JunkSocialReason)
		case ChannelPhone:
			ok = strings.TrimSpace(l.Phone) != ""
		}
		if ok {
			counted = append(counted, ch)
		}
	}
	return counted, ignored
}

// ---------------------------------------------------------------------------------------------
// Group suggestions

type groupRule struct {
	re    *regexp.Regexp
	typ   string
	group string
}

func rule(pattern, typ, group string) groupRule {
	return groupRule{regexp.MustCompile(`(?i)` + pattern), typ, group}
}

// Name patterns that suggest a group type. Only suggestions: admins can change them per lead.
// Local groups come from the GuestChatPro doc's examples; chains are national/global brands.
var groupRules = []groupRule{
	// Not a stay
	rule(`\bbeach ?club\b`, GroupSkip, ""),
	rule(`\bsauna\b`, GroupSkip, ""),
	rule(`\boyo\b|\bcapital o\b|\bcollection o\b|\bsuper o\b|\btownhouse oak\b`, GroupSkip, "OYO"),
	rule(`\bkos\b|\bkost\b|\bkos-kosan\b|\bboarding ?house\b|\bmonthly room`, GroupSkip, ""),

	// Local groups (+15) and small groups (+8)
	rule(`\balaya\b`, GroupGroup, "Alaya"),
	rule(`\badiwana\b`, GroupGroup, "Adiwana"),
	rule(`\bini ?vie\b`, GroupGroup, "iNi ViE"),
	rule(`\bsono\b`, GroupGroup, "SONO / Cross Hotels"),
	rule(`\bcross (hotels|collection|vibe)\b|\bx1 villa`, GroupGroup, "SONO / Cross Hotels"),
	rule(`\bmaya (ubud|sanur)\b`, GroupGroup, "Maya"),
	rule(`\bsudamala\b`, GroupGroup, "Sudamala"),
	rule(`\bdouble[- ]six\b`, GroupGroup, "Double Six"),
	rule(`\bluxe estates?\b`, GroupSmallGroup, "Luxe Estates"),
	rule(`\bbreig\b`, GroupSmallGroup, "BREIG"),
	rule(`\bekosistem\b`, GroupSmallGroup, "Ekosistem"),
	rule(`\bnakula\b`, GroupSmallGroup, "Nakula"),
	rule(`\bluxe nomad\b`, GroupSmallGroup, "The Luxe Nomad"),

	// Chains (−15)
	rule(`\bnovotel\b|\bibis\b|\bmercure\b|\bpullman\b|\bsofitel\b|\bmgallery\b|\bm[oö]venpick\b|\bfairmont\b|\braffles\b|\bswiss[oô]tel\b`, GroupChain, "Accor"),
	rule(`\bmarriott\b|\bsheraton\b|\bwestin\b|\bw bali\b|\bst\.? regis\b|\britz[- ]carlton\b|\bfour points\b|\baloft\b|\ble m[eé]ridien\b|\brenaissance\b|\bjw\b|\bluxury collection\b|\bautograph collection\b|\bmoxy\b|\bbulgari\b|\bmandapa\b`, GroupChain, "Marriott"),
	rule(`\bintercontinental\b|\bholiday inn\b|\bcrowne plaza\b|\bhotel indigo\b|\bkimpton\b|\bregent\b|\bsix senses\b|\bvoco\b`, GroupChain, "IHG"),
	rule(`\bhilton\b|\bconrad\b|\bdoubletree\b|\bwaldorf astoria\b|\bcurio collection\b`, GroupChain, "Hilton"),
	rule(`\bhyatt\b|\bandaz\b|\balila\b`, GroupChain, "Hyatt"),
	rule(`\bwyndham\b|\bramada\b|\bdays inn\b|\btryp\b`, GroupChain, "Wyndham"),
	rule(`\bascott\b|\bcitadines\b|\bsomerset\b`, GroupChain, "Ascott"),
	rule(`\bswiss[- ]?bel`, GroupChain, "Swiss-Belhotel"),
	rule(`\bharper\b|\baston\b|\bfave ?hotel\b|\bquest (hotel|san)\b|\bneo hotel\b`, GroupChain, "Archipelago"),
	rule(`\bprime plaza\b`, GroupChain, "Prime Plaza"),
	rule(`\bfour seasons\b`, GroupChain, "Four Seasons"),
	rule(`\bharris\b|\bpop! hotel\b|\bpop hotel\b|\byello hotel\b`, GroupChain, "Tauzia"),
	rule(`\bsantika\b|\bamaris\b`, GroupChain, "Santika"),
	rule(`\bbest western\b`, GroupChain, "Best Western"),
	rule(`\bradisson\b`, GroupChain, "Radisson"),
	rule(`\banantara\b|\bavani\b`, GroupChain, "Minor Hotels"),
	rule(`\bclub med\b`, GroupChain, "Club Med"),
	rule(`\bmeli[aá]\b`, GroupChain, "Meliá"),
	rule(`\bhard rock hotel\b`, GroupChain, "Hard Rock"),

	// Restaurant with a few rooms (−5)
	rule(`\brestaurant\b|\bresto\b|\bwarung\b|\bbistro\b`, GroupRestaurantStay, ""),
}

// SuggestGroup suggests a group type and group name from a lead's name and category.
func SuggestGroup(name, category string) (groupType, groupName string) {
	for _, r := range groupRules {
		if r.re.MatchString(name) {
			return r.typ, r.group
		}
	}
	c := strings.ToLower(category)
	if c == "restaurant" || c == "cafe" || c == "bar" {
		return GroupRestaurantStay, ""
	}
	return GroupIndependent, ""
}

// ---------------------------------------------------------------------------------------------
// Areas

type Area struct {
	Name                             string
	LowLat, LowLng, HighLat, HighLng float64
}

func (a Area) contains(lat, lng float64) bool {
	return lat >= a.LowLat && lat <= a.HighLat && lng >= a.LowLng && lng <= a.HighLng
}

func (a Area) size() float64 { return (a.HighLat - a.LowLat) * (a.HighLng - a.LowLng) }

// AreaFor picks the lead's area: the smallest area whose rectangle contains it (so "Berawa" wins over
// "Badung"), else an area named in its address, else "".
func AreaFor(areas []Area, lat, lng float64, address string) string {
	if lat != 0 || lng != 0 {
		best := -1
		for i, a := range areas {
			if a.contains(lat, lng) && (best < 0 || a.size() < areas[best].size()) {
				best = i
			}
		}
		if best >= 0 {
			return areas[best].Name
		}
	}
	addr := strings.ToLower(address)
	// Prefer longer names first ("Nusa Dua" before "Dua"-like collisions).
	sorted := append([]Area(nil), areas...)
	sort.Slice(sorted, func(i, j int) bool { return len(sorted[i].Name) > len(sorted[j].Name) })
	for _, a := range sorted {
		if a.Name != "" && regexp.MustCompile(`\b`+regexp.QuoteMeta(strings.ToLower(a.Name))+`\b`).MatchString(addr) {
			return a.Name
		}
	}
	return ""
}
