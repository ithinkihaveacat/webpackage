package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/WICG/webpackage/go/signedexchange/structuredheader"

	"github.com/WICG/webpackage/go/signedexchange"
)

var (
	flagInput     = flag.String("i", "", "Signed-exchange input file")
	flagSignature = flag.Bool("signature", false, "Print only signature value")
	flagVerify    = flag.Bool("verify", false, "Perform signature verification")
	flagCert      = flag.String("cert", "", "Certificate CBOR file. If specified, used instead of fetching from signature's cert-url")
	flagJSON      = flag.Bool("json", false, "Print output as JSON")
)

func run() error {
	in := os.Stdin
	if *flagInput != "" {
		var err error
		in, err = os.Open(*flagInput)
		if err != nil {
			return err
		}
		defer in.Close()
	}

	e, err := signedexchange.ReadExchange(in)
	if err != nil {
		return err
	}

	if *flagJSON {
		jsonPrintHeaders(e, time.Now(), signedexchange.DefaultCertFetcher, os.Stdout)
		return nil
	}

	if *flagSignature {
		fmt.Println(e.SignatureHeaderValue)
		return nil
	}

	if *flagVerify {
		if err := verify(e); err != nil {
			return err
		}
		fmt.Println()
	}

	e.PrettyPrintHeaders(os.Stdout)
	e.PrettyPrintPayload(os.Stdout)

	return nil
}

func verify(e *signedexchange.Exchange) error {
	certFetcher := signedexchange.DefaultCertFetcher
	if *flagCert != "" {
		f, err := os.Open(*flagCert)
		if err != nil {
			return fmt.Errorf("could not open %s: %v", *flagCert, err)
		}
		defer f.Close()
		certBytes, err := ioutil.ReadAll(f)
		if err != nil {
			return fmt.Errorf("could not read %s: %v", *flagCert, err)
		}
		certFetcher = func(_ string) ([]byte, error) {
			return certBytes, nil
		}
	}

	verificationTime := time.Now()
	if decodedPayload, ok := e.Verify(verificationTime, certFetcher, log.New(os.Stdout, "", 0)); ok {
		e.Payload = decodedPayload
		fmt.Println("The exchange has a valid signature.")
	}
	return nil
}

func jsonPrintHeaders(e *signedexchange.Exchange, verificationTime time.Time, certFetcher func(url string) ([]byte, error), w io.Writer) {
	_, ok := e.Verify(verificationTime, certFetcher, log.New(ioutil.Discard, "", 0))
	h, _ := structuredheader.ParseParameterisedList(e.SignatureHeaderValue)
	f := struct {
		Valid                bool
		Payload              []byte                      // hides "Payload" in nested signedexchange.Exchange
		SignatureHeaderValue structuredheader.Parameters // hides "SignatureHeaderValue" in nested signedexchange.Exchange
		*signedexchange.Exchange
	}{
		ok,
		[]byte{}, // make "Payload" "empty" (the tag `json:"-"` is supposed to do this, but doesn't--?)
		h[0].Params,
		e,
	}
	s, _ := json.MarshalIndent(f, "", "  ")
	w.Write(s)
}

func main() {
	flag.Parse()
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
