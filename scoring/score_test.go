// This is free and unencumbered software released into the public
// domain.  For more information, see <http://unlicense.org> or the
// accompanying UNLICENSE file.

package scoring_test

import (
	"testing"

	"github.com/a8m/expect"
	"github.com/nelsam/vidar/scoring"
)

func TestSort(t *testing.T) {
	expect := expect.New(t)

	v := []string{
		"SomeLongThing",
		"Something",
		"Thing",
		"thing",
		"thisIsBad",
		"thingimajigger",
		"thingy",
		"thisIsNotAGoodMatch",
		"Thang",
		"tang",
		"bacon",
		"eggs",
	}
	expect(scoring.Sort(v, "thing")).To.Equal([]string{
		"thing",
		"Thing",
		"thingy",
		"thingimajigger",
		"Thang",
		"tang",
		"thisIsBad",
		"thisIsNotAGoodMatch",
		"Something",
		"SomeLongThing",
	})
}
