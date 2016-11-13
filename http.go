package main

import (
	"fmt"
	"net/http"
)

func writeMetricsHeaders(w http.ResponseWriter) {
	// Last elapsed
	fmt.Fprintf(w, "# HELP %s_response_time Last successful ping response time\n", metricsPrefix)
	fmt.Fprintf(w, "# TYPE %s_response_time gauge\n", metricsPrefix)
	// Completed
	fmt.Fprintf(w, "# HELP %s_completed Counter of successful pings\n", metricsPrefix)
	fmt.Fprintf(w, "# TYPE %s_completed counter\n", metricsPrefix)
	// Timeouts
	fmt.Fprintf(w, "# HELP %s_timeouts Counter of timeouts\n", metricsPrefix)
	fmt.Fprintf(w, "# TYPE %s_timeouts counter\n", metricsPrefix)
}

func writeHostMetrics(w http.ResponseWriter, r hostResults) {
	// Last elapsed
	fmt.Fprintf(w, "%s_response_time{host=\"%s\"} %d\n", metricsPrefix, r.host, r.lastElapsed)
	// Completed
	fmt.Fprintf(w, "%s_completed{host=\"%s\"} %d\n", metricsPrefix, r.host, r.completed)
	// Timeouts
	fmt.Fprintf(w, "%s_timeouts{host=\"%s\"} %d\n", metricsPrefix, r.host, r.timeouts)
}

func startServer(cfg Configuration, reads chan *readOp) {
	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		read := &readOp{
			resp: make(chan map[string]*hostResults)}
		reads <- read
		writeMetricsHeaders(w)
		for _, v := range <-read.resp {
			writeHostMetrics(w, *v)
		}

	})
	var listen = fmt.Sprintf(":%d", cfg.Port)
	fmt.Printf("Starting server on %s ...\n", listen)
	http.ListenAndServe(listen, nil)
}
