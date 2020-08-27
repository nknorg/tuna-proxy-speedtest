package main

import (
	"errors"
	"time"

	nkn "github.com/nknorg/nkn-sdk-go"
	"github.com/nknorg/tuna"
)

type tunaProxy struct {
	tunaEntry *tuna.TunaEntry
}

func newTunaProxy(seed []byte, port int) (*tunaProxy, error) {
	serviceName := "httpproxy"
	maxPrice := "0.01"
	service := tuna.Service{
		Name:       serviceName,
		TCP:        []uint32{uint32(port)},
		Encryption: "xsalsa20-poly1305",
	}
	serviceInfo := tuna.ServiceInfo{
		ListenIP: "127.0.0.1",
		MaxPrice: maxPrice,
	}
	config := &tuna.EntryConfiguration{
		SubscriptionPrefix: tuna.DefaultSubscriptionPrefix,
		DialTimeout:        10,
		Services: map[string]tuna.ServiceInfo{
			serviceName: serviceInfo,
		},
	}

	account, err := nkn.NewAccount(seed)
	if err != nil {
		return nil, err
	}

	wallet, err := nkn.NewWallet(account, nil)
	if err != nil {
		return nil, err
	}

	te, err := tuna.NewTunaEntry(service, serviceInfo, wallet, config)
	if err != nil {
		return nil, err
	}

	return &tunaProxy{
		tunaEntry: te,
	}, nil
}

func (tp *tunaProxy) start() error {
	errChan := make(chan error, 1)
	go func() {
		errChan <- tp.tunaEntry.Start(true)
	}()

	select {
	case err := <-errChan:
		return err
	case <-tp.tunaEntry.OnConnect.C:
		return nil
	case <-time.After(time.Minute):
		return errors.New("tuna connect timeout")
	}
}

func (tp *tunaProxy) stop() {
	tp.tunaEntry.Close()
}
