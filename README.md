# Introduction

Tool to generate baseline values and top X talkers for all hostgroups in FastNetMon Advanced and then store them in MongoDB. It uses data from traffic metrics in Clickhouse and it can read Clickhouse configuration and hostgroups directly from MongoDB.

# Build process

```go build```

# Configuration

Create file /etc/fastnetmon/baseline_exporter.conf and put following content into it:

```
{
  "calculaton_period": 604799,
  "number_of_top_talkers": 100, 
  "log_level": "info"
}
```


# Run

```
sudo ./baseline_exporter
```

# Expect following data MongoDB collection named baseline_exporter_hostgroups_baseline

```
{ "_id" : ObjectId("624ede05ce548cbb2b711388"), "name" : "global", "incoming" : { "packets" : { "quantile_95" : NumberLong(334) }, "bits" : { "quantile_95" : NumberLong(67849921) }, "flows" : { "quantile_95" : NumberLong(0) }, "tcp_packets" : { "quantile_95" : NumberLong(334) }, "udp_packets" : { "quantile_95" : NumberLong(0) }, "icmp_packets" : { "quantile_95" : NumberLong(0) }, "fragmented_packets" : { "quantile_95" : NumberLong(0) }, "tcp_syn_packets" : { "quantile_95" : NumberLong(0) }, "tcp_bits" : { "quantile_95" : NumberLong(67849921) }, "udp_bits" : { "quantile_95" : NumberLong(0) }, "icmp_bits" : { "quantile_95" : NumberLong(0) }, "fragmented_bits" : { "quantile_95" : NumberLong(0) }, "tcp_syn_bits" : { "quantile_95" : NumberLong(0) } }, "outgoing" : { "packets" : { "quantile_95" : NumberLong(331) }, "bits" : { "quantile_95" : NumberLong(176284) }, "flows" : { "quantile_95" : NumberLong(0) }, "tcp_packets" : { "quantile_95" : NumberLong(331) }, "udp_packets" : { "quantile_95" : NumberLong(0) }, "icmp_packets" : { "quantile_95" : NumberLong(0) }, "fragmented_packets" : { "quantile_95" : NumberLong(0) }, "tcp_syn_packets" : { "quantile_95" : NumberLong(0) }, "tcp_bits" : { "quantile_95" : NumberLong(176284) }, "udp_bits" : { "quantile_95" : NumberLong(0) }, "icmp_bits" : { "quantile_95" : NumberLong(0) }, "fragmented_bits" : { "quantile_95" : NumberLong(0) }, "tcp_syn_bits" : { "quantile_95" : NumberLong(0) } } }
{ "_id" : ObjectId("624ede26ce548cbb2b7113cb"), "name" : "my_new_group", "incoming" : { "packets" : { "quantile_95" : NumberLong(334) }, "bits" : { "quantile_95" : NumberLong(67849921) }, "flows" : { "quantile_95" : NumberLong(0) }, "tcp_packets" : { "quantile_95" : NumberLong(334) }, "udp_packets" : { "quantile_95" : NumberLong(0) }, "icmp_packets" : { "quantile_95" : NumberLong(0) }, "fragmented_packets" : { "quantile_95" : NumberLong(0) }, "tcp_syn_packets" : { "quantile_95" : NumberLong(0) }, "tcp_bits" : { "quantile_95" : NumberLong(67849921) }, "udp_bits" : { "quantile_95" : NumberLong(0) }, "icmp_bits" : { "quantile_95" : NumberLong(0) }, "fragmented_bits" : { "quantile_95" : NumberLong(0) }, "tcp_syn_bits" : { "quantile_95" : NumberLong(0) } }, "outgoing" : { "packets" : { "quantile_95" : NumberLong(331) }, "bits" : { "quantile_95" : NumberLong(176284) }, "flows" : { "quantile_95" : NumberLong(0) }, "tcp_packets" : { "quantile_95" : NumberLong(331) }, "udp_packets" : { "quantile_95" : NumberLong(0) }, "icmp_packets" : { "quantile_95" : NumberLong(0) }, "fragmented_packets" : { "quantile_95" : NumberLong(0) }, "tcp_syn_packets" : { "quantile_95" : NumberLong(0) }, "tcp_bits" : { "quantile_95" : NumberLong(176284) }, "udp_bits" : { "quantile_95" : NumberLong(0) }, "icmp_bits" : { "quantile_95" : NumberLong(0) }, "fragmented_bits" : { "quantile_95" : NumberLong(0) }, "tcp_syn_bits" : { "quantile_95" : NumberLong(0) } } }
```

# Expect following data MongoDB collection named baseline_exporter_hostgroups_top_talkers

```
{ "_id" : ObjectId("624edc85ce548cbb2b7112c8"), "name" : "global", "incoming" : { "packets" : [ { "host" : "10.18.62.249", "value" : NumberLong(361) } ], "bits" : [ { "host" : "10.18.62.249", "value" : NumberLong(94801440) } ], "flows" : [ { "host" : "10.18.62.249", "value" : NumberLong(0) } ], "tcp_packets" : [ { "host" : "10.18.62.249", "value" : NumberLong(361) } ], "udp_packets" : [ { "host" : "10.18.62.249", "value" : NumberLong(0) } ], "icmp_packets" : [ { "host" : "10.18.62.249", "value" : NumberLong(0) } ], "fragmented_packets" : [ { "host" : "10.18.62.249", "value" : NumberLong(0) } ], "tcp_syn_packets" : [ { "host" : "10.18.62.249", "value" : NumberLong(0) } ], "tcp_bits" : [ { "host" : "10.18.62.249", "value" : NumberLong(94801440) } ], "udp_bits" : [ { "host" : "10.18.62.249", "value" : NumberLong(0) } ], "icmp_bits" : [ { "host" : "10.18.62.249", "value" : NumberLong(0) } ], "fragmented_bits" : [ { "host" : "10.18.62.249", "value" : NumberLong(0) } ], "tcp_syn_bits" : [ { "host" : "10.18.62.249", "value" : NumberLong(0) } ] }, "outgoing" : { "packets" : [ { "host" : "10.18.62.249", "value" : NumberLong(354) } ], "bits" : [ { "host" : "10.18.62.249", "value" : NumberLong(187952) } ], "flows" : [ { "host" : "10.18.62.249", "value" : NumberLong(0) } ], "tcp_packets" : [ { "host" : "10.18.62.249", "value" : NumberLong(354) } ], "udp_packets" : [ { "host" : "10.18.62.249", "value" : NumberLong(0) } ], "icmp_packets" : [ { "host" : "10.18.62.249", "value" : NumberLong(0) } ], "fragmented_packets" : [ { "host" : "10.18.62.249", "value" : NumberLong(0) } ], "tcp_syn_packets" : [ { "host" : "10.18.62.249", "value" : NumberLong(0) } ], "tcp_bits" : [ { "host" : "10.18.62.249", "value" : NumberLong(187952) } ], "udp_bits" : [ { "host" : "10.18.62.249", "value" : NumberLong(0) } ], "icmp_bits" : [ { "host" : "10.18.62.249", "value" : NumberLong(0) } ], "fragmented_bits" : [ { "host" : "10.18.62.249", "value" : NumberLong(0) } ], "tcp_syn_bits" : [ { "host" : "10.18.62.249", "value" : NumberLong(0) } ] } }
{ "_id" : ObjectId("624edc85ce548cbb2b7112ca"), "name" : "my_new_group", "incoming" : { "packets" : [ { "host" : "10.18.62.249", "value" : NumberLong(361) } ], "bits" : [ { "host" : "10.18.62.249", "value" : NumberLong(94801440) } ], "flows" : [ { "host" : "10.18.62.249", "value" : NumberLong(0) } ], "tcp_packets" : [ { "host" : "10.18.62.249", "value" : NumberLong(361) } ], "udp_packets" : [ { "host" : "10.18.62.249", "value" : NumberLong(0) } ], "icmp_packets" : [ { "host" : "10.18.62.249", "value" : NumberLong(0) } ], "fragmented_packets" : [ { "host" : "10.18.62.249", "value" : NumberLong(0) } ], "tcp_syn_packets" : [ { "host" : "10.18.62.249", "value" : NumberLong(0) } ], "tcp_bits" : [ { "host" : "10.18.62.249", "value" : NumberLong(94801440) } ], "udp_bits" : [ { "host" : "10.18.62.249", "value" : NumberLong(0) } ], "icmp_bits" : [ { "host" : "10.18.62.249", "value" : NumberLong(0) } ], "fragmented_bits" : [ { "host" : "10.18.62.249", "value" : NumberLong(0) } ], "tcp_syn_bits" : [ { "host" : "10.18.62.249", "value" : NumberLong(0) } ] }, "outgoing" : { "packets" : [ { "host" : "10.18.62.249", "value" : NumberLong(354) } ], "bits" : [ { "host" : "10.18.62.249", "value" : NumberLong(187952) } ], "flows" : [ { "host" : "10.18.62.249", "value" : NumberLong(0) } ], "tcp_packets" : [ { "host" : "10.18.62.249", "value" : NumberLong(354) } ], "udp_packets" : [ { "host" : "10.18.62.249", "value" : NumberLong(0) } ], "icmp_packets" : [ { "host" : "10.18.62.249", "value" : NumberLong(0) } ], "fragmented_packets" : [ { "host" : "10.18.62.249", "value" : NumberLong(0) } ], "tcp_syn_packets" : [ { "host" : "10.18.62.249", "value" : NumberLong(0) } ], "tcp_bits" : [ { "host" : "10.18.62.249", "value" : NumberLong(187952) } ], "udp_bits" : [ { "host" : "10.18.62.249", "value" : NumberLong(0) } ], "icmp_bits" : [ { "host" : "10.18.62.249", "value" : NumberLong(0) } ], "fragmented_bits" : [ { "host" : "10.18.62.249", "value" : NumberLong(0) } ], "tcp_syn_bits" : [ { "host" : "10.18.62.249", "value" : NumberLong(0) } ] } }
```

# Example log output:

```2022/04/07 13:50:50 Started Baseline exporter
2022/04/07 13:50:50 We have no configuration file /etc/fastnetmon/baseline_exporter.conf, start with default options
2022/04/07 13:50:50 Baseline exporter configuration: {CalculationPeriod:604800 AggregationFunction:quantile(0.95) NumberOfTopTalkers:100 LogLevel:info}
2022/04/07 13:50:50 Successfully connected to MongoDB and executed PING query successfully
2022/04/07 13:50:50 Successfully read main configuration of FastNetMon from MongoDB
2022/04/07 13:50:50 Preparing to read all hostgroups
2022/04/07 13:50:50 Loaded 2 hostgroups
2022/04/07 13:50:50 Hostgroup global loaded with networks 
2022/04/07 13:50:50 Hostgroup my_new_group loaded with networks 11.22.33.44/24,2a03:2880:f158:181:face:b00c:0:25de/64,192.168.8.106/24,10.18.62.249/24
2022/04/07 13:50:50 Trying to connect to Clickhouse on 127.0.0.1:9000
2022/04/07 13:50:50 Successfully connected to Clickhouse
2022/04/07 13:50:50 Start baseline generation for global
2022/04/07 13:50:50 Retrieve traffic metrics for hostgroup global
2022/04/07 13:50:50 Updated baseline in MongoDB for global
2022/04/07 13:50:50 Start baseline generation for my_new_group
2022/04/07 13:50:50 Retrieve traffic metrics for hostgroup my_new_group
2022/04/07 13:50:50 Updated baseline in MongoDB for my_new_group
2022/04/07 13:50:50 Start top talkers generation for global
2022/04/07 13:50:50 Updated top talkers in MongoDB for global
2022/04/07 13:50:50 Start top talkers generation for my_new_group
2022/04/07 13:50:50 Updated top talkers in MongoDB for my_new_group
```
