package parse

import (
	"log"
	. "macaddress_io_grabber/models"
	"reflect"
	"testing"
)

func TestMerge(t *testing.T) {

	r11 := &OUI{Assignment: "00", OrgName: "r100", OrgAddress: "r100"}
	r12 := &OUI{Assignment: "20", OrgName: "r120", OrgAddress: "r120"}
	r13 := &OUI{Assignment: "30", OrgName: "r130", OrgAddress: "r130"}
	var r1 Registry = []*OUI{r11, r12, r13}

	r21 := &OUI{Assignment: "12", OrgName: "r212", OrgAddress: "r212"}
	r22 := &OUI{Assignment: "23", OrgName: "r223", OrgAddress: "r223"}
	r23 := &OUI{Assignment: "24", OrgName: "r224", OrgAddress: "r224"}
	var r2 Registry = []*OUI{r21, r22, r23}

	r := merge(r1, r2)
	target := Registry([]*OUI{r11, r21, r12, r22, r23, r13})

	if !reflect.DeepEqual(r, target) {
		t.FailNow()
	}

	r31 := &OUI{Assignment: "00", OrgName: "r312", OrgAddress: "r300"}
	r32 := &OUI{Assignment: "12", OrgName: "r323", OrgAddress: "r312"}
	r33 := &OUI{Assignment: "23", OrgName: "r324", OrgAddress: "r323"}
	var r3 Registry = []*OUI{r31, r32, r33}

	r = merge(r3, r2, r1)
	target = Registry([]*OUI{r31, r11, r32, r21, r12, r33, r22, r23, r13})

	if !reflect.DeepEqual(r, target) {
		log.Println(r)
		log.Println(target)
		t.FailNow()
	}

	r = merge(r1, r2, r3)
	target = Registry([]*OUI{r11, r31, r21, r32, r12, r22, r33, r23, r13})

	if !reflect.DeepEqual(r, target) {
		log.Println(r)
		log.Println(target)
		t.FailNow()
	}
}
