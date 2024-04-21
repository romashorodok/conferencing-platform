#!/bin/bash

SSL_CERTIFICATE=
SSL_CERTIFICATE_KEY=

# unset DOMAIN
if [ "$DOMAIN" = "localhost" ] || [ -z "$DOMAIN" ]; then
    openssl genrsa -des3 -passout pass:x -out key.pem 2048
    cp key.pem key.pem.orig

    openssl rsa -passin pass:x -in key.pem.orig -out key.pem
    openssl req -new -key key.pem -out cert.csr -subj "/CN=localhost"
    openssl x509 -req -days 3650 -in cert.csr -signkey key.pem -out cert.pem

    SSL_CERTIFICATE=$(pwd)/cert.pem
    SSL_CERTIFICATE_KEY=$(pwd)/key.pem
else
    echo "TODO: create certs by certboot"
fi;

export SSL_CERTIFICATE SSL_CERTIFICATE_KEY
