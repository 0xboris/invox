package invoice

import "testing"

func TestStatusTransitions(t *testing.T) {
	tests := []struct {
		status Status
		action Action
		want   Status
		ok     bool
	}{
		{Draft, Building, Built, true},
		{Editing, Building, Built, true},
		{"", Building, Built, true},
		{"sent", Building, Built, true},
		{Archived, Building, Archived, true},
		{Built, Archiving, Archived, true},
		{Draft, Archiving, "", false},
		{Editing, Archiving, "", false},
		{Editing, Rearchiving, Archived, true},
		{Built, Rearchiving, Archived, true},
		{Draft, Rearchiving, "", false},
		{Built, Emailing, Built, true},
		{Archived, Emailing, Archived, true},
		{Draft, Emailing, "", false},
		{"", Emailing, "", false},
		{Draft, Numbering, Draft, true},
		{Built, Numbering, Built, true},
		{Archived, Numbering, "", false},
		{Editing, Numbering, "", false},
	}
	for _, tc := range tests {
		got, ok := tc.status.Apply(tc.action)
		if got != tc.want || ok != tc.ok {
			t.Errorf("Status(%q).Apply(%d) = %q, %v; want %q, %v", tc.status, tc.action, got, ok, tc.want, tc.ok)
		}
		if tc.status.Allows(tc.action) != tc.ok {
			t.Errorf("Status(%q).Allows(%d) = %v, want %v", tc.status, tc.action, !tc.ok, tc.ok)
		}
	}
}
