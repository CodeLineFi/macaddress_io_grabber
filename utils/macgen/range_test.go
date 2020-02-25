package macgen

import (
	"context"
	"log"
	"macaddress_io_grabber/models"
	"math/big"
	"strings"
	"testing"
	"time"
)

func TestGeneratorFactory_ByRangeGenerator01(t *testing.T) {

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
		f     GeneratorFactory
		first string
		last  string
		count uint
		err   error
	}{
		{
			f:     f1,
			first: "1000000000000000",
			last:  "100000FFFFFFFFFF",
			count: f1.Count,
		},
		{
			f:     f1,
			first: "0000000000000000",
			last:  "000000FFFFFFFFFF",
			count: f1.Count,
		},
		{
			f:     f1,
			first: "1000000000000000",
			last:  "100000000000000F",
			count: 16,
		},
		{
			f:     f1,
			first: "1034567891234567",
			last:  "1034567891234567",
			count: 1,
		},
		{
			f:     f1,
			first: "1134567891234567",
			last:  "113456789123FFFF",
			err:   ErrInconsistentTransType,
		},
		{
			f:     f1,
			first: "1234567891234567",
			last:  "123456789123FFFF",
			err:   ErrInconsistentAdminType,
		},
		{
			f:     f1,
			first: "1C00000000000000",
			last:  "1CFFFFFFFFFFFFFF",
			count: f1.Count,
		},
		{
			f:     f1,
			first: "1000000000000000",
			last:  "1CFFFFFFFFFFFFFF",
			count: f1.Count,
		},
		{
			f:     f1,
			first: "1CFFFFFFFFFFFFF0",
			last:  "1CFFFFFFFFFFFFFF",
			count: 16,
		},
		{
			f:     f1,
			first: "123",
			last:  "123",
			err:   ErrInconsistentAdminType,
		},
		{
			f:     f1,
			first: "113",
			last:  "113",
			err:   ErrInconsistentTransType,
		},
		{
			f:     f1,
			first: "12000000",
			last:  "13ffffff",
			err:   ErrInconsistentAdminType,
		},
		{
			f:     f1,
			first: "103",
			last:  "103",
			count: f1.Count,
		},
		{
			f: GeneratorFactory{
				Length:    1,
				ChunkSize: 4,
				Separator: '.',
				Upper:     false,
				Multicast: false,
				Local:     false,
				Count:     100000,
			},
			first: "10",
			last:  "10",
			count: 1,
		},
		{
			f: GeneratorFactory{
				Length:    6,
				ChunkSize: 4,
				Separator: '.',
				Upper:     false,
				Multicast: false,
				Local:     true,
				Count:     1000,
			},
			first: "CEFFFFFFFF00",
			last:  "CFFFFFFFFFFF",
			count: 256,
		},
		{
			f: GeneratorFactory{
				Length:    6,
				ChunkSize: 4,
				Separator: '.',
				Upper:     false,
				Multicast: false,
				Local:     true,
				Count:     1000,
			},
			first: "020000000000",
			last:  "030000000000",
			count: 1000,
		},
		{
			f: GeneratorFactory{
				Length:    6,
				ChunkSize: 4,
				Separator: '.',
				Upper:     false,
				Multicast: false,
				Local:     false,
				Count:     1000,
			},
			first: "010000000000",
			last:  "020000000000",
			err:   ErrInconsistentTransType,
		},
		{
			f: GeneratorFactory{
				Length:    6,
				ChunkSize: 4,
				Separator: '.',
				Upper:     false,
				Multicast: true,
				Local:     false,
				Count:     1000,
			},
			first: "010000000000",
			last:  "020000000000",
			count: 1000,
		},
		{
			f: GeneratorFactory{
				Length:    6,
				ChunkSize: 4,
				Separator: '.',
				Upper:     false,
				Multicast: false,
				Local:     true,
				Count:     1000,
			},
			first: "010000000000",
			last:  "020000000000",
			count: 1,
		},
		{
			f: GeneratorFactory{
				Length:    6,
				ChunkSize: 4,
				Separator: '.',
				Upper:     false,
				Multicast: true,
				Local:     true,
				Count:     1000,
			},
			first: "010000000000",
			last:  "020000000000",
			err:   ErrInconsistentAdminType,
		},
	} {
		g, err := test.f.ByRangeGenerator(test.first, test.last)
		if err != nil {
			if err == test.err {
				continue
			}
			t.Log(test)
			t.Log(err)
			t.FailNow()
		}

		if test.err != nil {
			log.Println("has to be failed with", test.err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)

		defer cancel()

		m1, _ := new(big.Int).SetString(test.first, 16)
		m2, _ := new(big.Int).SetString(test.last, 16)

		i := uint(0)

		for r := range g.Generate(ctx) {

			if r.Error != nil {
				t.Log(test)
				t.Log(r.Error)
				t.FailNow()
			}

			i++
			res, err := models.NewMacNaive(string(r.MAC))
			if err != nil {
				t.Log(test)
				t.Log(err)
				t.FailNow()
			}
			n, ok := new(big.Int).SetString(res.String(), 16)
			if !ok {
				t.Log(test)
				t.Log("cannot parse", res.String())
				t.FailNow()
			}

			if (n.Cmp(m1) < 0 || n.Cmp(m2) > 0) && !strings.HasPrefix(res.String(), test.first) {
				t.Log(test)
				t.Log(res)
				t.Log(n.Bytes())
				t.Log(m1.Bytes())
				t.Log(m2.Bytes())
				t.FailNow()
			}

			if res.AdministrationType() == models.LAA != test.f.Local {
				t.Log(test.first, test.last)
				t.Log(res, "has wrong admin type:", res.AdministrationType())
				t.FailNow()
			}

			if res.TransmissionType() == models.Multicast != test.f.Multicast {
				t.Log(test.first, test.last)
				t.Log(res, "has wrong trans type:", res.AdministrationType())
				t.FailNow()
			}
		}

		if i != test.count {
			t.Log(test)
			t.Log(i, test.count)
			t.FailNow()
		}
	}
}

func init() {
	log.SetFlags(log.Flags() | log.Lshortfile)
}

func TestGeneratorFactory_ByPrefixGenerator02(t *testing.T) {

	type res struct {
		count uint
		err   error
	}

	for _, test := range []struct {
		first string
		last  string
		count uint
		res   [4]res
	}{
		{
			first: "00",
			last:  "01",
			res: [4]res{
				{count: 256},
				{count: 256},
				{err: ErrInconsistentAdminType},
				{err: ErrInconsistentAdminType},
			},
		},
		{
			first: "01",
			last:  "02",
			res: [4]res{
				{err: ErrInconsistentTransType},
				{count: 256},
				{count: 256},
				{err: ErrInconsistentAdminType},
			},
		},
		{
			first: "02",
			last:  "03",
			res: [4]res{
				{err: ErrInconsistentAdminType},
				{err: ErrInconsistentAdminType},
				{count: 256},
				{count: 256},
			},
		},
		{
			first: "03",
			last:  "04",
			res: [4]res{
				{count: 256},
				{err: ErrInconsistentAdminType},
				{err: ErrInconsistentTransType},
				{count: 256},
			},
		},
		{
			first: "04",
			last:  "05",
			res: [4]res{
				{count: 256},
				{count: 256},
				{err: ErrInconsistentAdminType},
				{err: ErrInconsistentAdminType},
			},
		},
		{
			first: "05",
			last:  "06",
			res: [4]res{
				{err: ErrInconsistentTransType},
				{count: 256},
				{count: 256},
				{err: ErrInconsistentAdminType},
			},
		},
		{
			first: "06",
			last:  "07",
			res: [4]res{
				{err: ErrInconsistentAdminType},
				{err: ErrInconsistentAdminType},
				{count: 256},
				{count: 256},
			},
		},
		{
			first: "07",
			last:  "08",
			res: [4]res{
				{count: 256},
				{err: ErrInconsistentAdminType},
				{err: ErrInconsistentTransType},
				{count: 256},
			},
		},
		{
			first: "F8",
			last:  "F9",
			res: [4]res{
				{count: 256},
				{count: 256},
				{err: ErrInconsistentAdminType},
				{err: ErrInconsistentAdminType},
			},
		},
		{
			first: "F9",
			last:  "FA",
			res: [4]res{
				{err: ErrInconsistentTransType},
				{count: 256},
				{count: 256},
				{err: ErrInconsistentAdminType},
			},
		},
		{
			first: "FA",
			last:  "FB",
			res: [4]res{
				{err: ErrInconsistentAdminType},
				{err: ErrInconsistentAdminType},
				{count: 256},
				{count: 256},
			},
		},
		{
			first: "FB",
			last:  "FC",
			res: [4]res{
				{count: 256},
				{err: ErrInconsistentAdminType},
				{err: ErrInconsistentTransType},
				{count: 256},
			},
		},
		{
			first: "FC",
			last:  "FD",
			res: [4]res{
				{count: 256},
				{count: 256},
				{err: ErrInconsistentAdminType},
				{err: ErrInconsistentAdminType},
			},
		},
		{
			first: "FD",
			last:  "FE",
			res: [4]res{
				{err: ErrInconsistentAdminType},
				{count: 256},
				{count: 256},
				{err: ErrInconsistentAdminType},
			},
		},
		{
			first: "FE",
			last:  "FF",
			res: [4]res{
				{err: ErrInconsistentAdminType},
				{err: ErrInconsistentAdminType},
				{count: 256},
				{count: 256},
			},
		},
		{
			first: "",
			last:  "",
			res: [4]res{
				{count: 1000},
				{count: 1000},
				{count: 1000},
				{count: 1000},
			},
		},
	} {

		for i, res := range test.res {

			f := GeneratorFactory{
				Length:    2,
				ChunkSize: 0,
				Separator: ' ',
				Multicast: i&1 != 0,
				Local:     i&2 != 0,
				Upper:     true,
				Count:     1000,
			}

			g, err := f.ByRangeGenerator(test.first, test.last)
			if err != nil {
				if res.err == err {
					continue
				}
				t.Log(err)
				t.Log(test.first, test.last, i, res.count, res.err)
				t.FailNow()
			}
			if res.err != nil {
				t.Log(f)
				t.Log(test.first, test.last)
				t.Log("has to be failed with error:", res.err)
				t.FailNow()
			}

			ctx, _ := context.WithTimeout(context.Background(), time.Second*10)

			count := uint(0)

			set := map[string]struct{}{}

			for r := range g.Generate(ctx) {
				if r.Error != nil {
					t.Log(f)
					t.Log(err)
					t.FailNow()
				}

				if _, ok := set[string(r.MAC)]; ok {
					t.Log(f)
					t.Log("duplicate record:", string(r.MAC))
					t.FailNow()
				}

				set[string(r.MAC)] = struct{}{}

				res, err := newMAC(string(r.MAC))
				if err != nil {
					t.Log(f)
					t.Log(err)
					t.FailNow()
				}

				if res.isLocal() != f.Local {
					t.Log(f)
					t.Log(res, "has wrong admin type:", res.isLocal())
					t.FailNow()
				}

				if res.isMulticast() != f.Multicast {
					t.Log(f)
					t.Log(res, "has wrong trans type:", res.isMulticast())
					t.FailNow()
				}

				count++
			}

			if res.count != count {
				t.Log(test.first, test.last, i, res.count, res.err)
				t.Log(res.count, count)
				t.FailNow()
			}
		}
	}
}
