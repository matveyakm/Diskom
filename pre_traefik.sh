mkdir -p certs
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout server.key \
  -out server.crt \
  -subj "/CN=discom.spbgu.90.188.89.63.nip.io" \
  -addext "subjectAltName = DNS:discom.spbgu.90.188.89.63.nip.io,DNS:discom.spbgu.localhost,IP:90.188.89.63"

htpasswd -nb traefik_admin "8YILv8k99c#{2" | sed -e 's/\$/\$\$/g'
