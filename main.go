package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"time"

	"github.com/ddo/go-spin"
	"github.com/nknorg/go-fast"
)

var (
	Version string
)

func main() {
	seedHex := flag.String("s", "", "wallet secret seed")
	outputFile := flag.String("o", "", "write result to file in unit of Kbps")
	version := flag.Bool("version", false, "print version")

	flag.Parse()

	if *version {
		fmt.Println(Version)
		return
	}

	seed, err := hex.DecodeString(*seedHex)
	if err != nil {
		log.Fatal(err)
	}

	status := ""
	spinner := spin.New("")

	ticker := time.NewTicker(100 * time.Millisecond)

	defer ticker.Stop()

	go func() {
		for range ticker.C {
			fmt.Printf("%c[2K %s  %s\r", 27, spinner.Spin(), status)
		}
	}()

	port, err := getFreePort()
	if err != nil {
		log.Fatal(err)
	}

	tp, err := newTunaProxy(seed, port)
	if err != nil {
		log.Fatal(err)
	}

	defer tp.stop()

	err = tp.start()
	if err != nil {
		log.Fatal(err)
	}

	for i := 5; i > 0; i-- {
		status = fmt.Sprintf("starting speedtest in %ds", i)
		time.Sleep(time.Second)
	}

	status = "starting speedtest"

	fastCom, err := fast.New(fmt.Sprintf("http://127.0.0.1:%d", port))
	if err != nil {
		log.Fatal(err)
	}

	err = fastCom.Init()
	if err != nil {
		log.Fatal(err)
		return
	}

	status = "connecting"

	urls, err := fastCom.GetUrls()
	if err != nil {
		log.Fatal(err)
		return
	}

	status = "loading"

	KbpsChan := make(chan float64)
	done := make(chan struct{})

	go func() {
		var Kbps float64
		for Kbps = range KbpsChan {
			status = format(Kbps)
		}

		fmt.Printf("\r%c[2K -> %s\n", 27, status)

		if len(*outputFile) > 0 {
			err = ioutil.WriteFile(*outputFile, []byte(fmt.Sprintf("%f", Kbps)), 0666)
			if err != nil {
				log.Fatal(err)
			}
		}

		close(done)
	}()

	err = fastCom.Measure(urls, KbpsChan)
	if err != nil {
		log.Fatal(err)
	}

	<-done
}

func format(Kbps float64) string {
	unit := "Kbps"
	f := "%.f %s"

	if Kbps > 1000000 { // Gbps
		f = "%.2f %s"
		unit = "Gbps"
		Kbps /= 1000000

	} else if Kbps > 1000 { // Mbps
		f = "%.2f %s"
		unit = "Mbps"
		Kbps /= 1000
	}

	return fmt.Sprintf(f, Kbps, unit)
}
