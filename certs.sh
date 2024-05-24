#!/bin/bash

SSL_CERTIFICATE=
SSL_CERTIFICATE_KEY=

CERT_DIR="/etc/letsencrypt/live/$DOMAIN"

FULL_CHAIN="$CERT_DIR/fullchain.pem"
PRIV_KEY="$CERT_DIR/privkey.pem"

if [ "$CERT_DIR_RESET" = "true" ]; then
    rm -rf "$CERT_DIR"
    mkdir -p "$CERT_DIR"
    touch "$FULL_CHAIN"
    touch "$PRIV_KEY"

    echo "$BASE64_FULL_CHAIN" | base64 -d > $FULL_CHAIN
    echo "$BASE64_PRIV_KEY" | base64 -d > $PRIV_KEY

    SSL_CERTIFICATE=$FULL_CHAIN
    SSL_CERTIFICATE_KEY=$PRIV_KEY
fi

if [ -f "$FULL_CHAIN" ] && [ -f "$PRIV_KEY" ]; then
    echo "KEY ALREDY exist"

    SSL_CERTIFICATE=$FULL_CHAIN
    SSL_CERTIFICATE_KEY=$PRIV_KEY
elif [ -n "$BASE64_FULL_CHAIN" ] && [ -n "$BASE64_PRIV_KEY" ] && [ -n "$DOMAIN" ]; then
    echo "GENERATE KEY"

    echo "$BASE64_FULL_CHAIN" | base64 -d > $FULL_CHAIN
    echo "$BASE64_PRIV_KEY" | base64 -d > $PRIV_KEY

    SSL_CERTIFICATE=$FULL_CHAIN
    SSL_CERTIFICATE_KEY=$PRIV_KEY
else
    if [ "$DOMAIN" = "localhost" ] || [ -z "$DOMAIN" ]; then
        openssl genrsa -des3 -passout pass:x -out key.pem 2048
        cp key.pem key.pem.orig

        openssl rsa -passin pass:x -in key.pem.orig -out key.pem
        openssl req -new -key key.pem -out cert.csr -subj "/CN=localhost"
        openssl x509 -req -days 3650 -in cert.csr -signkey key.pem -out cert.pem

        SSL_CERTIFICATE=$(pwd)/cert.pem
        SSL_CERTIFICATE_KEY=$(pwd)/key.pem
    else
        certbot certonly -v --standalone -d $DOMAIN --register-unsafely-without-email --agree-tos

        SSL_CERTIFICATE=/etc/letsencrypt/live/$DOMAIN/fullchain.pem
        SSL_CERTIFICATE_KEY=/etc/letsencrypt/live/$DOMAIN/privkey.pem
    fi;
fi;


cat $SSL_CERTIFICATE
echo "[base64] Using FULL_CHAIN:"
cat $SSL_CERTIFICATE | base64
echo ""

cat $SSL_CERTIFICATE_KEY
echo "[base64] Using PRIV_KEY:"
cat $SSL_CERTIFICATE_KEY | base64
echo ""

export SSL_CERTIFICATE SSL_CERTIFICATE_KEY
