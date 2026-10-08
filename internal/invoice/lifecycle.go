package invoice

// Status is invoice.status: where an invoice is in its life. An invoice
// file may hold any text there; the statuses below are the ones invox sets
// and checks.
type Status string

const (
	Draft    Status = "draft"    // made by `new`
	Built    Status = "built"    // its PDF was built
	Editing  Status = "editing"  // a working copy made by `archive edit`
	Archived Status = "archived" // in the archive
)

// Action is a step that a status allows or refuses.
type Action int

const (
	// Building compiles the PDF.
	Building Action = iota
	// Archiving moves a new invoice into the archive.
	Archiving
	// Rearchiving moves a working copy from `archive edit` back.
	Rearchiving
	// Emailing drafts an email with the PDF.
	Emailing
	// Numbering gives an unarchived invoice a number that later ones skip.
	Numbering
)

// anyStatus is the transitions key for every status not listed.
const anyStatus Status = "\x00any"

// transitions lists, for each action, the statuses it starts from and the
// status it leaves.
var transitions = map[Action]map[Status]Status{
	Building:    {Archived: Archived, anyStatus: Built},
	Archiving:   {Built: Archived},
	Rearchiving: {Editing: Archived, Built: Archived},
	Emailing:    {Built: Built, Archived: Archived},
	Numbering:   {Draft: Draft, Built: Built},
}

// Apply returns the status a leaves an invoice in, and false when s does
// not allow a.
func (s Status) Apply(a Action) (Status, bool) {
	table := transitions[a]
	if next, ok := table[s]; ok && s != anyStatus {
		return next, true
	}
	next, ok := table[anyStatus]
	return next, ok
}

// Allows reports whether s allows a.
func (s Status) Allows(a Action) bool {
	_, ok := s.Apply(a)
	return ok
}
