# Trend

Trend is a tool to collect data from coinmarketcap.com API and aggregate it in an Influxdb instance.

## Setup

Start `influxdb`:

```bash
docker run -d -p 8086:8086 \                                                                                                                                                                                                           13:56
        -v $PWD/influxdb.conf:/etc/influxdb/influxdb.conf:ro \
        -v crypto:/var/lib/influxdb/data \
        influxdb -config /etc/influxdb/influxdb.conf
```




