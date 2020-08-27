package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/ddo/go-spin"
	"github.com/nknorg/go-fast"
)

var (
	Version string
)

func main() {
	numTests := flag.Int("n", 1, "number of tests")
	seedHex := flag.String("s", "", "wallet secret seed")
	uploadURL := flag.String("upload", "", "upload result to server")
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

	res := make([]float64, 0, *numTests)

	for i := 0; i < *numTests; i++ {
		err := func() error {
			port, err := getFreePort()
			if err != nil {
				return err
			}

			tp, err := newTunaProxy(seed, port)
			if err != nil {
				return err
			}

			err = tp.start()
			if err != nil {
				return err
			}

			for i := 5; i > 0; i-- {
				status = fmt.Sprintf("starting speedtest in %ds", i)
				time.Sleep(time.Second)
			}

			status = "starting speedtest"

			fastCom, err := fast.New(fmt.Sprintf("http://127.0.0.1:%d", port))
			if err != nil {
				return err
			}

			err = fastCom.Init()
			if err != nil {
				return err
			}

			status = "connecting"

			urls, err := fastCom.GetUrls()
			if err != nil {
				return err
			}

			status = "loading"

			KbpsChan := make(chan float64)
			done := make(chan struct{})

			var Kbps float64
			go func() {
				for Kbps = range KbpsChan {
					status = format(Kbps)
				}
				fmt.Printf("\r%c[2K -> %s\n", 27, status)
				close(done)
			}()

			err = fastCom.Measure(urls, KbpsChan)
			if err != nil {
				return err
			}

			<-done

			tp.stop()

			if Kbps > 0 {
				res = append(res, Kbps)
			}

			return nil
		}()
		if err != nil {
			log.Println(err)
		}
	}

	if len(res) == 0 {
		return
	}

	log.Println("Results:")
	for _, Kbps := range res {
		fmt.Println(format(Kbps))
	}

	if len(*uploadURL) > 0 {
		b, err := json.Marshal(struct {
			Throughput []float64
		}{
			Throughput: res,
		})
		if err != nil {
			log.Fatalf("Upload results error: %v", err)
		}

		client := &http.Client{
			Timeout: 30 * time.Second,
		}

		resp, err := client.Post(*uploadURL, "application/json", bytes.NewBuffer(b))
		if err != nil {
			log.Fatalf("Upload results error: %v", err)
		}

		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatalf("Read response error: %v", err)
		}

		if len(body) > 0 {
			log.Println(string(body))
		}
	}
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
