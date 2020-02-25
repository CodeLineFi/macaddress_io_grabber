package download

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	ieeeMaL   = "http://standards-oui.ieee.org/oui/oui.txt"
	ieeeMaM   = "http://standards-oui.ieee.org/oui28/mam.txt"
	ieeeMaS   = "http://standards-oui.ieee.org/oui36/oui36.txt"
	ieeeIAB   = "http://standards-oui.ieee.org/iab/iab.txt"
	ieeeCID   = "http://standards-oui.ieee.org/cid/cid.txt"
	wireshark = "https://code.wireshark.org/review/gitweb?p=wireshark.git;a=blob_plain;f=manuf.tmpl;hb=HEAD"
)

const (
	MalFile       = "mal.txt"
	MamFile       = "mam.txt"
	MasFile       = "mas.txt"
	IabFile       = "iab.txt"
	CidFile       = "cid.txt"
	WiresharkFile = "wireshark.txt"
)

func Download(dir string) (err error) {

	err = os.MkdirAll(dir, 0755)
	if err != nil {
		return
	}

	err = downloadR(filepath.Join(dir, MalFile), ieeeMaL)
	if err != nil {
		return errors.New(MalFile + ": " + err.Error())
	}
	err = downloadR(filepath.Join(dir, MamFile), ieeeMaM)
	if err != nil {
		return errors.New(MamFile + ": " + err.Error())
	}
	err = downloadR(filepath.Join(dir, MasFile), ieeeMaS)
	if err != nil {
		return errors.New(MasFile + ": " + err.Error())
	}
	err = downloadR(filepath.Join(dir, IabFile), ieeeIAB)
	if err != nil {
		return errors.New(IabFile + ": " + err.Error())
	}
	err = downloadR(filepath.Join(dir, CidFile), ieeeCID)
	if err != nil {
		return errors.New(CidFile + ": " + err.Error())
	}
	err = downloadR(filepath.Join(dir, WiresharkFile), wireshark)
	if err != nil {
		return errors.New(WiresharkFile + ": " + err.Error())
	}

	return nil
}

func downloadR(filename string, uri string) (err error) {
	for i := 0; i < 4; i++ {
		err = download(filename, uri)
		if err == nil {
			return
		}
		time.Sleep(30 * time.Second)
	}
	return
}

func download(filename string, uri string) (err error) {

	var out *os.File
	var resp *http.Response

	out, err = os.Create(filename)
	if err != nil {
		return
	}

	r, err := http.NewRequest(http.MethodGet, uri, nil)

	ctx, cancel := context.WithTimeout(r.Context(), time.Minute*10)
	defer cancel()

	r = r.WithContext(ctx)

	resp, err = http.DefaultClient.Do(r)
	if err != nil {
		return
	}
	defer func() {
		e := resp.Body.Close()
		if err == nil {
			err = e
		}
	}()

	_, err = io.Copy(out, resp.Body)

	return
}
