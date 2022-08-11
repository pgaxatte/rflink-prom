package main

import (
	"bufio"
	"context"
	"flag"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/subchen/go-log"
	"github.com/tarm/serial"
)

var (
	VERSION = "0.0.1"

	serialPort = flag.String("port", "/dev/ttyUSB0", "The serial device rflink is connected to.")
	serialBaud = flag.Int("baud", 57600, "The baud rate of the serial connection.")
	promAddr   = flag.String("listen", ":8080", "The address to listen on for the Prometheus HTTP endpoint.")
	mPathFlag  = flag.String("namemap", "", "Mapping file to match sensors id with a name.")
	verbose    = flag.Bool("v", false, "Increase verbosity")
	timeout    = flag.Int("timeout", 180, "Number of seconds to wait before considering a sensor has disappeared")

	// Nested map containing all the gauges. First dimension is the vendor name
	// and the id of the sensor, joined by ' '. Second dimension is the type of
	// data extracted from this sensor. Sensors can send multiple types of data
	// at once.
	//          vendor+id     type
	sensors = map[string]map[string]*Metric{}
)

var mapping *Mapping

func startPromHttpServer(wg *sync.WaitGroup) *http.Server {
	srv := &http.Server{Addr: *promAddr}

	http.Handle("/metrics", promhttp.Handler())

	go func() {
		defer wg.Done() // let main know we are done cleaning up

		// always returns error. ErrServerClosed on graceful close
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			// unexpected error. port in use?
			log.Fatalf("Error while starting Prometheus HTTTP server: %s", err)
		}
	}()

	// returning reference so caller can call Shutdown()
	return srv
}

func main() {
	flag.Parse()

	log.Default.Level = log.INFO
	if *verbose {
		log.Default.Level = log.DEBUG
	}

	log.Infof("rflink-prom v%s -- Prometheus exporter for rflink", VERSION)

	// Read mapping file if any
	mappingPath := *mPathFlag
	if mappingPath == "" {
		mappingPath = "mapping.yaml"
	}
	mapping, err := readMapping(mappingPath)
	if err != nil {
		log.Warnf("Cannot parse mapping file %s: %v. Skipping", mappingPath, err)
	}
	if mapping != nil {
		log.Infof("Using id to name mapping: %+v", mapping.IdNames)
	}

	// Setup prom HTTP server
	httpServerExitDone := &sync.WaitGroup{}
	httpServerExitDone.Add(1)
	promSrv := startPromHttpServer(httpServerExitDone)
	log.Infof("Serving prometheus metrics on %s", *promAddr)

	// Setup serial port
	port, err := serial.OpenPort(&serial.Config{
		Name: *serialPort,
		Baud: *serialBaud,
	})
	if err != nil {
		log.Fatalf("Failed to open device %s: %v", *serialPort, err)
	}
	defer port.Close()
	log.Infof("rflink connection established on %s", *serialPort)

	reader := bufio.NewReader(port)

	// Goroutine that reads rflink and creates/updates the prometheus metrics
	// accordingly
	go func() {
		for {
			line, _, err := reader.ReadLine()
			if err != nil {
				log.Errorf("Cannot read from serial: %v", err)
				// TODO: properly shutdown promhttp via WaitGroup
				// cf. https://stackoverflow.com/questions/39320025/how-to-stop-http-listenandserve
				os.Exit(1)
			}

			log.Debugf("Received from rflink: %s", line)

			err = updateMetrics(string(line), mapping)
			if err != nil {
				log.Errorf("Cannot update metrics from message: %v, skipping", err)
				continue
			}
		}
	}()

	// Goroutine that expires prometheus metrics according to the timeout
	// The loop is run every 1/4th of the timeout
	// TODO: this is a naive approach that goes through all metrics in a loop
	//       It should be replaced by a more event-like system
	go func() {
		for {
			log.Debugf("Checking all expired metrics after %d seconds of absence", *timeout)
			for _, v := range sensors {
				for _, m := range v {
					m.EnforceExpiration()
				}
			}
			time.Sleep(time.Second * time.Duration(*timeout) / 4)
		}
	}()

	// wait for goroutine started in startHttpServer() to stop
	httpServerExitDone.Wait()

	log.Infof("Stopping prometheus exporter")

	// 10 seconds timeout before forcing shutdown
	d := time.Now().Add(5)
	ctx, cancel := context.WithDeadline(context.Background(), d)
	defer cancel()

	// now close the server gracefully ("shutdown")
	if err := promSrv.Shutdown(ctx); err != nil {
		panic(err) // failure/timeout shutting down the server gracefully
	}
	log.Infof("Bye bye")
}
