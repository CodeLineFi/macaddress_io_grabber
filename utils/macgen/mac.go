package macgen

import (
	"bytes"
	"encoding/hex"
)

type mac []byte

func newMAC(m string) (mac, error) {
	l := (len(m) + 1) / 2

	res := make(mac, l)

	if len(m)&1 != 0 {
		m += "0"
	}

	err := res.decode([]byte(m))
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (m mac) encode() []byte {
	bb := make([]byte, len(m)*2)
	hex.Encode(bb, m)
	return bb
}

func (m mac) decode(bb []byte) error {
	_, err := hex.Decode(m, bb)
	return err
}

func (m mac) setPrefix(s string) error {

	bb := m.encode()

	copy(bb, s)

	return m.decode(bb)
}

func (m mac) setTransType(multicast bool) {
	if len(m) < 1 {
		return
	}

	const b = byte(1)

	if multicast {
		m[0] = m[0] | b
	} else {
		m[0] = m[0] & ^b
	}
}

func (m mac) isMulticast() bool {
	return m[0]&byte(1) != 0
}

func (m mac) setAdminType(local bool) {
	if len(m) < 1 {
		return
	}

	const b = byte(2)

	if local {
		m[0] |= b
	} else {
		m[0] &= ^b
	}
}

func (m mac) isLocal() bool {
	return m[0]&byte(2) != 0
}

func (m mac) format(sep byte, chunk uint, upperCase bool) []byte {

	s := m.encode()
	if upperCase {
		s = bytes.ToUpper(s)
	}

	el := uint(len(s))

	if chunk < 1 || el <= chunk {
		return s
	}

	l := (el-1)/chunk + el

	res := make([]byte, l)

	for i, k := uint(0), uint(0); i < l; i++ {
		if i%(chunk+1) == chunk {
			res[i] = sep
		} else {
			res[i] = s[k]
			k++
		}
	}

	return res
}
