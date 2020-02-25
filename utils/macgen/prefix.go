package macgen

import (
	"context"
	"math/big"
)

type generatorPrefix struct {
	length    uint
	upper     bool
	multicast bool
	local     bool
	separator byte
	chunkSize uint
	count     uint
	gen       func() mac
	prefix    string
	maxCount  *big.Int
}

func (g *generatorPrefix) Generate(ctx context.Context) <-chan result {

	ch := make(chan result, 256)

	if g.maxCount.IsUint64() {
		mc := g.maxCount.Uint64()
		if len(g.prefix) < 2 {
			mc /= 4
		}
		if mc < uint64(g.count*16) {
			go g.generateGreedy(ctx, ch, mc)
			return ch
		}
	}
	go g.generateLazy(ctx, ch)

	return ch
}

func (g *generatorPrefix) generateGreedy(ctx context.Context, ch chan<- result, maxCount uint64) {

	defer close(ch)

	first := make(mac, g.length)
	err := first.setPrefix(g.prefix)
	if err != nil {
		ch <- result{Error: err}
		return
	}
	first.setTransType(g.multicast)
	first.setAdminType(g.local)

	n := new(big.Int).SetBytes(first)

	macs := make(map[string]struct{})

	one := big.NewInt(1)

	for i := uint64(0); i < maxCount; i++ {

		m := make(mac, g.length)

		bb := n.Bytes()
		copy(m[len(m)-len(bb):], bb)

		if len(g.prefix) < 2 {
			m.setTransType(g.multicast)
			m.setAdminType(g.local)
		}

		s := m.format(g.separator, g.chunkSize, g.upper)
		macs[string(s)] = struct{}{}

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

func (g *generatorPrefix) generateLazy(ctx context.Context, ch chan<- result) {

	defer close(ch)

	macs := make(map[string]struct{})

	for i := uint(0); i < g.count; {
		m := g.gen()

		err := m.setPrefix(g.prefix)

		if err != nil {
			ch <- result{Error: err}
			return
		}

		m.setTransType(g.multicast)
		m.setAdminType(g.local)

		_, ok := macs[string(m)]
		if ok {
			select {
			case <-ctx.Done():
				ch <- result{Error: ctx.Err()}
				return
			default:
				continue
			}
		}

		i++

		macs[string(m)] = struct{}{}

		s := m.format(g.separator, g.chunkSize, g.upper)

		select {
		case <-ctx.Done():
			ch <- result{Error: ctx.Err()}
			return
		case ch <- result{MAC: s}:
		}
	}
}
