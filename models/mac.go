package models

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

const (
	UAA       = "UAA"
	LAA       = "LAA"
	Multicast = "multicast"
	Unicast   = "unicast"
	Broadcast = "broadcast"
)

type InvalidMAC struct {
	MAC string
}

func (e InvalidMAC) Error() string {
	return "invalid OUI or MAC address: " + e.MAC
}

type MAC struct {
	hex []byte
}

var (
	reFilter        = regexp.MustCompile(`(?i)[^0-9a-f]+`)
	reNaiveValidate = regexp.MustCompile(`(?i)^([0-9a-f]+[:._-]?)+$`)
	reValidate      = regexp.MustCompile(`(?i)^([0-9a-f]{2}[:._-]?){3}([0-9a-f]+[:._-]?)*$`)
)

// NewMacNaive creates MAC object from string without validating length of given MAC
func NewMacNaive(str string) (MAC, error) {
	if str == "" {
		return MAC{}, nil
	}
	return validateMac(str, reNaiveValidate)
}

// NewMac creates MAC object from string with validating length of given MAC
func NewMac(str string) (MAC, error) {
	return validateMac(str, reValidate)
}

func validateMac(str string, re *regexp.Regexp) (MAC, error) {
	str = strings.TrimSpace(str)

	if !re.MatchString(str) {
		return MAC{}, InvalidMAC{MAC: str}
	}

	s := reFilter.ReplaceAllString(str, "")
	if s == "" {
		return MAC{}, InvalidMAC{MAC: str}
	}

	return MAC{hex: []byte(strings.ToUpper(s))}, nil
}

func (mac MAC) setPrefix(s string) MAC {
	bb := mac.hex
	copy(bb, s)
	m, _ := NewMac(string(bb))
	return m
}

func (mac MAC) Length() int {
	return len(mac.hex)
}

func (mac MAC) TransmissionType() string {
	if len(mac.hex) > 1 {
		if mac.isBroadCast() {
			return Broadcast
		}
		switch mac.hex[1] {
		case '1', '3', '5', '7', '9', 'B', 'D', 'F':
			return Multicast
		default:
			return Unicast
		}
	}
	return ""
}

func (mac MAC) isBroadCast() bool {
	if !mac.IsValid() {
		return false
	}

	for _, quartet := range mac.hex {
		if quartet != 'F' {
			return false
		}
	}

	return true
}

func (mac MAC) AdministrationType() string {
	if len(mac.hex) > 1 {
		switch mac.hex[1] {
		case '2', '3', '6', '7', 'A', 'B', 'E', 'F':
			return LAA
		default:
			return UAA
		}
	}
	return ""
}

func (mac MAC) IsValid() bool {
	return len(mac.hex) == 12 || len(mac.hex) == 16
}

func (mac MAC) String() string {
	return string(mac.hex)
}

func (mac MAC) Format() string {
	if len(mac.hex) < 3 {
		return string(mac.hex)
	}
	l := (len(mac.hex)-1)/2 + len(mac.hex)
	res := make([]byte, l)
	for i, k := 0, 0; i < l; i++ {
		if i%3 == 2 {
			res[i] = ':'
		} else {
			res[i] = mac.hex[k]
			k++
		}
	}

	return string(res)
}

func (mac *MAC) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	m, err := NewMac(s)
	if err != nil {
		return err
	}
	*mac = m
	return nil
}

func (mac MAC) MarshalJSON() ([]byte, error) {
	s := fmt.Sprintf(`"%s"`, mac.Format())
	return []byte(s), nil
}

func (mac MAC) BorderLeft() MAC {
	s := mac.String()
	res, _ := NewMac(s + strings.Repeat("0", 12-len(s)))
	return res
}

func (mac MAC) BorderRight() MAC {
	s := mac.String()
	res, _ := NewMac(s + strings.Repeat("F", 12-len(s)))
	return res
}

func (mac MAC) Incr() MAC {
	m := MAC{
		hex: make([]byte, len(mac.hex)),
	}

	incr := true

	for i := len(mac.hex) - 1; i >= 0; i-- {
		m.hex[i] =
			mac.hex[i]
		if incr {
			if m.hex[i] != 'F' {
				incr = false
			}
			switch m.hex[i] {
			case 'F':
				m.hex[i] = '0'
			case '9':
				m.hex[i] = 'A'
			default:
				m.hex[i]++
			}
		}
	}

	if incr {
		m.hex = make([]byte, len(mac.hex)+2)
		m.hex[0] = '0'
		m.hex[1] = '1'
	}

	return m
}

func (mac MAC) Ceil() MAC {
	l := len(mac.hex)
	if l%2 == 0 {
		return mac
	}

	bb := make([]byte, l+1)
	copy(bb, mac.hex)
	bb[l] = '0'

	return MAC{
		hex: bb,
	}
}

func (mac MAC) Resize(n uint) MAC {
	bb := bytes.Repeat([]byte{'0'}, int(n))
	copy(bb, mac.hex)
	return MAC{hex: bb}
}
