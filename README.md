# Iced-Mocha Core

Locally core relies on SQLite, you must install `sqlite3` to run core.

Once `sqite3` is installed run `./scripts/setup.sh` followed by `./scripts/genereateCert.sh` to generate a private key/cert file for a local https server.

Core can optionally be run inside Docker. To run using docker run `docker-compose up -d --build core`. To use outside of Docker
simply run `go run main.go`.
