# ping

[![License](https://img.shields.io/github/license/invzhi/ping.svg)](LICENSE)

ICMP ping write by Go

## get code

```bash
go get github.com/invzhi/ping
```

## how to use

Build code:

```bash
go build github.com/invzhi/ping
```

Linux should set system setting:

```bash
sudo sysctl -w net.ipv4.ping_group_range="0 2147483647"
```

Run it with `sudo`:

```bash
sudo ./ping example.com
```