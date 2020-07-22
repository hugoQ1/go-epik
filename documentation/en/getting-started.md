# epik

epik is an implementation of the **Filecoin Distributed Storage Network**. You can run the epik software client to join the **Filecoin Testnet**.

For more details about Filecoin, check out the [Filecoin Docs](https://docs.filecoin.io) and [Filecoin Spec](https://filecoin-project.github.io/specs/).

## What can I learn here?

- How to install epik on [Arch Linux](https://docs.lotu.sh/en+install-epik-arch), [Ubuntu](https://docs.lotu.sh/en+install-epik-ubuntu), or [MacOS](https://docs.lotu.sh/en+install-epik-macos).
- Joining the [epik Testnet](https://docs.lotu.sh/en+join-testnet).
- [Storing](https://docs.lotu.sh/en+storing-data) or [retrieving](https://docs.lotu.sh/en+retrieving-data) data.
- Mining Filecoin using the **epik Storage Miner** in your [CLI](https://docs.lotu.sh/en+mining).

## How is epik designed?

epik is architected modularly to keep clean API boundaries while using the same process. Installing epik will include two separate programs:

- The **epik Node**
- The **epik Storage Miner**

The **epik Storage Miner** is intended to be run on the machine that manages a single storage miner instance, and is meant to communicate with the **epik Node** via the websocket **JSON-RPC** API for all of the chain interaction needs.

This way, a mining operation may easily run a **epik Storage Miner** or many of them, connected to one or many **epik Node** instances.
