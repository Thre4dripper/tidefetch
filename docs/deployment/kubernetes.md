# Kubernetes and k3s

Deploy one Tidefetch pod with persistent storage, a secret-backed password and a ClusterIP service, using either the Helm chart or the raw manifests in `packaging/kubernetes/`. Both run the pod as UID 1000 without privileges and use a Recreate strategy, because one aria2 session volume must never be written by two pods.

## Prerequisites

- Kubernetes 1.27 or newer, including k3s
- A default StorageClass, or explicit classes for the two claims
- `kubectl`, and Helm 3.8 or newer for the chart

## Helm chart

The chart is published as an OCI artifact:

```sh
helm install tidefetch oci://ghcr.io/thre4dripper/charts/tidefetch \
  --namespace tidefetch --create-namespace \
  --set auth.password='replace-this-password' \
  --set persistence.downloads.size=200Gi
```

Key values:

| Value | Default | Meaning |
| --- | --- | --- |
| `image.tag` | the chart's `appVersion` | Image tag to run |
| `auth.enabled` | `true` | Require a password; `false` adds `-no-auth` |
| `auth.password` | none | Password, stored in a generated Secret |
| `auth.existingSecret`, `auth.secretKey` | none, `password` | Use your own Secret instead |
| `persistence.config.size` | `2Gi` | Config claim |
| `persistence.downloads.size` | `100Gi` | Downloads claim; or set `persistence.downloads.existingClaim` |
| `persistence.*.storageClass` | cluster default | Storage class per claim |
| `bittorrent.enabled` | `false` | Add a `LoadBalancer` service for port 6881 TCP and UDP |
| `ingress.enabled`, `ingress.host`, `ingress.className` | `false`, `tidefetch.example.com`, `nginx` | Ingress; TLS secret via `ingress.tls.secretName` |
| `resources` | 25m CPU and 64Mi requested, 512Mi limit | Pod resources |
| `timezone` | `Etc/UTC` | `TZ` inside the container |

Upgrade and remove:

```sh
helm upgrade tidefetch oci://ghcr.io/thre4dripper/charts/tidefetch -n tidefetch --reuse-values
helm uninstall tidefetch -n tidefetch     # the claims are kept
```

## Raw manifests

`packaging/kubernetes/` holds a namespace, two claims, the Deployment and a Service, applied with Kustomize.

### Storage

Defaults are a 2 GiB config claim and a 100 GiB downloads claim, both `ReadWriteOnce`. Edit `storage.yaml` before applying, for example to choose a class:

```yaml
spec:
  storageClassName: longhorn
```

On k3s the default `local-path` class ties the data to one node. Use Longhorn, NFS CSI or another replicated class if a node failure must not strand the queue.

### Password

```sh
kubectl apply -f packaging/kubernetes/namespace.yaml
umask 077
openssl rand -base64 36 > web-password
kubectl -n tidefetch create secret generic tidefetch-web-password \
  --from-file=password=./web-password
cat web-password    # keep it in a password manager
rm web-password
```

### Deploy

```sh
kubectl apply -k packaging/kubernetes
kubectl -n tidefetch rollout status deployment/tidefetch --timeout=180s
kubectl -n tidefetch get pods,pvc,service
```

Test without an Ingress:

```sh
kubectl -n tidefetch port-forward service/tidefetch 8210:8210
```

Then open <http://127.0.0.1:8210>.

## Ingress and TLS

Copy `packaging/kubernetes/ingress.example.yaml`, set the hostname and issuer, and apply it. Tidefetch must be served at the root of the hostname, not under a path. Current ingress controllers pass its WebSocket through without extra annotations. See [Reverse proxy and TLS](../reverse-proxy.md).

## BitTorrent peer ports

The base Service exposes only the web UI. For inbound peers, add a `LoadBalancer` Service, or set `bittorrent.enabled=true` in the chart:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: tidefetch-bittorrent
  namespace: tidefetch
spec:
  type: LoadBalancer
  externalTrafficPolicy: Local
  selector:
    app.kubernetes.io/name: tidefetch
  ports:
    - name: bittorrent-tcp
      port: 6881
      targetPort: bittorrent-tcp
      protocol: TCP
    - name: bittorrent-udp
      port: 6881
      targetPort: bittorrent-udp
      protocol: UDP
```

MetalLB is the usual LoadBalancer on bare metal. Forward TCP and UDP 6881 from your router to the assigned address. Never expose aria2's RPC port 6800.

## Upgrade

Pin a release tag rather than `latest`, then apply and watch the Recreate rollout:

```yaml
image: ghcr.io/thre4dripper/tidefetch:0.1.0
```

```sh
kubectl apply -k packaging/kubernetes
kubectl -n tidefetch rollout status deployment/tidefetch
kubectl -n tidefetch logs deployment/tidefetch --tail=100
```

The brief downtime is intentional: two aria2 processes must not write the same session volume.

## Rotate the password

```sh
umask 077
openssl rand -base64 36 > web-password
kubectl -n tidefetch create secret generic tidefetch-web-password \
  --from-file=password=./web-password \
  --dry-run=client -o yaml | kubectl apply -f -
rm web-password
kubectl -n tidefetch rollout restart deployment/tidefetch
```

Browser sessions end when the pod restarts.

## Backups

Prefer CSI VolumeSnapshots. Otherwise scale to zero, copy the config claim, and scale back up:

```sh
kubectl -n tidefetch scale deployment/tidefetch --replicas=0
kubectl -n tidefetch wait --for=delete pod -l app.kubernetes.io/name=tidefetch --timeout=120s
# snapshot or copy the tidefetch-config claim here
kubectl -n tidefetch scale deployment/tidefetch --replicas=1
```

The config claim is the irreplaceable one; the downloads claim follows your normal retention policy.

## Troubleshooting

```sh
kubectl -n tidefetch describe pod -l app.kubernetes.io/name=tidefetch
kubectl -n tidefetch logs deployment/tidefetch --tail=200
kubectl -n tidefetch get events --sort-by=.lastTimestamp
kubectl -n tidefetch get pvc
```

The usual causes are an unbound claim, image pull credentials, a missing password Secret, or a storage backend that cannot apply `fsGroup` 1000.

## Uninstall

```sh
kubectl delete -k packaging/kubernetes
kubectl -n tidefetch delete secret tidefetch-web-password
```

Claims may survive depending on your storage policy. Back them up before deleting the namespace.
