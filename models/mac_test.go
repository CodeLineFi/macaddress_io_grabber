package models

import (
	"encoding/json"
	"log"
	"reflect"
	"testing"
)

func TestNewMac(t *testing.T) {
	tests := map[string]struct {
		Str     string
		IsError bool
	}{
		"1d:a9:0E":   {"1DA90E", false},
		"11:11-11":   {"111111", false},
		"11:FA:AE":   {"11FAAE", false},
		"11:F0:00:A": {"11F000A", false},
		"11-FF_00":   {"11FF00", false},
		"11.fE.02":   {"11FE02", false},
		"11FF.a29e":  {"11FFA29E", false},
		"":           {"", true},
		":::":        {"", true},
		"abcde":      {"", true},
		"abcdef":     {"ABCDEF", false},
		"ebceds":     {"", true},
		"00:0:000":   {"", true},
	}

	for str, target := range tests {
		mac, err := NewMac(str)
		if err != nil != target.IsError {
			log.Println(str)
			log.Println(err)
			t.FailNow()
		}
		if mac.String() != target.Str {
			log.Println("Input:", str)
			log.Println("Result:", mac.String())
			log.Println("Target:", target)
			t.FailNow()
		}
	}
}

func TestNewMacNaive(t *testing.T) {
	tests := map[string]struct {
		Str     string
		IsError bool
	}{
		"1d:a9:0E":   {"1DA90E", false},
		"11:11-11":   {"111111", false},
		"11:FA:AE":   {"11FAAE", false},
		"11:F0:00:A": {"11F000A", false},
		"11-FF_00":   {"11FF00", false},
		"11.fE.02":   {"11FE02", false},
		"11FF.a29e":  {"11FFA29E", false},
		"":           {"", false},
		":::":        {"", true},
		"abcde":      {"ABCDE", false},
		"abcdef":     {"ABCDEF", false},
		"ebceds":     {"", true},
		"00:0:000":   {"000000", false},
	}

	for str, target := range tests {
		mac, err := NewMacNaive(str)
		if err != nil != target.IsError {
			log.Println(str)
			log.Println(err)
			t.FailNow()
		}
		if mac.String() != target.Str {
			log.Println("Input:", str)
			log.Println("Result:", mac.String())
			log.Println("Target:", target)
			t.FailNow()
		}
	}
}

func TestMAC_Length(t *testing.T) {

	si := map[string]int{
		"123450":     6,
		"12:34:56":   6,
		"12.34.56.7": 7,
		"12345670":   8,
		"123456789":  9,
		"1234567890": 10,
		"0000000000": 10,
	}

	for str, target := range si {
		mac, err := NewMac(str)
		if err != nil {
			log.Println(str)
			log.Println(err)
			t.FailNow()
		}

		if mac.Length() != target {
			log.Println("Input:", str)
			log.Println("Result:", mac.Length())
			log.Println("Target:", target)
			t.FailNow()
		}
	}

}

func TestMAC_AdministrationType(t *testing.T) {
	ss := map[string]string{
		"10:34:56:78:90:AB:CD:EF": UAA,
		"11:34:56:78:90:AB:CD:EF": UAA,
		"12:34:56:78:90:AB:CD:EF": LAA,
		"13:34:56:78:90:AB":       LAA,
		"14:34:56:78:90:AB":       UAA,
		"15:34:56:78:90:AB:CD:EF": UAA,
		"16:34:56:78:90:AB:CD:EF": LAA,
		"17:34:56:78:90:AB:CD:EF": LAA,
		"18:34:56:78:90:AB:CD:EF": UAA,
		"19:34:56:78:90:AB:CD:EF": UAA,
		"1a:34:56:78:90:AB:CD:EF": LAA,
		"1b:34:56:78:90:AB:CD:EF": LAA,
		"1c:34:56:78:90:AB:CD:EF": UAA,
		"1d:34:56:78:90:AB:CD:EF": UAA,
		"1e:34:56:78:90:AB:CD:EF": LAA,
		"1f:34:56:78:90:AB:CD:EF": LAA,
	}

	for str, res := range ss {
		mac, err := NewMac(str)
		if err != nil {
			log.Println(err)
			t.FailNow()
		}
		if mac.AdministrationType() != res {
			log.Println(str, "AdministrationType:", mac.AdministrationType(), "Should be:", res)
			t.FailNow()
		}
	}
}

func TestMAC_TransmissionType(t *testing.T) {
	ss := map[string]string{
		"12:34:56:78:90:AB:CD:EF": Unicast,
		"13:34:56:78:90:AB:CD:EF": Multicast,
		"10:34:56:78:90:AB":       Unicast,
		"11:34:56:78:90:AB":       Multicast,
		"12:34:56:78:90:AB":       Unicast,
		"13:34:56:78:90:AB":       Multicast,
		"14:34:56:78:90:AB":       Unicast,
		"15:34:56:78:90:AB":       Multicast,
		"16:34:56:78:90:AB":       Unicast,
		"17:34:56:78:90:AB":       Multicast,
		"18:34:56:78:90:AB":       Unicast,
		"19:34:56:78:90:AB":       Multicast,
		"1a:34:56:78:90:AB":       Unicast,
		"1b:34:56:78:90:AB":       Multicast,
		"1c:34:56:78:90:AB":       Unicast,
		"1d:34:56:78:90:AB":       Multicast,
		"1e:34:56:78:90:AB":       Unicast,
		"1f:34:56:78:90:AB":       Multicast,
		"FF:FF:FF:FF:FF":          Multicast,
		"FF:FF:FF:FF:FF:FF":       Broadcast,
		"FF:FF:FF:FF:FF:FF:FF":    Multicast,
		"FF:FF:FF:FF:FF:FF:FF:FF": Broadcast,
		"FF:FF:FF:FF:FF:F1":       Multicast,
		"FF:FF:FF:FF:FF:FF:FF:F1": Multicast,
	}

	for str, res := range ss {
		mac, err := NewMac(str)
		if err != nil {
			log.Println(err)
			t.FailNow()
		}
		if mac.TransmissionType() != res {
			log.Println(str, "TransmissionType:", mac.TransmissionType(), "Should be:", res)
			t.FailNow()
		}
	}
}

func TestMAC_MarshalJSON(t *testing.T) {

	input := []byte(`"AB:CD:EF"`)

	m := new(MAC)
	err := json.Unmarshal(input, m)
	if err != nil {
		log.Println(string(input))
		log.Println(err)
		t.FailNow()
	}

	output, err := json.Marshal(m)
	if err != nil {
		log.Println(string(input))
		log.Println(err)
		t.FailNow()
	}

	if !reflect.DeepEqual(input, output) {
		log.Println("Result", string(output))
		log.Println("Target", string(input))
		t.FailNow()
	}
}

func TestMAC_Format(t *testing.T) {

	m := map[string]string{
		"abcd:01":  "AB:CD:01",
		"abcde0":   "AB:CD:E0",
		"abcdef":   "AB:CD:EF",
		"abcdef0":  "AB:CD:EF:0",
		"abcdef01": "AB:CD:EF:01",
	}

	for input, target := range m {
		m, _ := NewMac(input)
		if m.Format() != target {
			log.Println(input, "should be", target, "but instead", m.Format())
			t.FailNow()
		}
		m, _ = NewMacNaive(input)
		if m.Format() != target {
			log.Println(input, "should be", target, "but instead", m.Format())
			t.FailNow()
		}
	}
}

func TestMAC_String(t *testing.T) {

	m := map[string]string{
		"abcd:01":  "ABCD01",
		"abcde0":   "ABCDE0",
		"abcdef":   "ABCDEF",
		"abcdef0":  "ABCDEF0",
		"abcdef01": "ABCDEF01",
	}

	for input, target := range m {
		m, _ := NewMac(input)
		if m.String() != target {
			log.Println(input, "should be", target, "but instead", m.Format())
			t.FailNow()
		}
		m, _ = NewMacNaive(input)
		if m.String() != target {
			log.Println(input, "should be", target, "but instead", m.Format())
			t.FailNow()
		}
	}
}

func TestMAC_IsValid(t *testing.T) {

	m := map[string]bool{
		"":                  false,
		"a":                 false,
		"aa":                false,
		"aaa":               false,
		"aaaa":              false,
		"aaaaa":             false,
		"aaaaaa":            false,
		"aaaaaaa":           false,
		"aaaaaaaa":          false,
		"aaaaaaaaa":         false,
		"aaaaaaaaaa":        false,
		"aaaaaaaaaaaa":      true,
		"aaaaaaaaaaaaa":     false,
		"aaaaaaaaaaaaaa":    false,
		"aaaaaaaaaaaaaaa":   false,
		"aaaaaaaaaaaaaaaa":  true,
		"aaaaaaaaaaaaaaaaa": false,
	}

	for input, target := range m {
		m, _ := NewMac(input)
		if m.IsValid() != target {
			log.Println(input, "should be", target, "but instead", m.Format())
			t.FailNow()
		}
	}
}
