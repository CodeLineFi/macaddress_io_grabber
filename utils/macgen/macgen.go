package macgen

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"macaddress_io_grabber/models"
	"macaddress_io_grabber/utils/redispool"
	"math/big"
	"math/rand"
	"strings"
	"time"
)

var (
	ErrInconsistentAdminType = errors.New("macgen: inconsistent administration type")
	ErrInconsistentTransType = errors.New("macgen: inconsistent transmission type")
	ErrIncorrectRange        = errors.New("macgen: range is incorrect")
)

func init() {
	r = rand.New(rand.NewSource(time.Now().UnixNano()))

	go func() {
		for {
			i := r.Uint64()
			b := make([]byte, 8)
			binary.LittleEndian.PutUint64(b, i)

			select {
			case gb <- b:
			case gi <- i:
			}

		}
	}()
}

var r *rand.Rand
var gb = make(chan []byte, 128)
var gi = make(chan uint64, 128)

func generate(length uint) []byte {

	if length < 1 {
		return []byte{}
	}

	b := make([]byte, (length+7)/8*8)

	for i := len(b); i > 0; i -= 8 {
		copy(b[i-8:i], <-gb)
	}

	b = b[:length]

	return b
}

type result struct {
	MAC   []byte
	Error error
}

type Generator interface {
	Generate(ctx context.Context) <-chan result
}

type GeneratorFactory struct {
	Length    int
	Upper     bool
	Multicast bool
	Local     bool
	Separator byte
	ChunkSize uint
	Count     uint
}

func (gf *GeneratorFactory) ByPrefixGenerator(prefix string) (Generator, error) {

	m, err := models.NewMacNaive(prefix)
	if err != nil {
		return nil, err
	}

	// Check Administration and Transmission Types
	if m.Length() >= 2 {
		if (m.AdministrationType() == models.LAA) != gf.Local {
			return nil, ErrInconsistentAdminType
		}
		if (m.TransmissionType() == models.Multicast) != gf.Multicast {
			return nil, ErrInconsistentTransType
		}
	}

	maxCount := big.NewInt(1)
	if gf.Length*2-m.Length() > 0 {
		maxCount.SetString(strings.Repeat("F", gf.Length*2-m.Length()), 16)
		maxCount.Add(maxCount, big.NewInt(1))

		if m.Length() < 2 {
			maxCount.Div(maxCount, big.NewInt(4))
		}
	}

	var g *generatorPrefix

	g = &generatorPrefix{
		prefix:    m.String(),
		length:    0,
		upper:     gf.Upper,
		multicast: gf.Multicast,
		local:     gf.Local,
		separator: gf.Separator,
		chunkSize: gf.ChunkSize,
		count:     gf.Count,
		maxCount:  maxCount,
		gen: func() mac {
			return mac(generate(g.length))
		},
	}

	if gf.Length > 0 {
		g.length = uint(gf.Length)
	}

	return g, nil
}

func (gf *GeneratorFactory) ByRandPrefixGenerator() (Generator, error) {

	m := mac(<-gb)

	m.setTransType(gf.Multicast)
	m.setAdminType(gf.Local)

	pr := m.format(' ', 0, true)

	switch (<-gi) % 3 {
	case 0:
		pr = pr[:6]
	case 1:
		pr = pr[:7]
	case 2:
		pr = pr[:9]
	}

	return gf.ByPrefixGenerator(string(pr))
}

func (gf *GeneratorFactory) ByOUIGenerator(pool *redispool.Pool) (Generator, error) {

	var g *generatorOUI

	g = &generatorOUI{
		rPool:     pool,
		length:    0,
		upper:     gf.Upper,
		separator: gf.Separator,
		chunkSize: gf.ChunkSize,
		count:     gf.Count,
		gen: func() mac {
			return mac(generate(g.length))
		},
	}

	if gf.Length > 0 {
		g.length = uint(gf.Length)
	}

	return g, nil
}

func (gf *GeneratorFactory) ByRangeGenerator(prefix1, prefix2 string) (Generator, error) {

	var commonLen int

	p1, err := fixPrefix(prefix1)
	if err != nil {
		return nil, err
	}
	p2, err := fixPrefix(prefix2)
	if err != nil {
		return nil, err
	}

	// Trim prefixes to generated MAC length
	if len(p1) > gf.Length*2 {
		p1 = p1[:gf.Length*2]
	}
	if len(p2) > gf.Length*2 {
		p2 = p2[:gf.Length*2]
	}

	// Extend Prefixes to generated MAC length
	b := bytes.Repeat([]byte{'0'}, gf.Length*2)
	copy(b, p1)
	p1 = b

	b = bytes.Repeat([]byte{'f'}, gf.Length*2)
	copy(b, p2)
	p2 = b

	p1, _ = newMAC(string(p1))
	p2, _ = newMAC(string(p2))

	if bytes.Compare(p1, p2) > 0 {
		return nil, ErrIncorrectRange
	}

	// Try to move borders to exclude irrelevant values from range
	if t := fixRangeLeft(p1, gf.Multicast, gf.Local); t != 0 {
		if bytes.Compare(p1, p2) > 0 {
			if t&2 != 0 {
				return nil, ErrInconsistentAdminType
			}
			return nil, ErrInconsistentTransType
		}
	}

	if t := fixRangeRight(p2, gf.Multicast, gf.Local); t != 0 {
		if bytes.Compare(p1, p2) > 0 {
			if t&2 != 0 {
				return nil, ErrInconsistentAdminType
			}
			return nil, ErrInconsistentTransType
		}
	}

	// Find piece which is common for 2 prefixes
	for i := 0; i < len(p1) && i < len(p2); i++ {
		if p1[i] != p2[i] {
			break
		}
		commonLen++
	}

	maxCount := new(big.Int).Sub(new(big.Int).SetBytes(p2), new(big.Int).SetBytes(p1))
	maxCount.Add(maxCount, big.NewInt(1))

	var g *generatorRange

	g = &generatorRange{
		length:    0,
		upper:     gf.Upper,
		multicast: gf.Multicast,
		local:     gf.Local,
		separator: gf.Separator,
		chunkSize: gf.ChunkSize,
		prefix:    prefix1[:commonLen],
		first:     p1,
		last:      p2,
		count:     gf.Count,
		maxCount:  maxCount,
		gen: func(m *big.Int, n *big.Int) *big.Int {
			l := new(big.Int).Sub(n, m)
			l = new(big.Int).Rand(r, l)
			l = new(big.Int).Add(m, l)
			return l
		},
	}

	if gf.Length > 0 {
		g.length = uint(gf.Length)
	}

	return g, nil
}

func fixPrefix(p string) ([]byte, error) {
	m, err := models.NewMacNaive(p)
	if err != nil {
		return nil, err
	}

	mac, err := newMAC(m.String())
	if err != nil {
		return nil, err
	}

	return mac.encode()[:m.Length()], nil
}

// Moves left border up
func fixRangeLeft(p mac, m, l bool) byte {

	n := p[0]
	t := p[0]

	for (n&1 != 0 != m) || (n&2 != 0 != l) {
		if n == 255 {
			// Dead end
			for i := len(p) - 1; i > 0; i-- {
				p[i] = 255
			}
			return (t | n) & ^(t & n) & 3
		}
		n++
	}

	if t != n {
		p[0] = n
		for i := len(p) - 1; i > 0; i-- {
			p[i] = 0
		}
	}

	return (t | n) & ^(t & n) & 3
}

// Moves right border down
func fixRangeRight(p mac, m, l bool) byte {

	n := p[0]
	t := p[0]

	for (n&1 != 0 != m) || (n&2 != 0 != l) {
		if n == 0 {
			// Dead end
			for i := len(p) - 1; i > 0; i-- {
				p[i] = 0
			}
			return (t | n) & ^(t & n) & 3
		}
		n--
	}

	if t != n {
		p[0] = n
		for i := len(p) - 1; i > 0; i-- {
			p[i] = 255
		}
	}

	return (t | n) & ^(t & n) & 3
}
