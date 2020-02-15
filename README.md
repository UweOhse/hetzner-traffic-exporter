# hetzner-traffic-exporter

This program talks to the [Hetzner API](https://robot.your-server.de/doc/webservice/en.html) 
and outputs traffic statistics for all servers and ip addresses found.

Note that Hetzner updates the traffic data hourly. If you use the exported numbers for dashboards, do not use, for example, `rate(sum(hetzner_traffic_output_gb[5m]))`, but something like 65m or 75m.

## License

[GPLv2](https://www.ohse.de/uwe/licenses/GPL-2)

## Installation
```
make install
```
will compile the program and install it into /usr/local/bin . You need a working go installation for it.

## Usage

Pass your hetzner API username and password as environment variables: `HETZNER_USER` and `HETZNER_PASS`

### testing
```
HETZNER_USER=yourusername \
HETZNER_PASSWORD=yourpassword \
./hetzner-traffic-exporter -1
```
one-shot mode questions the API once, outputs the metrics, and exits.

### real
```
HETZNER_USER=yourusername \
HETZNER_PASSWORD=yourpassword \
./hetzner-traffic-exporter
```
This starts the program in daemon mode, listing on the API socket.

### systemd
`hetzner-traffic-exporter.service` is an example systemd service file. You need to change the username and password, before you install it.

This is just an example. I never tried it, and hope i never have to.

## Options

* -1 
  oneshot mode: outputs metrics once to stdout, and exists.
* -listen=1.2.3.4:56
  address of the listener socket to use. defaults to something sensible.
* -interval=123
  set the update interval to 123 minutes. By default this exporter calls the hetzner API every 10 minutes, which should be enough in most cases.
* -version
  does what you should expect.
* -license
  shows license information.
* -help
  shows help.


## Exported Metrics 
```
hetzner_traffic_input_gb...
hetzner_traffic_output_gb...
hetzner_traffic_total_gb{address="2a01:4f8:13b:3b0c::/64",dns_name="",product="AX60-SSD",server_name="x7.ohse.de",server_number="917021"} 0.1609
hetzner_traffic_total_gb{address="94.130.128.194",dns_name="oldonx7.naturfotografen-forum.de",product="AX60-SSD",server_name="x7.ohse.de",server_number="917021"} 373.9031
hetzner_traffic_total_gb{address="94.130.128.243",dns_name="x7.ohse.de",product="AX60-SSD",server_name="x7.ohse.de",server_number="917021"} 3.7023
```

In words: The exporter prints three set of metrics `hetzner_traffic_input_gb`, `hetzner_traffic_output_gb` and `hetzner_traffic_total_gb`. Only the last one is shown above, it is the sum of the first two, unless rounding errors happen.

`address` is the address the value at the end of each line has been collected for. It can be an IPv4 address, an IPv4 net or an IPv6 net.

`dns_name` is the reverse mapping of `address`, as given by the hetzner API. It will be empty if no reverse mapping has been entered.

`server_name` is the name of the server in the hetzner API. It can be empty if no name is set.

`product` ist the product name at hetzner.

`server_number` is the server number. This might be useful to aggregate by if `server_name` is not set.

## Rate limits

Hetzner limits the API used to 200 calls per hour (and API end point). This exporter by default updates the data every 10 minutes, so you still should have 194 calls left, which should be plenty unless you use the API extensively.

You can reduce the interval to one minute, or increase it to 60 minutes. Do not increase it over one hour, as this means you will lose data of the hours before midnight.

## Security
Well, if i can get you to install this without doing at least a short code audit, you do not need to worry about security. believe me: if you install software just because you found it on github your computers security depends on luck alone.

So, you did the audit? Yeah, my variable naming sucks.

Now continue, and audit the 20 or so packages imported directly or indirectly.

That's too much work? Well, yes. I thought the same and didn't audit them, too. Possibly nobody ever will bother to do that. Maybe i should bite the bullet and not use the prometheus libraries, but right now i fail to find the motivation for that.

Computer security in 2020 sucks.
