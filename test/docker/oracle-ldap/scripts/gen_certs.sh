#! /bin/sh
if  ! command -v step ; then
  echo "step binary not found, pls install from https://smallstep.com/docs/step-cli/installation/"
  exit 1
fi
WD=$(dirname $0)
if [ -r $WD/ldap.env ]; then
  . $WD/ldap.env
fi
SERVER=ldap
DOMAIN=${LDAP_DOMAIN:-oracle.local}
DIR=$WD/../certs
mkdir -p $DIR
cd $DIR||exit
if [ ! -r cakey.pem ]; then
  step certificate create "Root CA" "ca.crt" "ca.key" \
    --no-password --insecure \
    --profile root-ca \
    --not-before "2021-01-01T00:00:00+00:00" \
    --not-after "2031-01-01T00:00:00+00:00" \
    --san "$DOMAIN" \
    --san "ca.$DOMAIN" \
    --kty RSA --size 2048
fi
step certificate create "${SERVER}.${DOMAIN}" "${SERVER}.${DOMAIN}.crt" "${SERVER}.${DOMAIN}.key" \
  --no-password --insecure \
  --profile leaf \
  --ca "ca.crt" \
  --ca-key "ca.key" \
  --not-before "2021-01-01T00:00:00+00:00" \
  --not-after "2031-01-01T00:00:00+00:00" \
  --san="${SERVER}.${DOMAIN}" \
  --san="${SERVER}" \
  --san="localhost" \
  --kty=RSA --size 2048

# join crt and ca
cat "${SERVER}.${DOMAIN}.crt" ca.crt >>"${SERVER}.${DOMAIN}-full.crt"
openssl x509 -in "${SERVER}.${DOMAIN}-full.crt" -noout -text