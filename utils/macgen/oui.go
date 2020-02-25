package macgen

import (
	"context"
	"github.com/gomodule/redigo/redis"
	"macaddress_io_grabber/utils/redispool"
)

type generatorOUI struct {
	length    uint
	upper     bool
	separator byte
	chunkSize uint
	count     uint
	gen       func() mac
	rPool     *redispool.Pool
}

func (g *generatorOUI) Generate(ctx context.Context) <-chan result {
	ch := make(chan result, 256)

	go g.generate(ctx, ch)

	return ch
}

func (g *generatorOUI) generate(ctx context.Context, ch chan<- result) {

	defer close(ch)

	conn := g.rPool.Get()
	defer func() {
		_ = conn.Close()
	}()

	set := make(map[string]struct{})

	for i := uint(0); i < g.count; {
		m := g.gen()

		oui, err := redis.String(conn.Do("SRANDMEMBER", "br:glb:ouis"))
		if err != nil {
			ch <- result{
				Error: err,
			}
			return
		}

		err = m.setPrefix(oui)
		if err != nil {
			ch <- result{
				Error: err,
			}
			return
		}

		_, ok := set[string(m)]
		if ok {
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
