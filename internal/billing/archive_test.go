package billing

import "testing"

func TestEntryNewer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		left, right ArchiveEntry
		want        bool
	}{
		{name: "later issue date", left: ArchiveEntry{IssueDate: "2026-03-07"}, right: ArchiveEntry{IssueDate: "2026-03-06"}, want: true},
		{name: "earlier issue date", left: ArchiveEntry{IssueDate: "2026-03-05"}, right: ArchiveEntry{IssueDate: "2026-03-06"}, want: false},
		{name: "a date beats no date", left: ArchiveEntry{IssueDate: "2026-03-05"}, right: ArchiveEntry{IssueDate: "soon"}, want: true},
		{name: "same date, higher number", left: ArchiveEntry{IssueDate: "2026-03-06", Number: "A-002"}, right: ArchiveEntry{IssueDate: "2026-03-06", Number: "A-001"}, want: true},
		{name: "same number, later filename", left: ArchiveEntry{Filename: "b.yaml"}, right: ArchiveEntry{Filename: "a.yaml"}, want: true},
		{name: "same filename, later path", left: ArchiveEntry{Path: "/y/a.yaml"}, right: ArchiveEntry{Path: "/x/a.yaml"}, want: true},
		{name: "equal", left: ArchiveEntry{}, right: ArchiveEntry{}, want: false},
	}
	for _, tt := range tests {
		if got := tt.left.Newer(tt.right); got != tt.want {
			t.Errorf("%s: Newer = %v, want %v", tt.name, got, tt.want)
		}
	}
}
