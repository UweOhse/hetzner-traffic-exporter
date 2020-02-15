# hetzner-traffic-exporter

This program talks to the [Hetzner API](https://robot.your-server.de/doc/webservice/de.html#storage-box) 
and outputs traffic statistics for all servers and ip addresses found.

## Installation
```
make install
```
will compile the program and install it into /usr/local/bin

## Usage

Pass your hetzner API username and password as environment variables: `HETZNER_USER` and `HETZNER_PASS`

### testing
```
HETZNER_USER=myusername \
HETZNER_PASSWORD=mypassword \
./hetzner-traffic-exporter -1
```
one-shot mode questions the API once, outputs the metrics, and exits.

### real
```
HETZNER_USER=myusername \
HETZNER_PASSWORD=mypassword \
./hetzner-traffic-exporter
```
This starts the program in daemon mode, listing on the API socket.

### systemd
`hetzner-traffic-exporter.service` is an example systemd service file. You need to change the username and password, before you install it.

This is just an example. I never tried it, as i do not use the horrible mess called systemd on the server. Please don't bother to discuss systemds merits with me, i don't care.


## Options

* -1 
  oneshot mode: outputs metrics once to stdout, and exists.
* --web.listen-address=1.2.3.4:56
  address of the listener socket to use. defaults to something sensible.
* --version -- 



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

## Security
Well, if i can get you to install this without doing at least a short code audit, you do not need to worry about security. believe me: if you install software just because you found it on github your computers security depends on luck alone.

So, you did the audit? Yeah, my variable naming sucks.

Now continue, and audit the 20 or so packages imported directly or indirectly.

That's too much work? Well, yes. I thought the same and didn't audit them, too. Possibly nobody ever will bother to do that. Maybe i should bite the bullet and not use the prometheus libraries, but right now i fail to find the motivation for that.

Computer security in 2020 sucks.

