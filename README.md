# network-health-exporter
Monitor your or others network health.
network-health-exporter is similar to a tool like smokeping but export the results to prometheus.

Metrics available :
* network_health_response_time
* network_health_completed
* network_health_timeouts

## build
Install go & glide

Run ```make```

## Running the exporter
The exporter will load the conf.json file specified by : ```--config```

The config file specifies a list of targets to ping regularly

Example of config :

```json
{"Targets":["8.8.8.8","s3.amazon.com"],
 "IntervalSeconds":5,
 "TimeoutSeconds":10,
 "Port":9106}
```
