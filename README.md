# ReliableBroadcast

Implementation of Bracha Broadcast algorithm in Go


## Setup + Running

You need cfssl (https://github.com/cloudflare/cfssl) which this project uses to manage keys.
Do the following to install it:

```
go get -u github.com/cloudflare/cfssl/cmd/...
```

Then specify the configuration of clients in `config.json`. There should be `3f + 1` clients, where
`f` is the number of byzantine nodes supported

To run the demo (which will build everything then run each client in a tmux window) run 
```
make run
```

To do things manually, run `make` -- it will generate all the required keys for you and build the binary

Then run:

```
client -config config.json -id $CLI_ID
```

This will run a particular client


