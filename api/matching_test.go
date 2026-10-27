package main

import (
	"strings"
	"testing"
)

func TestPreferenceValidation(t *testing.T) {
	valid := MatchPreferences{Values: []string{"真诚沟通", "尊重边界"}, Pace: "balanced", Contact: "deep", Energy: 3, Rhythm: "flexible", MBTI: "INFP", Daily: "散步", Together: "看展"}
	if !validPreferences(valid) {
		t.Fatal("valid preferences rejected")
	}
	for _, change := range []func(*MatchPreferences){
		func(p *MatchPreferences) { p.Values = []string{"真诚沟通", "真诚沟通"} },
		func(p *MatchPreferences) { p.Values = []string{"未知标签"} },
		func(p *MatchPreferences) { p.Pace = "invalid" },
		func(p *MatchPreferences) { p.Energy = 6 },
		func(p *MatchPreferences) { p.MBTI = "ABCD" },
		func(p *MatchPreferences) { p.Daily = strings.Repeat("长", 121) },
	} {
		p := valid
		change(&p)
		if validPreferences(p) {
			t.Fatalf("invalid preferences accepted: %+v", p)
		}
	}
}
