package api

import "testing"

func TestPatternClient(t *testing.T) {
	c := NewPatternClient(NewClient())

	t.Run("find", func(t *testing.T) {
		const perPage = 30

		for i := 1; i <= 2; i++ {
			patterns, err := c.Find(i, 10, FindRecommendedPatterns)
			if err != nil {
				t.Fatal("cannot get patterns:", err)
			}

			t.Log("page", i)
			testLogPatterns(t, patterns)
		}
	})

	t.Run("search_title", func(t *testing.T) {
		patterns, err := c.SearchTitle("prostate")
		if err != nil {
			t.Fatal("cannot search patterns:", err)
		}

		testLogPatterns(t, patterns)
	})
}

func testLogPatterns(t *testing.T, patterns []Pattern) {
	for i, pattern := range patterns {
		t.Logf(
			"%02d: %s (%q, v%d) by %s",
			i+1, pattern.DecodedName(), pattern.ToyTag, pattern.Version2, pattern.AuthorOrAnon(),
		)
	}
}
