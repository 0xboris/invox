package validate

import "github.com/0xboris/invox/internal/invoice"

// validationJSON is the --json output of validate. The invoice's fields are
// null when it is invalid.
type validationJSON struct {
	Valid      bool          `json:"valid"`
	Number     *string       `json:"number"`
	CustomerID *string       `json:"customerId"`
	LineItems  *int          `json:"lineItems"`
	Total      *string       `json:"total"`
	Currency   *string       `json:"currency"`
	Errors     []problemJSON `json:"errors"`
}

// problemJSON is one problem with the invoice. File, Line and Field are null
// when the problem does not name them.
type problemJSON struct {
	File    *string `json:"file"`
	Line    *int    `json:"line"`
	Field   *string `json:"field"`
	Message string  `json:"message"`
}

// invalidInvoiceProblems returns the problems in err, an error from
// invoice.LoadContext, and reports whether err is only problems with the
// invoice's content. It reports false for any other error, such as a file
// that could not be read.
func invalidInvoiceProblems(err error) ([]problemJSON, bool) {
	var problems []problemJSON
	var walk func(error) bool
	walk = func(err error) bool {
		switch e := err.(type) {
		case *invoice.DecodeError:
			field := e.Path
			if field == "" {
				field = e.Field
			}
			problems = append(problems, problemJSON{File: &e.File, Line: &e.Line, Field: optional(field), Message: e.Problem})
		case *invoice.UnknownCustomerError:
			problems = append(problems, problemJSON{File: &e.Path, Field: optional("customer_id"), Message: "unknown customer_id `" + e.CustomerID + "`"})
		case *invoice.ValidationError:
			for _, problem := range e.Problems {
				problems = append(problems, problemJSON{File: optional(problem.File), Field: optional(problem.Field), Message: problem.Message})
			}
		case interface{ Unwrap() []error }:
			for _, inner := range e.Unwrap() {
				if !walk(inner) {
					return false
				}
			}
		default:
			return false
		}
		return true
	}
	if !walk(err) || len(problems) == 0 {
		return nil, false
	}
	return problems, true
}

func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
