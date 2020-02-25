package macgen

import (
	"context"
	"macaddress_io_grabber/models"
	"strings"
	"testing"
	"time"
)

func TestGeneratorFactory_ByPrefixGenerator(t *testing.T) {

	f1 := GeneratorFactory{
		Length:    8,
		ChunkSize: 4,
		Separator: '.',
		Local:     false,
		Multicast: false,
		Upper:     false,
		Count:     10000,
	}

	for _, test := range []struct {
		f      GeneratorFactory
		prefix string
		count  uint
		err    error
	}{
		{
			f:      f1,
			prefix: "100000",
		},
		{
			f:      f1,
			prefix: "000000",
		},
		{
			f:      f1,
			prefix: "100000000000000",
			count:  16,
		},
		{
			f:      f1,
			prefix: "103456789123456789",
			count:  1,
		},
		{
			f:      f1,
			prefix: "1F",
			err:    ErrInconsistentAdminType,
		},
		{
			f:      f1,
			prefix: "1E",
			err:    ErrInconsistentAdminType,
		},
		{
			f:      f1,
			prefix: "1D",
			err:    ErrInconsistentTransType,
		},
		{
			f:      f1,
			prefix: "1C",
		},
		{
			f:      f1,
			prefix: "",
		},
		{
			f: GeneratorFactory{
				Length:    1,
				ChunkSize: 4,
				Separator: '.',
				Local:     false,
				Multicast: false,
				Upper:     false,
				Count:     100000,
			},
			prefix: "10",
			count:  1,
		},
	} {
		g, err := test.f.ByPrefixGenerator(test.prefix)
		if err != nil {
			if err == test.err {
				continue
			}
			t.Log(test)
			t.Log(err)
			t.FailNow()
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)

		defer cancel()

		i := uint(0)

		for r := range g.Generate(ctx) {

			if r.Error != nil {
				t.Log(test)
				t.Log(r.Error)
				t.Log(i)
				t.FailNow()
			}

			i++

			res, err := models.NewMacNaive(string(r.MAC))
			if err != nil {
				t.Log(test)
				t.Log(err)
				t.FailNow()
			}

			if !strings.HasPrefix(res.String(), test.prefix) && !strings.HasPrefix(test.prefix, res.String()) {
				t.Log(res.String(), test.prefix)
				t.FailNow()
			}
		}

		if test.count > 0 && i != test.count ||
			test.count == 0 && i != test.f.Count {

			t.Log(test)
			t.Log(i, test.f.Count)
			t.FailNow()
		}
	}
}
