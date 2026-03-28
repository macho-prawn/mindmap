package provider

import (
	"testing"
	"time"

	compute "google.golang.org/api/compute/v1"
)

func TestSelectActiveMacsecKeyName(t *testing.T) {
	now := time.Date(2026, time.March, 28, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name   string
		macsec *compute.InterconnectMacsec
		want   string
	}{
		{
			name: "nil macsec",
			want: "",
		},
		{
			name: "single key without start time",
			macsec: &compute.InterconnectMacsec{
				PreSharedKeys: []*compute.InterconnectMacsecPreSharedKey{
					{Name: "key-a"},
				},
			},
			want: "key-a",
		},
		{
			name: "active key uses latest eligible start time",
			macsec: &compute.InterconnectMacsec{
				PreSharedKeys: []*compute.InterconnectMacsecPreSharedKey{
					{Name: "key-old", StartTime: "2026-03-27T12:00:00Z"},
					{Name: "key-active", StartTime: "2026-03-28T11:00:00Z"},
					{Name: "key-future", StartTime: "2026-03-28T13:00:00Z"},
				},
			},
			want: "key-active",
		},
		{
			name: "all future keys fall back to newest configured key",
			macsec: &compute.InterconnectMacsec{
				PreSharedKeys: []*compute.InterconnectMacsecPreSharedKey{
					{Name: "key-soon", StartTime: "2026-03-28T13:00:00Z"},
					{Name: "key-latest", StartTime: "2026-03-28T15:00:00Z"},
				},
			},
			want: "key-latest",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := selectActiveMacsecKeyName(now, tc.macsec); got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

func TestFormatASN(t *testing.T) {
	if got := formatASN(nil); got != "" {
		t.Fatalf("expected empty ASN for nil bgp, got %q", got)
	}
	if got := formatASN(&compute.RouterBgp{Asn: 64512}); got != "64512" {
		t.Fatalf("expected 64512, got %q", got)
	}
}
