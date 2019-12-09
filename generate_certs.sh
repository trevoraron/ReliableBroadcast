#!/bin/bash
# Script that generates keys based on config.json
# Usage: ./generate_certs.sh $path_to_config.json
echo "Using file: " $1

num_clients="$(jq '.clients | length' config.json)"

echo "Generating Certificate Authority"
# First Generate Certificate Authority
cd certs
cfssl gencert -initca ca-csr.json | cfssljson -bare ca

# Now generate the clients
# We want things to start at zero so
echo "Generating Clients"
for i in $(seq 0 $(($num_clients - 1)));
do
  echo "Client: " $i
  cfssl gencert \
    -ca=ca.pem \
    -ca-key=ca-key.pem \
    -config=ca-config.json \
    -profile=massl \
    client-csr.json | cfssljson -bare client$i
done

