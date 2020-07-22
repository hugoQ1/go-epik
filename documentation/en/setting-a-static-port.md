# Static Ports

Depending on how your network is set up, you may need to set a static port to successfully connect to peers to perform storage deals with your **epik Storage Miner**.

## Setup

To change the random **swarm port**, you may edit the `config.toml` file located under `$EPIK_STORAGE_PATH`. The default location of this file is `$HOME/.epikstorage`.

To change the port to `1347`:

```sh
[Libp2p]
  ListenAddresses = ["/ip4/0.0.0.0/tcp/1347", "/ip6/::/tcp/1347"]
```

After changing the port value, restart your **daemon**.

## Announce Addresses

If the **swarm port** is port-forwarded from another address, it is possible to control what addresses
are announced to the network.

```sh
[Libp2p]
  AnnounceAddresses = ["/ip4/<public-ip>/tcp/1347"]
```

If non-empty, this array specifies the swarm addresses to announce to the network. If empty, the daemon will announce inferred swarm addresses.

Similarly, it is possible to set `NoAnnounceAddresses` with an array of addresses to not announce to the network.

## Ubuntu's Uncomplicated Firewall

Open firewall manually:

```sh
ufw allow 1347/tcp
```

Or open and modify the profile located at `/etc/ufw/applications.d/epik-daemon`:

```sh
[epik Daemon]
title=epik Daemon
description=epik Daemon firewall rules
ports=1347/tcp
```

Then run these commands:

```sh
ufw update epik-daemon
ufw allow epik-daemon
```
