# Reverse proxy and TLS

Tidefetch serves HTTP and WebSocket traffic on the same port. Proxies must
preserve the original `Host` header and support WebSocket upgrades. Tidefetch
currently expects to run at the root of a hostname; a subpath such as
`example.com/tidefetch/` is not supported.

## Security choices

Recommended:

```sh
tidefetch serve -host 127.0.0.1 -password 'a-long-unique-password'
```

This keeps Tidefetch authentication enabled behind the TLS proxy. If an
identity-aware proxy already enforces access, `-no-auth` is possible, but the
listener should remain on loopback or a private container network.

Never pass the aria2 RPC port through the public proxy.

## Caddy

Host process:

```caddyfile
tidefetch.example.com {
    encode zstd gzip
    reverse_proxy 127.0.0.1:8210
}
```

Docker network:

```caddyfile
tidefetch.example.com {
    encode zstd gzip
    reverse_proxy tidefetch:8210
}
```

Caddy provisions and renews TLS automatically when public DNS points to the
server and ports 80/443 are reachable.

## Nginx

Add the `map` in the top-level `http` context:

```nginx
map $http_upgrade $connection_upgrade {
    default upgrade;
    ''      close;
}
```

Virtual host:

```nginx
server {
    listen 443 ssl http2;
    server_name tidefetch.example.com;

    ssl_certificate     /etc/letsencrypt/live/tidefetch.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/tidefetch.example.com/privkey.pem;

    client_max_body_size 32m;

    location / {
        proxy_pass http://127.0.0.1:8210;
        proxy_http_version 1.1;
        proxy_set_header Host $http_host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;
        proxy_read_timeout 3600s;
        proxy_send_timeout 3600s;
    }
}
```

The long read timeout keeps the state WebSocket alive. The upload limit permits
`.torrent` and Metalink files without allowing unbounded request bodies.

## Traefik with Compose

Attach Tidefetch to the same external network as Traefik:

```yaml
services:
  tidefetch:
    image: ghcr.io/thre4dripper/tidefetch:latest
    environment:
      TIDEFETCH_PASSWORD_FILE: /run/secrets/web_password
    networks:
      - proxy
    labels:
      traefik.enable: "true"
      traefik.http.routers.tidefetch.rule: Host(`tidefetch.example.com`)
      traefik.http.routers.tidefetch.entrypoints: websecure
      traefik.http.routers.tidefetch.tls: "true"
      traefik.http.routers.tidefetch.tls.certresolver: letsencrypt
      traefik.http.services.tidefetch.loadbalancer.server.port: "8210"

networks:
  proxy:
    external: true
```

Traefik handles WebSocket upgrades automatically. Do not publish 8210 to the
host when Traefik is the only intended entry point.

## Kubernetes Ingress

The included manifest exposes a ClusterIP service named `tidefetch`. A generic
Ingress is:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: tidefetch
  namespace: tidefetch
spec:
  ingressClassName: nginx
  tls:
    - hosts: [tidefetch.example.com]
      secretName: tidefetch-tls
  rules:
    - host: tidefetch.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: tidefetch
                port:
                  number: 8210
```

For cert-manager, add the issuer annotation used by the cluster. Nginx Ingress
supports WebSockets without service-specific annotations on current versions.

## Tailscale Serve

For private tailnet access without public DNS:

```sh
tailscale serve --bg https / http://127.0.0.1:8210
tailscale serve status
```

Keep Tidefetch authentication enabled for defense in depth, especially when
the tailnet has many users or shared devices.

## Validation

Check HTTP and the WebSocket endpoint through the public hostname:

```sh
curl -I https://tidefetch.example.com/
curl -sS https://tidefetch.example.com/api/state
```

The state endpoint can return `401 Unauthorized` before login; that proves the
request reached Tidefetch. In browser developer tools, `/api/ws` should switch
protocols with status 101 after authentication.
