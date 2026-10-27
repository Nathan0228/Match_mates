package main

import (
	"strings"
)

// MatchPreferences contains optional, self-reported prompts. MBTI and the two
// short texts are displayed to people but do not influence the score.
type MatchPreferences struct {
	Values   []string `json:"values"`
	Pace     string   `json:"pace"`
	Contact  string   `json:"contact"`
	Energy   int      `json:"energy"`
	Rhythm   string   `json:"rhythm"`
	MBTI     string   `json:"mbti"`
	Daily    string   `json:"daily"`
	Together string   `json:"together"`
}

var valueOptions = map[string]bool{
	"真诚沟通": true, "尊重边界": true, "共同成长": true, "稳定陪伴": true,
	"独立空间": true, "热爱生活": true, "互相支持": true, "未来规划": true,
}

func validPreferences(p MatchPreferences) bool {
	if len(p.Values) > 3 {
		return false
	}
	seen := map[string]bool{}
	for _, v := range p.Values {
		if !valueOptions[v] || seen[v] {
			return false
		}
		seen[v] = true
	}
	if !oneOf(p.Pace, "", "slow", "balanced", "soon") ||
		!oneOf(p.Contact, "", "frequent", "deep", "flexible") ||
		!oneOf(p.Rhythm, "", "early", "flexible", "late") ||
		p.Energy < 0 || p.Energy > 5 ||
		len([]rune(p.Daily)) > 120 || len([]rune(p.Together)) > 120 {
		return false
	}
	if p.MBTI == "" {
		return true
	}
	if len(p.MBTI) != 4 {
		return false
	}
	return oneOf(p.MBTI[:1], "E", "I") && oneOf(p.MBTI[1:2], "S", "N") &&
		oneOf(p.MBTI[2:3], "T", "F") && oneOf(p.MBTI[3:4], "J", "P")
}

func oneOf(value string, options ...string) bool {
	for _, option := range options {
		if value == option {
			return true
		}
	}
	return false
}

func similarity(u User, c Candidate) float64 {
	var score float64
	add := func(weight, value float64, observed bool) {
		if observed {
			score += weight * value
		} else {
			score += weight * 0.5
		}
	}
	add(0.30, overlap(u.Preferences.Values, c.Preferences.Values), len(u.Preferences.Values) > 0 && len(c.Preferences.Values) > 0)
	add(0.28, overlap(u.Hobbies, c.Hobbies), len(u.Hobbies) > 0 && len(c.Hobbies) > 0)
	add(0.16, paceScore(u.Preferences.Pace, c.Preferences.Pace), u.Preferences.Pace != "" && c.Preferences.Pace != "")
	add(0.11, sameOr(u.Preferences.Contact, c.Preferences.Contact, 0.6), u.Preferences.Contact != "" && c.Preferences.Contact != "")
	add(0.08, 1-float64(abs(u.Preferences.Energy-c.Preferences.Energy))/4, u.Preferences.Energy > 0 && c.Preferences.Energy > 0)
	add(0.05, rhythmScore(u.Preferences.Rhythm, c.Preferences.Rhythm), u.Preferences.Rhythm != "" && c.Preferences.Rhythm != "")
	add(0.02, sameOr(strings.TrimSpace(strings.ToLower(u.Campus)), strings.TrimSpace(strings.ToLower(c.Campus)), 0.5), u.Campus != "" && c.Campus != "")
	return score
}

func overlap(a, b []string) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	x, y := map[string]bool{}, map[string]bool{}
	for _, item := range a {
		if key := strings.TrimSpace(strings.ToLower(item)); key != "" {
			x[key] = true
		}
	}
	for _, item := range b {
		if key := strings.TrimSpace(strings.ToLower(item)); key != "" {
			y[key] = true
		}
	}
	if len(x) == 0 || len(y) == 0 {
		return 0
	}
	shared := 0
	for key := range x {
		if y[key] {
			shared++
		}
	}
	firstTwo := float64(shared) / 2
	if firstTwo > 1 {
		firstTwo = 1
	}
	return 0.65*firstTwo + 0.35*(2*float64(shared)/float64(len(x)+len(y)))
}

func paceScore(a, b string) float64 {
	if a == b {
		return 1
	}
	if (a == "slow" && b == "soon") || (a == "soon" && b == "slow") {
		return 0.3
	}
	return 0.7
}

func rhythmScore(a, b string) float64 {
	if a == b {
		return 1
	}
	if a == "flexible" || b == "flexible" {
		return 0.75
	}
	return 0.25
}

func sameOr(a, b string, otherwise float64) float64 {
	if a == b {
		return 1
	}
	return otherwise
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func commonItems(a, b []string) []string {
	lookup := map[string]bool{}
	for _, item := range a {
		lookup[strings.ToLower(strings.TrimSpace(item))] = true
	}
	result := []string{}
	for _, item := range b {
		if lookup[strings.ToLower(strings.TrimSpace(item))] {
			result = append(result, item)
		}
	}
	return result
}

func matchReasons(u User, c Candidate) []string {
	reasons := []string{}
	if common := commonItems(u.Preferences.Values, c.Preferences.Values); len(common) > 0 {
		reasons = append(reasons, "都看重"+common[0])
	}
	if common := commonItems(u.Hobbies, c.Hobbies); len(common) > 0 {
		reasons = append(reasons, "都喜欢"+common[0])
	}
	if u.Preferences.Pace != "" && u.Preferences.Pace == c.Preferences.Pace {
		reasons = append(reasons, "认识节奏相近")
	}
	return reasons
}
