package api

import "testing"

func TestV2StartTimeSoComHoraInformada(t *testing.T) {
	cases := []struct {
		entry v2TimeEntry
		want  string
	}{
		{v2TimeEntry{Date: "2026-06-01T09:30:00Z", HasStartTime: true}, "09:30"},
		{v2TimeEntry{Date: "2026-06-01T14:05:00-03:00", HasStartTime: true}, "14:05"},
		{v2TimeEntry{Date: "2026-06-01T09:30:00Z", HasStartTime: false}, ""},
		{v2TimeEntry{Date: "2026-06-01", HasStartTime: true}, ""},
		{v2TimeEntry{Date: "lixo", HasStartTime: true}, ""},
	}
	for _, c := range cases {
		if got := v2StartTime(c.entry); got != c.want {
			t.Errorf("v2StartTime(%+v) = %q, esperava %q", c.entry, got, c.want)
		}
	}
}
