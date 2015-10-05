// This is free and unencumbered software released into the public
// domain.  For more information, see <http://unlicense.org> or the
// accompanying UNLICENSE file.
package suggestions

import (
	"sort"
	"strings"

	"github.com/nelsam/gxui"
)

// Adapter is an adapter that is based on gxui's
// FilteredListAdapter and CodeSuggestionAdapter.  There are some
// differences mostly revolving around displaying the suggestions.
type Adapter struct {
	gxui.DefaultAdapter
	suggestions []gxui.CodeSuggestion
	scores      []int
}

func (a *Adapter) SetSuggestions(suggestions []gxui.CodeSuggestion) {
	a.suggestions = suggestions
	a.DefaultAdapter.SetItems(suggestions)
}

func (a *Adapter) Suggestion(item gxui.AdapterItem) gxui.CodeSuggestion {
	return item.(gxui.CodeSuggestion)
}

func (a *Adapter) Sort(partial string) {
	partialLower := strings.ToLower(partial)
	a.scores = make([]int, len(a.suggestions))
	for i, suggestion := range a.suggestions {
		a.scores[i] = score(suggestion.Name(), strings.ToLower(suggestion.Name()), partial, partialLower)
	}
	sort.Sort(a)
	a.DefaultAdapter.SetItems(a.suggestions)
}

func (a *Adapter) Len() int {
	return len(a.suggestions)
}

func (a *Adapter) Less(i, j int) bool {
	return a.scores[i] < a.scores[j]
}

func (a *Adapter) Swap(i, j int) {
	a.suggestions[i], a.suggestions[j] = a.suggestions[j], a.suggestions[i]
	a.scores[i], a.scores[j] = a.scores[j], a.scores[i]
}

func score(suggestion, suggestionLower, partial, partialLower string) int {
	for i := len(partial); i > 0; i-- {
		c := 0
		if strings.Contains(suggestion, partial[:i]) {
			c = i*i + 1
		} else if strings.Contains(suggestionLower, partialLower[:i]) {
			c = i * i
		}
		if c > 0 {
			return c + score(suggestion, suggestionLower, partial[i:], partialLower[i:])
		}
	}
	return 0
}
