package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	client "github.com/influxdata/influxdb/client/v2"
	"github.com/lanzafame/trend"
)

func main() {
	// configure influxdb client
	var (
		influxAddr = flag.String("influx", "https://influx.skapa.xyz", "Address of influxdb")
		influxDB   = flag.String("db", "crypto", "Name of the influxdb Database to push values to")
		username   = flag.String("user", "", "Username for the influxdb")
		password   = flag.String("pass", "", "Password for the user of influxdb")
		crypto     = flag.String("crypto", "", "Crypto currency to track, empty will track all available crypto currencies")
		convert    = flag.String("convert", "aud", "Currency to convert to, i.e. BTC<->AUD")
	)
	flag.Parse()

	c, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     *influxAddr,
		Username: *username,
		Password: *password,
	})
	if err != nil {
		log.Fatal(err)
	}

	bpconf := client.BatchPointsConfig{
		Database:  *influxDB,
		Precision: "s",
	}

	// coinmarketcap api only updates every 5 minutes
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	quit := make(chan struct{}, 1)
	done := make(chan struct{}, 1)
	ticks := make(chan trend.Ticker)
	ticker := time.NewTicker(5 * time.Minute)
	logs := make(chan string)

	// signal handling go func
	go func() {
		<-sigs
		close(quit)
	}()

	// collector go func
	if *crypto == "" {
		go collectAllCryptos(*convert, ticker, ticks, quit)
	} else {
		go collectSpecificCrypto(*crypto, *convert, ticker, ticks, quit)
	}

	// persister go func
	go func() {
		for {
			select {
			case t := <-ticks:
				// Create a new point batch
				bp, err := client.NewBatchPoints(bpconf)
				if err != nil {
					log.Fatal(err)
				}

				pt := t.MarshalInfluxdbLineProto(*convert)

				bp.AddPoint(client.NewPointFrom(pt))

				err = c.Write(bp)
				if err != nil {
					log.Printf("write: %v", err)
				} else {
					logs <- "wrote to influx\n"
				}
			case <-quit:
				done <- struct{}{}
			}
		}
	}()

	for {
		select {
		case l := <-logs:
			log.Print(l)
		case <-done:
			os.Exit(-1)
		}
	}
}

func collectSpecificCrypto(crypto, convert string, ticker *time.Ticker, ticks chan trend.Ticker, quit chan struct{}) {
	for {
		select {
		case <-ticker.C:
			t, err := trend.GetNewTick(crypto, convert)
			if err != nil {
				log.Fatal(err)
			}
			ticks <- t
		case <-quit:
			ticker.Stop()
			return
		}
	}
}

func collectAllCryptos(convert string, ticker *time.Ticker, ticks chan trend.Ticker, quit chan struct{}) {
	for {
		select {
		case <-ticker.C:
			ts, err := trend.GetNewTicks(convert)
			if err != nil {
				log.Fatal(err)
			}
			for _, t := range ts {
				ticks <- t
			}
		case <-quit:
			ticker.Stop()
			return
		}
	}
}
