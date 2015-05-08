#! /bin/sh

# usage: ./<this-script-name> > file

set -e

openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 1826 -nodes
echo "ssl_key: |"
sed 's/^/  /' key.pem
echo "ssl_certificate: |"
sed 's/^/  /' cert.pem
rm -rf key.pem cert.pem
