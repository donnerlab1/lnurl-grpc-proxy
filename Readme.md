# lnurl-grpc-proxy

A rest -> grpc proxy for lnurl withdraw requests

This is meant to be used by clients behind firewalls (Bitcoin Bounty Hunt node e.g.)
## usage

```lnurl-grpc-proxy --grpc_port 10512 --base_url "http://localhost:10513" --http_host "localhost:10513"```
