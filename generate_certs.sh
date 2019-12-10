#!/bin/bash
# Script that generates keys based on config.json
num_clients="$(jq '.clients | length' config.json)"

echo "Generating Certificate Authority"
# First Generate Certificate Authority
cd certs
cfssl gencert -initca ca-csr.json | cfssljson -bare ca

# Now generate the client certs
# We need to generate a server and client cert for each user
echo "Generating Client Certs"
for i in $(seq 0 $(($num_clients - 1)));
do
  echo "Client: " $i
  client_name="client"$i
  # Generating the Temporary CSR for this client
  jq ".CN = \"$client_name\"" base-csr.json > $client_name.tmp.json
  address_quotes=$(jq ".clients[$i].address" ../config.json)
  address=$(echo "$address_quotes" | tr -d '"')
  echo $address
  # Generate Client Cert
  cfssl gencert \
    -ca=ca.pem \
    -ca-key=ca-key.pem \
    -config=ca-config.json \
    -profile=massl \
    $client_name.tmp.json | cfssljson -bare client$i
  # Generate Server Cert
  cfssl gencert \
    -ca=ca.pem \
    -ca-key=ca-key.pem \
    -config=ca-config.json \
    -hostname $address \
    -profile=massl \
    $client_name.tmp.json | cfssljson -bare server$i
done

