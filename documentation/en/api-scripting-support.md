# Remote API Support

You may want to delegate the work **epik Storage Miner** or **epik Node** performs to other machines. 
Here is how to setup the necessary authorization and environment variables.

## Environment variables

Environmental variables are variables that are defined for the current shell and are inherited by any child shells or processes. Environmental variables are used to pass information into processes that are spawned from the shell.

Using the [JWT you generated](https://lotu.sh/en+api#how-do-i-generate-a-token-18865), you can assign it and the **multiaddr** to the appropriate environment variable.

```sh
# epik Node
FULLNODE_API_INFO="JWT_TOKEN:/ip4/127.0.0.1/tcp/1234/http"

# epik Storage Miner
STORAGE_API_INFO="JWT_TOKEN:/ip4/127.0.0.1/tcp/2345/http"
```

You can also use `epik auth api-info --perm admin` to quickly create _API_INFO env vars

- The **epik Node**'s `mutliaddr` is in `~/.epik/api`.
- The default token is in `~/.epik/token`.
- The **epik Storage Miner**'s `multiaddr` is in `~/.epikstorage/config`.
- The default token is in `~/.epikstorage/token`.
