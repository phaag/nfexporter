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
 *  
 */

/*
 * dataSocket implements a UNIX socket server to receive data from nfcapd
 * Up to now the exporter implements flows/packets/bytes counters per
 * protocol(tcp/udp/icmp/other and the source identifier from the collector
 *
 */

package main

/*

#include <stdint.h>

typedef struct metric_record_s {
	// Ident
	uint64_t	exporterID; // 32bit: exporter_id:16 engineType:8 engineID:*

	// flow stat
	uint64_t numflows_tcp;
	uint64_t numflows_udp;
	uint64_t numflows_icmp;
	uint64_t numflows_other;
	// bytes stat
	uint64_t numbytes_tcp;
	uint64_t numbytes_udp;
	uint64_t numbytes_icmp;
	uint64_t numbytes_other;
	// packet stat
	uint64_t numpackets_tcp;
	uint64_t numpackets_udp;
	uint64_t numpackets_icmp;
	uint64_t numpackets_other;
} metric_record_t;

const int record_size = sizeof(metric_record_t);
*/
import "C"

import (
	"encoding/binary"
	"fmt"
	"os"
	"log"
	"net"
	"unsafe"
)

const packetPrefix byte = '@'

const metricSize int = C.record_size

type nfsenMetric struct {
	//  exporter ID
	exporterID uint64
    // flow stat
    numFlows_tcp uint64
    numFlows_udp uint64
    numFlows_icmp uint64
    numFlows_other uint64
    // bytes stat
    numBytes_tcp uint64
    numBytes_udp uint64
    numBytes_icmp uint64
    numBytes_other uint64
    // packet stat
    numPackets_tcp uint64
    numPackets_udp uint64
    numPackets_icmp uint64
    numPackets_other uint64
}

var metricList map[string]map[uint64]nfsenMetric

type socketConf struct {
	socketPath	string
	listener	net.Listener
}

func New(socketPath string) *socketConf {
    conf := new(socketConf)
    conf.socketPath = socketPath
	metricList = make(map[string]map[uint64]nfsenMetric)
    return conf
}

func (socket *socketConf) Open() error {

	if err := os.RemoveAll(socket.socketPath); err != nil {
		return err
	}
	listener, err := net.Listen("unix", socket.socketPath)
    if err != nil {
        return err
    }
	socket.listener = listener
	return nil

} // End of Open

func (socket *socketConf) Close() error {

	return socket.listener.Close()

} // End of Close

func processStat(conn net.Conn) {

    defer conn.Close()

	// storage for reading from socket.
    readBuf := make([]byte, 65536)

	dataLen, err := conn.Read(readBuf)
	if err != nil || dataLen == 0{
		fmt.Printf("Socket read error: %v\n", err)
		return
	}
	if readBuf[0] != packetPrefix {
		fmt.Printf("Message prefix error - got %u\n", readBuf[0])
		return
	}

	// version := readBuf[1]
	// payloadSize := int(binary.LittleEndian.Uint16(readBuf[2:4]))
	numMetrics	:= int(binary.LittleEndian.Uint16(readBuf[4:6]))
	// collectorID	:= int(binary.LittleEndian.Uint64(readBuf[8:16]))
	// uptime		:= int(binary.LittleEndian.Uint64(readBuf[16:24]))
	ilen := 0
	for i := 0; readBuf[24+i] != 0; i++ {
		ilen++
	}
	ident := string(readBuf[24:24+ilen])

	if _, ok := metricList[ident]; ok == false {
		metricList[ident] = make(map[uint64]nfsenMetric)
	}

/*
	fmt.Printf("Message size: %d, payload size: %d version: %d, numMetrics: %d\n",
		dataLen, payloadSize, version, numMetrics);
	fmt.Printf("Collector: %d, uptime: %d, ident: %s\n",
		collectorID, uptime, ident)
*/
	var metric nfsenMetric
	offset := 152
	for num := 0; num < numMetrics; num++ {
		var s *C.metric_record_t = (*C.metric_record_t)(unsafe.Pointer(&readBuf[offset]))
		metric.exporterID = uint64(s.exporterID)
		metric.numFlows_tcp   = uint64(s.numflows_tcp)
		metric.numFlows_udp   = uint64(s.numflows_udp)
		metric.numFlows_icmp  = uint64(s.numflows_icmp)
		metric.numFlows_other = uint64(s.numflows_other)

		metric.numBytes_tcp   = uint64(s.numbytes_tcp)
		metric.numBytes_udp   = uint64(s.numbytes_udp)
		metric.numBytes_icmp  = uint64(s.numbytes_icmp)
		metric.numBytes_other = uint64(s.numbytes_other)

		metric.numPackets_tcp   = uint64(s.numpackets_tcp)
		metric.numPackets_udp   = uint64(s.numpackets_udp)
		metric.numPackets_icmp  = uint64(s.numpackets_icmp)
		metric.numPackets_other = uint64(s.numpackets_other)

		mutex.Lock()
		metricList[ident][metric.exporterID] = metric
		mutex.Unlock()
		offset += metricSize
	}

} // end of processStat

func (socket *socketConf) Run() {

	go func() {
		for {
			// Accept new connections from nfcapd collectors and
			// dispatching them to goroutine processStat
			conn, err := socket.listener.Accept()
			if err != nil {
				log.Fatal("accept error:", err)
			}
			// fmt.Printf("New connection\n")
			go processStat(conn)
		}
	}()

} // End of Run
