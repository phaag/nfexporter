# Nfdump Exporter

This is a prototype exporter for nfdump. It exposes metrics processed by the Prometheus monitoring system.

It's purpose is to play and experiment with nfdump netflow data and Promtheus/Grafana to build a new graphical UI as a repacement for aging NfSen.

This experimental exporter exposes counters for flows/packets and bytes per protocol (tcp/udp/icmp/other) and the source identifier from the nfcapd collector. (currently hardwired "live")

## Metrics:

```
  namespace = "nfsen"
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
```



## Usage:

```
Usage of ./nfsen_exporter:
  -UNIX socket string
    	Path for nfcapd collectors to connect (default "/tmp/nfsen.sock")
  -listen string
    	Address to listen on for telemetry (default ":9141")
  -metrics URI string
    	Path under which to expose metrics (default "/metrics")

```

The nfsen_exporter listens on a UNIX socket for statistics sent by the nfcapd collector. 

Add this to prometheus.yml:

```
  - job_name: "nfsen"

    # metrics_path defaults to '/metrics'
    # scheme defaults to 'http'.

    static_configs:
      - targets: ["localhost:9141"]
```



## Nfdump

The metric export is integrated in nfdump 1.7-beta

In order not to pollute an existing nfdump netflow installation, forward the traffic from an existing collector. Add: `-R 127.0.0.1/9999` to the argument list and setup the new collector. You may also send it to another host, which runs also Prometheus for example. 

Build nfdump 1.7-beta:

`git clone -b unicorn https://github.com/phaag/nfdump.git nfdump.unicorn` 

Build nfdump with `sh bootstrap.sh; ./configure` but do not run make install, as it would replace your existing installation. Create a tmp flow dir and run the collector from the src directory. For example:

`./nfcapd -l <tmpflows> -S2 -y -p 9999 -m <metric socket>`  

When adding `-m <metric socket>` nfcapd exports the internal statistics every 5s the the exporter. 



## Note:

Only the statistics is exposed and not the netflow recods itself.
