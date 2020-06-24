# ReliableBroadcast

This is a demo reliable broadcast channel. The demo is for chatting -- each user can broadcast messages to all the other
users, and if any of the users get the message all non byzantine nodes will also have recieved that message.

## What is this?

A Reliable Broadcast channel ensures that all honest parties deliver the same broadcasted message or none at all.
A Reliable Broadcast channel will work in the face of `t` byzantine faults as long as `t < n / 3`. Byzantine faults
are when nodes can lie or go down. This is a networking primitive that is required for many MPC applications, such as
threshold signatures schemes

Specifically, this project is an implementation of the Bracha Broadcast algorithm in Go. 

## Network Topography

This system is a fully connected graph. All nodes maintain mTLS connections with all other nodes, if the nodes
are up. The system supports nodes leaving and coming back in, but if `t < n / 3` the system will not make progress
and messages during that time won't be delivered.

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


