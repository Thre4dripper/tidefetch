# Kubernetes and k3s

The manifests deploy one Tidefetch pod with a Recreate strategy, two PVCs, a
secret-mounted password, non-root security settings, health probes, and a
ClusterIP service.

## Prerequisites

- Kubernetes 1.27 or newer, including k3s
- A default StorageClass or explicit storage classes in `storage.yaml`
- A published Tidefetch image accessible to cluster nodes
- `kubectl` and this repository checkout

## Review storage

Defaults:

- Config PVC: 2 GiB, `ReadWriteOnce`
- Downloads PVC: 100 GiB, `ReadWriteOnce`

Edit `packaging/kubernetes/storage.yaml` before applying. For a specific class:

```yaml
spec:
  storageClassName: longhorn
```

On k3s, the default `local-path` class ties data to one node. Longhorn, NFS CSI,
or another replicated class is preferable when node failure must not strand
the queue.

## Create namespace and password

```sh
kubectl apply -f packaging/kubernetes/namespace.yaml
umask 077
openssl rand -base64 36 > web-password
kubectl -n tidefetch create secret generic tidefetch-web-password \
  --from-file=password=./web-password
cat web-password
rm web-password
```

Save the displayed password in a password manager.

## Deploy

```sh
kubectl apply -k packaging/kubernetes
kubectl -n tidefetch rollout status deployment/tidefetch --timeout=180s
kubectl -n tidefetch get pods,pvc,service
```

Test without an Ingress:

```sh
kubectl -n tidefetch port-forward service/tidefetch 8210:8210
```

Open `http://127.0.0.1:8210`.

## Ingress and TLS

Copy the example, replace the hostname and issuer, then apply:

```sh
cp packaging/kubernetes/ingress.example.yaml /tmp/tidefetch-ingress.yaml
$EDITOR /tmp/tidefetch-ingress.yaml
kubectl apply -f /tmp/tidefetch-ingress.yaml
kubectl -n tidefetch get ingress
```

The application must remain at the hostname root. Current ingress controllers
support its WebSocket endpoint without special annotations. See
[Reverse proxy and TLS](../reverse-proxy.md).

## Expose BitTorrent peer ports

The base service exposes only the web UI inside the cluster. Add a
LoadBalancer service if inbound peers are required:

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

MetalLB is a common LoadBalancer implementation for bare-metal homelabs. Point
router forwarding for TCP/UDP 6881 at the assigned address. Do not expose aria2
RPC port 6800.

## Pin and update the image

Edit `deployment.yaml` to a release tag or digest:

```yaml
image: ghcr.io/thre4dripper/tidefetch:0.2.0
```

Apply and watch the Recreate rollout:

```sh
kubectl apply -k packaging/kubernetes
kubectl -n tidefetch rollout status deployment/tidefetch
kubectl -n tidefetch logs deployment/tidefetch --tail=100
```

Recreate intentionally allows downtime so two aria2 processes never mount and
write the same session volume.

## Rotate the password

```sh
umask 077
openssl rand -base64 36 > web-password
kubectl -n tidefetch create secret generic tidefetch-web-password \
  --from-file=password=./web-password \
  --dry-run=client -o yaml | kubectl apply -f -
rm web-password
kubectl -n tidefetch rollout restart deployment/tidefetch
kubectl -n tidefetch rollout status deployment/tidefetch
```

Existing browser sessions end when the pod restarts.

## Backups

Prefer CSI VolumeSnapshots when the storage driver supports them. Otherwise,
scale down and back up the mounted PVC from a temporary pod or the storage
backend:

```sh
kubectl -n tidefetch scale deployment/tidefetch --replicas=0
kubectl -n tidefetch wait --for=delete pod -l app.kubernetes.io/name=tidefetch --timeout=120s
```

After the storage snapshot or copy:

```sh
kubectl -n tidefetch scale deployment/tidefetch --replicas=1
kubectl -n tidefetch rollout status deployment/tidefetch
```

Back up the config PVC at minimum. The downloads PVC follows the normal data
retention policy for the cluster.

## Troubleshooting

```sh
kubectl -n tidefetch describe pod -l app.kubernetes.io/name=tidefetch
kubectl -n tidefetch logs deployment/tidefetch --tail=200
kubectl -n tidefetch get events --sort-by=.lastTimestamp
kubectl -n tidefetch get pvc
```

Common causes are an unbound PVC, image-pull credentials, a missing password
secret, or a storage backend that cannot set ownership for fsGroup 1000.

## Uninstall

```sh
kubectl delete -k packaging/kubernetes
kubectl -n tidefetch delete secret tidefetch-web-password
```

PVCs may remain depending on deletion order and storage policy. Confirm and
back them up before deleting the namespace.
