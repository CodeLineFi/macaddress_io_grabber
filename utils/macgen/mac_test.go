package macgen

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewMac(t *testing.T) {

	tests := []struct {
		text    string
		success bool
		macLen  int
	}{
		{"", true, 0},
		{"0123456789AB", true, 6},
		{"0123456789abcdef", true, 8},
		{strings.Repeat("0123456789AB", 100), true, 600},
		{"a", true, 1},
		{"0101", true, 2},
		{"tt", false, 0},
	}

	for _, test := range tests {
		m, err := newMAC(test.text)
		if !test.success {
			if err == nil {
				t.Log("error has been expected:", test.text)
				t.FailNow()
			}
			continue
		}
		if err != nil {
			t.Log(err, test.text)
			t.FailNow()
		}
		if len(m) != test.macLen {
			t.Log("wrong length:", test.text)
			t.FailNow()
		}
	}
}

func TestMACSetPrefix(t *testing.T) {

	text := "FF0f9bc12345"

	tests := []struct {
		prefix  string
		success bool
		res     string
	}{
		{"", true, "fF0f9bc12345"},
		{"a", true, "AF0f9bc12345"},
		{"111111", true, "111111c12345"},
		{"987654321987654321", true, "987654321987"},
		{"asd", false, ""},
	}

	for _, test := range tests {

		m, err := newMAC(text)
		if err != nil {
			t.Log(err, text)
			t.FailNow()
		}

		err = m.setPrefix(test.prefix)
		if !test.success {
			if err == nil {
				t.Log("error has been expected:", test.prefix)
				t.FailNow()
			}
			continue
		}

		if err != nil {
			t.Log(err, test.prefix)
			t.FailNow()
		}

		r, err := newMAC(test.res)
		if err != nil {
			t.Log(err, test.res)
			t.FailNow()
		}
		if bytes.Compare(m, r) != 0 {
			t.Log("unexpected result")
			t.Log(m)
			t.Log(r)
			t.FailNow()
		}
	}
}

func TestMACSetTransType(t *testing.T) {

	m, err := newMAC("FE0f9bc12345")
	if err != nil {
		t.Log(err, m)
		t.FailNow()
	}

	r1, err := newMAC("FF0f9bc12345")
	if err != nil {
		t.Log(err, m)
		t.FailNow()
	}

	r2, err := newMAC("FE0f9bc12345")
	if err != nil {
		t.Log(err, m)
		t.FailNow()
	}

	m.setTransType(true)

	if bytes.Compare(m, r1) != 0 {
		t.Log("unexpected result")
		t.Log(m)
		t.Log(r1)
		t.FailNow()
	}

	m.setTransType(false)

	if bytes.Compare(m, r2) != 0 {
		t.Log("unexpected result")
		t.Log(m)
		t.Log(r2)
		t.FailNow()
	}
}

func TestMACSetAdminType(t *testing.T) {

	m, err := newMAC("Fc0f9bc12345")
	if err != nil {
		t.Log(err, m)
		t.FailNow()
	}

	r1, err := newMAC("FE0f9bc12345")
	if err != nil {
		t.Log(err, m)
		t.FailNow()
	}

	r2, err := newMAC("FC0f9bc12345")
	if err != nil {
		t.Log(err, m)
		t.FailNow()
	}

	m.setAdminType(true)

	if bytes.Compare(m, r1) != 0 {
		t.Log("unexpected result")
		t.Log(m)
		t.Log(r1)
		t.FailNow()
	}

	m.setAdminType(false)

	if bytes.Compare(m, r2) != 0 {
		t.Log("unexpected result")
		t.Log(m)
		t.Log(r2)
		t.FailNow()
	}
}

func TestMACFormat(t *testing.T) {

	m, err := newMAC("Fc0f9bc1234")
	if err != nil {
		t.Log(err, m)
		t.FailNow()
	}

	m.format(':', 2, true)

	tests := []struct {
		sep       byte
		chunk     uint
		upperCase bool
		res       string
	}{
		{':', 2, true, "FC:0F:9B:C1:23:40"},
		{'x', 2, true, "FCx0Fx9BxC1x23x40"},
		{'x', 2, false, "fcx0fx9bxc1x23x40"},
		{'x', 1, false, "fxcx0xfx9xbxcx1x2x3x4x0"},
		{'[', 1, false, "f[c[0[f[9[b[c[1[2[3[4[0"},
		{'[', 1, true, "F[C[0[F[9[B[C[1[2[3[4[0"},
		{'[', 3, true, "FC0[F9B[C12[340"},
		{'!', 4, true, "FC0F!9BC1!2340"},
		{'!', 5, true, "FC0F9!BC123!40"},
		{'!', 0, true, "FC0F9BC12340"},
	}

	for _, test := range tests {

		s := string(m.format(test.sep, test.chunk, test.upperCase))

		if s != test.res {
			t.Log(s)
			t.Log(test.res)
			t.FailNow()
		}

	}

}
