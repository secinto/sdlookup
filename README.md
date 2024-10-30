sdlookup 
==========
IP Lookups for Open Ports and Vulnerabilities from [internetdb.shodan.io](https://internetdb.shodan.io/)

This is a golang version of [nrich](https://gitlab.com/shodan-public/nrich) which support concurrency and more efficient output.
It is based on https://github.com/h4sh5/sdlookup which itself is based on https://github.com/j3ssie/sdlookup
## Install

```shell
go install github.com/secinto/sdlookup@main
```

## Usage

```shell
# Basic Usage
echo '1.2.3.4' | sdlookup -open
1.2.3.4:80
1.2.3.4:443

# lookup CIDR range
echo '1.2.3.4/24' | sdlookup -open -c 20
1.2.3.4:80
1.2.3.5:80

# get raw JSON response
echo '1.2.3.4' | sdlookup -json

```

