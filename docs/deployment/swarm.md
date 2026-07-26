# Docker Swarm

The provided stack uses one replica, an external Docker secret, named local
volumes, stop-first updates, and a node label that keeps state on one node.

Tidefetch must not be scaled horizontally against one aria2 session and one
config volume.

## Prerequisites

- An initialized Swarm
- A published GHCR image, or an equivalent image pushed to a registry every
  node can pull
- A manager shell with this repository checkout
- One labeled storage node

Initialize a single-node Swarm if needed:

```sh
docker swarm init --advertise-addr <server-lan-ip>
```

## Prepare the storage node

List nodes and attach the placement label:

```sh
docker node ls
docker node update --label-add tidefetch=true <NODE_NAME>
```

The stack's named volumes live on this node. For NAS-backed storage, replace
the volume declarations with bind mounts or a volume driver and keep the same
placement constraint.

## Create the web password secret

```sh
umask 077
openssl rand -base64 36 > tidefetch_password.txt
docker secret create tidefetch_password tidefetch_password.txt
cat tidefetch_password.txt
```

Store the displayed password in a password manager, then remove the local file:

```sh
rm tidefetch_password.txt
```

## Deploy

From the repository root:

```sh
docker stack deploy --with-registry-auth \
  -c packaging/swarm/stack.yml tidefetch
```

Inspect the service:

```sh
docker stack services tidefetch
docker service ps tidefetch_tidefetch --no-trunc
docker service logs -f --tail=100 tidefetch_tidefetch
```

Open `http://<any-swarm-node>:8210`. The HTTP port uses the routing mesh. The
BitTorrent ports use host mode and are available on the labeled task node.

## Use bind-mounted storage

Create paths on the labeled node:

```sh
sudo mkdir -p /srv/tidefetch/config /srv/downloads
sudo chown -R 1000:1000 /srv/tidefetch/config /srv/downloads
```

Replace the stack volume entries under the service:

```yaml
volumes:
  - /srv/tidefetch/config:/config
  - /srv/downloads:/downloads
```

Remove the top-level named volumes after they are no longer referenced.

## Update

Pin the desired image tag in `packaging/swarm/stack.yml`, then redeploy:

```sh
docker stack deploy --with-registry-auth \
  -c packaging/swarm/stack.yml tidefetch
```

The stack stops the existing task before starting the replacement, preventing
two aria2 processes from writing the same session.

Force a redeploy of the same tag only when necessary:

```sh
docker service update --force tidefetch_tidefetch
```

## Rotate the password

Swarm secrets are immutable. Create a versioned replacement:

```sh
umask 077
openssl rand -base64 36 > tidefetch_password_v2.txt
docker secret create tidefetch_password_v2 tidefetch_password_v2.txt
```

Change both secret references in `stack.yml` from `tidefetch_password` to
`tidefetch_password_v2`, redeploy, confirm login, then:

```sh
docker secret rm tidefetch_password
rm tidefetch_password_v2.txt
```

## Backup

Run the backup on the labeled node. Stop the service to flush the aria2
session:

```sh
docker service scale tidefetch_tidefetch=0
docker run --rm \
  -v tidefetch_tidefetch_config:/source:ro \
  -v "$PWD:/backup" \
  alpine tar -C /source -czf /backup/tidefetch-config.tar.gz .
docker service scale tidefetch_tidefetch=1
```

Confirm the real volume name with `docker volume ls`; Swarm prefixes it with
the stack name.

## Remove

```sh
docker stack rm tidefetch
```

This leaves named volumes and the external secret intact. Remove those only
after a verified backup.