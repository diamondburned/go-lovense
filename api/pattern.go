package api

import (
	"encoding/base64"
	"net/url"
	"strconv"
	"strings"

	"github.com/diamondburned/go-lovense/pattern"
)

// PatternClient handles pattern-fetching routes.
type PatternClient struct {
	*Client
}

// NewPatternClient returns a new PatternClient from the given Client.
func NewPatternClient(c *Client) *PatternClient {
	return &PatternClient{c}
}

// Pattern describes a pattern.
type Pattern struct {
	Author         string      `json:"author"`
	CDNPath        string      `json:"cdnPath"`
	Created        string      `json:"created"`     // YYYY/MM/DD HH:MM
	CreatedTime    int64       `json:"createdTime"` // UnixMilli
	Duration       int64       `json:"duration"`
	Email          string      `json:"email"`
	Favorite       interface{} `json:"favorite"` // null(?)
	FavoritesCount int64       `json:"favoritesCount"`
	ID             string      `json:"id"`
	IsAnony        string      `json:"isAnony"` // bool, "1"
	IsShowReview   string      `json:"isShowReview"`
	LikeCount      int64       `json:"likeCount"`
	Name           string      `json:"name"`
	Path           string      `json:"path"`
	PlayCount      int64       `json:"playCount"`
	RandomCode     string      `json:"randomCode"`
	Self           bool        `json:"self"`
	Status         string      `json:"status"`
	Text           string      `json:"text"`
	Timer          string      `json:"timer"`
	ToyTag         string      `json:"toyTag"`
	Updated        string      `json:"updated"` // YYYY/MM/DD HH:MM
	Version        string      `json:"version"`
	Version2       int64       `json:"version2"`
}

// DecodedName returns the Pattern's name decoded from base64 if possible.
func (p *Pattern) DecodedName() string {
	b, err := base64.StdEncoding.DecodeString(p.Name)
	if err != nil {
		return p.Name
	}
	return string(b)
}

// AuthorOrAnon returns the Author name or Anonymous if empty.
func (p *Pattern) AuthorOrAnon() string {
	if p.Author != "" {
		return p.Author
	}
	return "Anonymous"
}

// Features reads p.ToyTag and parses them into a list of features.
func (p *Pattern) Features() []pattern.Feature {
	t := strings.Split(p.ToyTag, ",")
	f := make([]pattern.Feature, len(t))
	for i, t := range t {
		f[i] = pattern.Feature(t)
	}
	return f
}

// PatternFindType
type PatternFindType string

const (
	FindRecommendedPatterns PatternFindType = "Recommended"
	FindPopularPatterns     PatternFindType = "Popular"
	FindRecentPatterns      PatternFindType = "recent"
	FindPickPatterns        PatternFindType = "pick"
)

// Find calls the /find endpoint, which lists patterns according to the given
// parameters.
//
// typ should typically be "Recommended".
// If pageSize is 0, then 15 is used by default.
// If page is 0, then 1 is used for the first page.
// There is currently no known page/pageSize.
func (c *PatternClient) Find(page, pageSize int, typ PatternFindType) ([]Pattern, error) {
	var patterns []Pattern

	if page == 0 {
		page = 1
	}

	if pageSize == 0 {
		pageSize = 15
	}

	res := ResponseBody{Data: &patterns}
	err := c.DoPOST("/wear/pattern/v2/find", &res, WithPOSTForm(url.Values{
		"pageSize": {strconv.Itoa(pageSize)},
		"page":     {strconv.Itoa(page)},
		"type":     {string(typ)},
	}))

	return patterns, err
}

// SearchTitle searches for patterns with the given keyword in its title.
func (c *PatternClient) SearchTitle(keyword string) ([]Pattern, error) {
	var patterns []Pattern

	res := ResponseBody{Data: &patterns}
	err := c.DoPOST("/wear/pattern/search_title", &res, WithPOSTForm(url.Values{
		"keyword": {string(keyword)},
	}))

	return patterns, err
}

// SearchAuthor searches for patterns with the given keyword in its author field.
func (c *PatternClient) SearchAuthor(keyword string) ([]Pattern, error) {
	var patterns []Pattern

	res := ResponseBody{Data: &patterns}
	err := c.DoPOST("/wear/pattern/search_author", &res, WithPOSTForm(url.Values{
		"keyword": {string(keyword)},
	}))

	return patterns, err
}

// DownloadPattern downloads the given pattern from the CDN and parses it into
// the pattern data.
func (c *PatternClient) DownloadPattern(p *Pattern) (*pattern.Pattern, error) {
	r, err := c.Do("GET", p.CDNPath)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	return pattern.Parse(r.Body)
}
