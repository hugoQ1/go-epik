# Epik documentation

This folder contains some Epik documentation mostly intended for Epik developers.

User documentation (including documentation for miners) has been moved to specific Epik sections in https://docs.filecoin.io:

- https://docs.filecoin.io/get-started/epik
- https://docs.filecoin.io/store/epik
- https://docs.filecoin.io/mine/epik
- https://docs.filecoin.io/build/epik

## The Lotu.sh site

The https://lotu.sh and https://docs.lotu.sh sites are generated from this folder based on the index provided by [.library.json](.library.json). This is done at the [epik-docs repository](https://github.com/EpiK-Protocol/go-epik-docs), which contains Epik as a git submodule.

To update the site, the epik-docs repository should be updated with the desired version for the epik git submodule. Once pushed to master, it will be auto-deployed.
