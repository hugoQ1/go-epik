# Storing Data

> There are recent bug reports with these instructions. If you happen to encounter any problems, please create a [GitHub issue](https://github.com/EpiK-Protocol/go-epik/issues/new) and a maintainer will address the problem as soon as they can.

Here are instructions for how to store data on the **epik Testnet**.

## Adding a file locally

Adding a file locally allows you to make miner deals on the **epik Testnet**.

```sh
epik client import ./your-example-file.txt
```

Upon success, this command will return a **Data CID**.

## List your local files

The command to see a list of files by `CID`, `name`, `size` in bytes, and `status`:

```sh
epik client local
```

An example of the output:

```sh
bafkreierupr5ioxn4obwly4i2a5cd2rwxqi6kwmcyyylifxjsmos7hrgpe Development/sample-1.txt 2332 ok
bafkreieuk7h4zs5alzpdyhlph4lxkefowvwdho3a3pml6j7dam5mipzaii Development/sample-2.txt 30618 ok
```

## Make a Miner Deal on epik Testnet

Get a list of all miners that can store data:

```sh
epik state list-miners
```

Get the requirements of a miner you wish to store data with:

```sh
epik client query-ask <miner>
```

Store a **Data CID** with a miner:

```sh
epik client deal <Data CID> <miner> <price> <duration>
```

Check the status of a deal:

```sh
epik client list-deals
```

- The `duration`, which represents how long the miner will keep your file hosted, is represented in blocks. Each block represents 25 seconds.

Upon success, this command will return a **Deal CID**.

The storage miner will need to **seal** the file before it can be retrieved. If the **epik Storage Miner** is not running on a machine designed for sealing, the process will take a very long time.
