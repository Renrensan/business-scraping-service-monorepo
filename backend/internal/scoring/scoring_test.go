package scoring

import (
	"math"
	"testing"
)

func TestCompute_DocWorkedExample(t *testing.T) {
	// "Alaya Resort Ubud: 4,186 reviews, rating 4.7, email + website + FB + TikTok, Group" -> 74, HOT
	l := Lead{
		Name: "Alaya Resort Ubud", ReviewCount: 4186, Rating: 4.7,
		Emails: []string{"info@alayaresorts.com"}, Website: "https://alayaresorts.com",
		FacebookURL: "https://facebook.com/alayaubud", TikTokURL: "https://tiktok.com/@alayaubud",
		GroupType: GroupGroup,
	}
	r := Compute(l, GuestChatProConfig(), "Ubud")
	if math.Abs(r.Breakdown.ReviewPoints-36.2) > 0.05 || math.Abs(r.Breakdown.RatingPoints-7) > 1e-9 || r.Breakdown.ChannelPoints != 16 || r.Breakdown.GroupBonus != 15 {
		t.Errorf("breakdown %+v", r.Breakdown)
	}
	if r.Score != 74 || r.Tier != TierHot {
		t.Errorf("score=%d tier=%s, want 74 hot", r.Score, r.Tier)
	}
}

func TestCompute_DocPointTables(t *testing.T) {
	c := GuestChatProConfig()
	for reviews, want := range map[int]float64{0: 0, 1: 0, 10: 10, 100: 20, 1000: 30, 10000: 40} {
		if got := Compute(Lead{ReviewCount: reviews, Rating: 4}, c, "").Breakdown.ReviewPoints; math.Abs(got-want) > 1e-9 {
			t.Errorf("reviews %d: %v want %v", reviews, got, want)
		}
	}
	for rating, want := range map[float64]float64{4.8: 8, 4.0: 0, 3.6: -4} {
		if got := Compute(Lead{Rating: rating}, c, "").Breakdown.RatingPoints; math.Abs(got-want) > 1e-9 {
			t.Errorf("rating %v: %v want %v", rating, got, want)
		}
	}
	// No rating is neutral, not -40.
	if r := Compute(Lead{}, c, ""); r.Breakdown.RatingPoints != 0 || !r.Breakdown.NoRating {
		t.Errorf("unrated: %+v", r.Breakdown)
	}
}

func TestCompute_ChannelsCappedAndPhoneNotCounted(t *testing.T) {
	c := GuestChatProConfig()
	c.Channels = append(c.Channels, ChannelLinkedIn) // 6 channels possible
	l := Lead{
		Emails: []string{"hello@villa.com"}, Website: "villa.com", InstagramURL: "https://instagram.com/villa",
		FacebookURL: "https://facebook.com/villa", TikTokURL: "https://tiktok.com/@villa", LinkedInURL: "https://linkedin.com/company/villa",
		Phone: "+62 811",
	}
	r := Compute(l, c, "")
	if r.Breakdown.ChannelPoints != 20 || len(r.Breakdown.Channels) != 6 {
		t.Errorf("channels %v points %v", r.Breakdown.Channels, r.Breakdown.ChannelPoints)
	}
	for _, ch := range Compute(l, GuestChatProConfig(), "").Breakdown.Channels {
		if ch == ChannelPhone {
			t.Error("phone must not count for GuestChatPro")
		}
	}
}

func TestCompute_TiersAndOverrides(t *testing.T) {
	c := GuestChatProConfig()
	big := Lead{ReviewCount: 11000, Rating: 4.5, Website: "swiss-belhotel.com", Emails: []string{"a@swiss-belhotel.com"}}

	// Swiss-Belexpress Kuta: 11,000 reviews but a chain -> LOW whatever the score.
	chain := big
	chain.GroupType = GroupChain
	if r := Compute(chain, c, "Kuta"); r.Tier != TierLow || r.Breakdown.TierReason == "" {
		t.Errorf("chain: %s", r.Tier)
	}
	skip := big
	skip.GroupType = GroupSkip
	if r := Compute(skip, c, "Kuta"); r.Tier != TierSkip {
		t.Errorf("skip: %s", r.Tier)
	}

	// Same score, different areas: 50 is HOT in Canggu (50/40) but GOOD in Kuta (62/45).
	l := Lead{ReviewCount: 1000, Rating: 4.5, Website: "villa.com", Emails: []string{"a@villa.com"}, InstagramURL: "https://instagram.com/villa"} // 30+5+12 = 47
	l.GroupType = GroupSmallGroup                                                                                                                 // +8 = 55
	if r := Compute(l, c, "Canggu"); r.Score != 55 || r.Tier != TierHot {
		t.Errorf("canggu %d %s", r.Score, r.Tier)
	}
	if r := Compute(l, c, "Kuta"); r.Tier != TierGood {
		t.Errorf("kuta %s", r.Tier)
	}
	if r := Compute(l, c, "Somewhere New"); r.Breakdown.Cutoff != (Cutoff{50, 38}) || r.Tier != TierHot {
		t.Errorf("default cutoff %+v %s", r.Breakdown.Cutoff, r.Tier)
	}
	if r := Compute(Lead{ReviewCount: 10, Rating: 4}, c, "nusa dua"); r.Tier != TierLow || r.Breakdown.Cutoff.Hot != 47 {
		t.Errorf("case-insensitive area: %+v", r.Breakdown.Cutoff)
	}
}

func TestJunkContacts(t *testing.T) {
	emails := map[string]bool{ // true = junk
		"info@villa-bali.com": false, "reservations@alayaresorts.com": false, "someone@gmail.com": false,
		"605a7baede844d278b89dc95ae0a9123@sentry-next.wixpress.com": true, "john@doe.com": true, "info@company.com": true,
		"yourmail@mail.com": true, "host@airbnb.com": true, "logo@2x.png": true, "noreply@booking.com": true,
	}
	for e, junk := range emails {
		if got := JunkEmailReason(e) != ""; got != junk {
			t.Errorf("email %q junk=%v want %v (%s)", e, got, junk, JunkEmailReason(e))
		}
	}
	sites := map[string]bool{
		"https://www.alayaresorts.com": false, "villa-bali.com/en": false,
		"https://www.booking.com/hotel/id/x.html": true, "https://airbnb.co.id/rooms/1": true, "https://linktr.ee/villa": true,
		"https://www.tripadvisor.com/Hotel_Review": true, "https://instagram.com/villa": true, "https://wa.me/62811": true,
	}
	for s, junk := range sites {
		if got := JunkWebsiteReason(s) != ""; got != junk {
			t.Errorf("site %q junk=%v want %v", s, got, junk)
		}
	}
	socials := map[string]bool{
		"https://www.facebook.com/alayaubud": false, "https://instagram.com/alaya.ubud/": false, "https://www.tiktok.com/@alaya": false,
		"https://www.facebook.com/tr?id=123": true, "https://www.facebook.com/profile.php?id=1": true, "https://instagram.com/p/Cx12/": true,
		"https://www.instagram.com/airbnb": true, "https://facebook.com/oyo.indonesia": true, "https://www.facebook.com/": true,
		"https://www.facebook.com/sharer/sharer.php?u=x": true,
	}
	for s, junk := range socials {
		if got := JunkSocialReason(s) != ""; got != junk {
			t.Errorf("social %q junk=%v want %v (%s)", s, got, junk, JunkSocialReason(s))
		}
	}
}

func TestCountChannels_JunkIgnoredAndReported(t *testing.T) {
	l := Lead{
		Emails:       []string{"x@sentry.wixpress.com", "info@villa.com"},
		Website:      "https://booking.com/villa",
		InstagramURL: "https://instagram.com/p/abc/",
		FacebookURL:  "https://facebook.com/villabali",
	}
	counted, ignored := CountChannels(l, GuestChatProConfig().Channels, true)
	if len(counted) != 2 || counted[0] != ChannelEmail || counted[1] != ChannelFacebook {
		t.Errorf("counted %v", counted)
	}
	if len(ignored) != 3 {
		t.Errorf("ignored %+v", ignored)
	}
	counted, _ = CountChannels(l, GuestChatProConfig().Channels, false)
	if len(counted) != 4 {
		t.Errorf("without cleaning counted %v", counted)
	}
}

func TestSuggestGroup(t *testing.T) {
	cases := map[string][2]string{
		"Alaya Resort Ubud":                         {GroupGroup, "Alaya"},
		"Adiwana Monkey Forest":                     {GroupGroup, "Adiwana"},
		"The Jimbaran Villa by iNi Vie Hospitality": {GroupGroup, "iNi ViE"},
		"Maya Ubud Resort & Spa":                    {GroupGroup, "Maya"},
		"Villa Maya Canggu":                         {GroupIndependent, ""},
		"Swiss-Belexpress Kuta Legian":              {GroupChain, "Swiss-Belhotel"},
		"Courtyard by Marriott Bali Seminyak":       {GroupChain, "Marriott"},
		"FOX Hotel Jimbaran Beach":                  {GroupIndependent, ""},
		"Harper Kuta":                               {GroupChain, "Archipelago"},
		"favehotel Kuta Square":                     {GroupChain, "Archipelago"},
		"OYO 123 Griya Bali":                        {GroupSkip, "OYO"},
		"Finns Beach Club":                          {GroupSkip, ""},
		"Kos Putri Melati":                          {GroupSkip, ""},
		"Wasabi Hotel & Restaurant":                 {GroupRestaurantStay, ""},
		"The Luxe Nomad Villa":                      {GroupSmallGroup, "The Luxe Nomad"},
		"Mövenpick Resort & Spa Jimbaran Bali":      {GroupChain, "Accor"},
	}
	for name, want := range cases {
		typ, grp := SuggestGroup(name, "Hotel")
		if typ != want[0] || grp != want[1] {
			t.Errorf("%q -> %s/%q, want %s/%q", name, typ, grp, want[0], want[1])
		}
	}
	if typ, _ := SuggestGroup("Sunset Corner", "Restaurant"); typ != GroupRestaurantStay {
		t.Errorf("restaurant category -> %s", typ)
	}
}

func TestAppliesAndAreaFor(t *testing.T) {
	if !Applies([]string{"Lodging"}, Lead{Types: []string{"Point Of Interest", "lodging"}}) || Applies([]string{"Lodging"}, Lead{Category: "Spa", Types: []string{"Spa"}}) || !Applies(nil, Lead{}) {
		t.Error("Applies")
	}
	areas := []Area{
		{"Badung", -8.71, 115.05, -8.45, 115.31},
		{"Berawa", -8.676, 115.126, -8.648, 115.154},
		{"Canggu", -8.67, 115.116, -8.625, 115.161},
		{"Nusa Dua", -8.83, 115.20, -8.77, 115.26},
	}
	if a := AreaFor(areas, -8.662, 115.140, ""); a != "Berawa" {
		t.Errorf("smallest containing area: %s", a)
	}
	if a := AreaFor(areas, -8.50, 115.10, ""); a != "Badung" {
		t.Errorf("only containing area: %s", a)
	}
	if a := AreaFor(areas, 0, 0, "Jl. Pratama, Nusa Dua, Kabupaten Badung"); a != "Nusa Dua" {
		t.Errorf("address fallback: %s", a)
	}
	if a := AreaFor(areas, -7.0, 110.0, "Yogyakarta"); a != "" {
		t.Errorf("no area: %s", a)
	}
}
