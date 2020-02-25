package macgen

import (
	"bytes"
	"context"
	"math/big"
)

type generatorRange struct {
	length    uint
	upper     bool
	multicast bool
	local     bool
	separator byte
	chunkSize uint
	count     uint
	gen       func(m, n *big.Int) *big.Int
	first     []byte
	last      []byte
	prefix    string
	maxCount  *big.Int
}

func (g *generatorRange) Generate(ctx context.Context) <-chan result {

	ch := make(chan result, 256)

	if g.maxCount.IsUint64() {
		mc := g.maxCount.Uint64()
		if mc < uint64(g.count*16) {
			go g.generateGreedy(ctx, ch, mc)
			return ch
		}
	}
	go g.generateLazy(ctx, ch)

	return ch
}

func (g *generatorRange) generateGreedy(ctx context.Context, ch chan<- result, maxCount uint64) {
	defer close(ch)

	n := new(big.Int).SetBytes(g.first)

	macs := make(map[string]struct{})

	one := big.NewInt(1)

	for i := uint64(0); i < maxCount; i++ {

		m := make(mac, g.length)

		bb := n.Bytes()
		copy(m[len(m)-len(bb):], bb)

		if m.isMulticast() == g.multicast && m.isLocal() == g.local {
			s := m.format(g.separator, g.chunkSize, g.upper)
			macs[string(s)] = struct{}{}
		}

		n.Add(n, one)
	}

	i := uint(0)

	for m := range macs {
		i++
		if i > g.count {
			return
		}
		select {
		case <-ctx.Done():
			ch <- result{Error: ctx.Err()}
			return
		case ch <- result{MAC: []byte(m)}:
		}
	}
}

func (g *generatorRange) generateLazy(ctx context.Context, ch chan<- result) {
	defer close(ch)

	set := make(map[string]struct{})

	f := new(big.Int).SetBytes(g.first)
	l := new(big.Int).SetBytes(g.last)

	for i := uint(0); i < g.count; {

		bb := g.gen(f, l).Bytes()

		m := make(mac, g.length)

		copy(m[len(m)-len(bb):], bb)

		m.setTransType(g.multicast)
		m.setAdminType(g.local)

		_, retry := set[string(m)]

		if bytes.Compare(g.first, m) > 0 || bytes.Compare(m, g.last) > 0 {
			retry = true
		}

		if retry {
			select {
			case <-ctx.Done():
				ch <- result{
					Error: ctx.Err(),
				}
				return
			default:
				continue
			}
		}

		set[string(m)] = struct{}{}

		i++

		select {
		case <-ctx.Done():
			ch <- result{
				Error: ctx.Err(),
			}
			return
		case ch <- result{MAC: m.format(g.separator, g.chunkSize, g.upper)}:
		}
	}
}
