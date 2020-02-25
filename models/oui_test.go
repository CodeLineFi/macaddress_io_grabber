package models

import (
	"log"
	"reflect"
	"testing"
)

func init() {
	log.SetFlags(log.Flags() | log.Lshortfile)
}

func TestRegistry_Unique(t *testing.T) {

	r1 := &OUI{Assignment: "00", OrgName: "n1", OrgAddress: "a1"}
	r2 := &OUI{Assignment: "00", OrgName: "n2", OrgAddress: "a2"}
	r3 := &OUI{Assignment: "30", OrgName: "n3", OrgAddress: "a3"}

	var test = func(s string, f func(old, new *OUI) bool, target Registry) {
		var r Registry = []*OUI{r1, r2, r3}
		r.Unique(f)
		if !reflect.DeepEqual(r, target) {
			log.Println(s, r, target)
			t.FailNow()
		}
	}

	test("nil",
		nil,
		Registry([]*OUI{r1, r3}),
	)
	test("false",
		func(old, new *OUI) bool {
			return false
		},
		Registry([]*OUI{r1, r3}),
	)
	test("true",
		func(old, new *OUI) bool {
			return true
		},
		Registry([]*OUI{r2, r3}),
	)
}

func TestRegistry_Swap(t *testing.T) {

	r1 := &OUI{Assignment: "00", OrgName: "n1", OrgAddress: "a1"}
	r2 := &OUI{Assignment: "00", OrgName: "n2", OrgAddress: "a2"}
	r3 := &OUI{Assignment: "30", OrgName: "n3", OrgAddress: "a3"}

	var test = func(s string, i, k int, target Registry) {
		var r Registry = []*OUI{r1, r2, r3}

		r.Swap(i, k)
		if !reflect.DeepEqual(r, target) {
			log.Println(s, "Couldn't swap values", i, k)
			t.FailNow()
		}
	}

	test("test_1", 0, 1, []*OUI{r2, r1, r3})
	test("test_2", 1, 0, []*OUI{r2, r1, r3})
	test("test_2", 0, 2, []*OUI{r3, r2, r1})
}
