/* 
 *  Copyright (c) 2021, Peter Haag
 *  All rights reserved.
 *  
 *  Redistribution and use in source and binary forms, with or without 
 *  modification, are permitted provided that the following conditions are met:
 *  
 *   * Redistributions of source code must retain the above copyright notice, 
 *     this list of conditions and the following disclaimer.
 *   * Redistributions in binary form must reproduce the above copyright notice, 
 *     this list of conditions and the following disclaimer in the documentation 
 *     and/or other materials provided with the distribution.
 *   * Neither the name of the author nor the names of its contributors may be 
 *     used to endorse or promote products derived from this software without 
 *     specific prior written permission.
 *  
 *  THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" 
 *  AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE 
 *  IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE 
 *  ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE 
 *  LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR 
 *  CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF 
 *  SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS 
 *  INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN 
 *  CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) 
 *  ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE 
 *  POSSIBILITY OF SUCH DAMAGE.
*/

/*
 * Poc to implement a metric exporter for nfcapd collectors to Prometheus
 */

package main

import (
	"fmt"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const namespace = "nfsen"

var mutex *sync.Mutex

var (
	listenAddress = flag.String("listen", ":9141",
		"Address to listen on for telemetry")
	metricsURI = flag.String("metrics URI", "/metrics",
		"Path under which to expose metrics")
	socketPath = flag.String("UNIX socket", "/tmp/nfsen.sock",
		"Path for nfcapd collectors to connect")
)

var (

	// Metrics
	uptime = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "collector", "uptime"),
		"nfsen uptime.",
		[]string{"version"}, nil,
	)
	flowsReceived = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "collector", "flows"),
		"How many flows have been received (per ident and protocol (tcp/udp/icmp/other)).",
		[]string{"ident", "proto"}, nil,
	)
	packetsReceived = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "collector", "packets"),
		"How many packets have been received (per ident and protocol) (tcp/udp/icmp/other).",
		[]string{"ident", "proto"}, nil,
	)
	bytesReceived = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "collector", "bytes"),
		"How many bytes have been received (per ident and protocol) (tcp/udp/icmp/other).",
		[]string{"ident", "proto"}, nil,
	)
)

type Exporter struct {

}

func NewExporter() *Exporter {
	return &Exporter{

	}
} // End of NewExporter

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- uptime
	ch <- flowsReceived
	ch <- packetsReceived
	ch <- bytesReceived
} // End of Describe

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	/*
	fmt.Printf("Ident     : %s\n", metric.ident)
	fmt.Printf("Uptime    : %d\n", metric.uptime)
	fmt.Printf("Flows tcp : %d\n", metric.numFlows_tcp)
	fmt.Printf("Flows udp : %d\n", metric.numFlows_udp)
	fmt.Printf("Flows icmp : %d\n", metric.numFlows_icmp)
	fmt.Printf("Flows other : %d\n", metric.numFlows_other)
	fmt.Printf("Bytes tcp : %d\n", metric.numBytes_tcp)
	fmt.Printf("Bytes udp : %d\n", metric.numBytes_udp)
	fmt.Printf("Bytes icmp : %d\n", metric.numBytes_icmp)
	fmt.Printf("Bytes other : %d\n", metric.numBytes_other)
	fmt.Printf("Packets tcp : %d\n", metric.numPackets_tcp)
	fmt.Printf("Packets udp : %d\n", metric.numPackets_udp)
	fmt.Printf("Packets icmp : %d\n", metric.numPackets_icmp)
	fmt.Printf("Packets other : %d\n", metric.numPackets_other)
	*/

	mutex.Lock()
	ch <- prometheus.MustNewConstMetric(uptime, prometheus.CounterValue, float64(metric.uptime), "v1.7-beta")
	ch <- prometheus.MustNewConstMetric(flowsReceived, prometheus.CounterValue, float64(metric.numFlows_tcp), metric.ident, "tcp")
	ch <- prometheus.MustNewConstMetric(flowsReceived, prometheus.CounterValue, float64(metric.numFlows_udp), metric.ident, "udp")
	ch <- prometheus.MustNewConstMetric(flowsReceived, prometheus.CounterValue, float64(metric.numFlows_icmp), metric.ident, "icmp")
	ch <- prometheus.MustNewConstMetric(flowsReceived, prometheus.CounterValue, float64(metric.numFlows_other), metric.ident, "other")

	// packets
	ch <- prometheus.MustNewConstMetric(packetsReceived, prometheus.CounterValue, float64(metric.numPackets_tcp), metric.ident, "tcp")
	ch <- prometheus.MustNewConstMetric(packetsReceived, prometheus.CounterValue, float64(metric.numPackets_udp), metric.ident, "udp")
	ch <- prometheus.MustNewConstMetric(packetsReceived, prometheus.CounterValue, float64(metric.numPackets_icmp), metric.ident, "icmp")
	ch <- prometheus.MustNewConstMetric(packetsReceived, prometheus.CounterValue, float64(metric.numPackets_other), metric.ident, "other")

	// bytes
	ch <- prometheus.MustNewConstMetric(bytesReceived, prometheus.CounterValue, float64(metric.numBytes_tcp), metric.ident, "tcp")
	ch <- prometheus.MustNewConstMetric(bytesReceived, prometheus.CounterValue, float64(metric.numBytes_udp), metric.ident, "udp")
	ch <- prometheus.MustNewConstMetric(bytesReceived, prometheus.CounterValue, float64(metric.numPackets_icmp), metric.ident, "icmp")
	ch <- prometheus.MustNewConstMetric(bytesReceived, prometheus.CounterValue, float64(metric.numPackets_other), metric.ident, "other")
	metric = nfsenMetric{}
	mutex.Unlock()

} // End of Collect

// cleanup on signal TERM/cntrl-C
func SetupCloseHandler(socketHandler *socketConf) {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Printf("Exit exporter\n")
		socketHandler.Close()
		os.Remove(*socketPath)
		os.Exit(0)
	}()
}

func main() {

	flag.Parse()

	exporter := NewExporter()
	prometheus.MustRegister(exporter)

	mutex = new(sync.Mutex)

	socketHandler := New(*socketPath)
	if err := socketHandler.Open(); err != nil {
		log.Fatal("Socket handler failed: ", err)
	}
	SetupCloseHandler(socketHandler)

	socketHandler.Run()

	http.Handle(*metricsURI, promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
             <head><title>NfSen Metric Exporter</title></head>
             <body>
             <h1>NfSen Metric Exporter</h1>
             <p><a href='` + *metricsURI + `'>Metrics</a></p>
             </body>
             </html>`))
	})
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}
