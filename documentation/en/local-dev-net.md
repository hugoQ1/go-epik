# Setup Local Devnet

Build the epik Binaries in debug mode, This enables the use of 2048 byte sectors.

```sh
make 2k
```

Download the 2048 byte parameters:
```sh
./epik fetch-params 2048
```

Pre-seal some sectors:

```sh
./epik-seed pre-seal --sector-size 2KiB --num-sectors 2
```

Create the genesis block and start up the first node:

```sh
./epik-seed genesis new localnet.json
./epik-seed genesis add-miner localnet.json ~/.genesis-sectors/pre-seal-t01000.json
./epik daemon --epik-make-genesis=dev.gen --genesis-template=localnet.json --bootstrap=false
```

Then, in another console, import the genesis miner key:

```sh
./epik wallet import ~/.genesis-sectors/pre-seal-t01000.key
```

Set up the genesis miner:

```sh
./epik-storage-miner init --genesis-miner --actor=t01000 --sector-size=2KiB --pre-sealed-sectors=~/.genesis-sectors --pre-sealed-metadata=~/.genesis-sectors/pre-seal-t01000.json --nosync
```

Now, finally, start up the miner:

```sh
./epik-storage-miner run --nosync
```

If all went well, you will have your own local epik Devnet running.
