Litestream Read Replica Example
===============================

This repository is an example of how to setup and deploy a multi-node SQLite
database using Litestream's live read replication feature. For more information,
please see the [Read Replication Guide](https://litestream.io/guides/read-replica/)
on the Litestream documentation site.

This example provides a Dockerfile and deployment instructions for [fly.io](https://fly.io),
however, any hosting provider can be used.

## Deployment

### Docker

To build the Docker image, run:

```sh
docker build -t litestream-read-replica-example .
```

### Deploy to fly.io

To deploy to [fly.io](https://fly.io), you'll need to create an application,
volumes, and configure your primary region.

```sh
# Application names are unique on fly.io so you'll need to pick one.
export APPNAME=...

# Create a new application but don't deploy yet.
fly launch --name "$APPNAME" --region ord --no-deploy

# Create a volume in our primary region first.
fly volumes create --region ord --size 1 data
```

We'll also add `[env]` and `[mounts]` sections to the `fly.toml` file. This
tells our application which region should be the primary and also attaches the
persistent `data` volume to each instance.

```toml
[env]
  FLY_PRIMARY_REGION = "ord"

[mounts]
  source="data"
  destination="/data"
```

Finally, we'll deploy our application:

```sh
fly deploy
```

Once we confirm that our primary is successfully running, we'll deploy our
replicas:

```sh
# Create volumes for each replica.
fly volumes create --region lhr --size 1 data
fly volumes create --region syd --size 1 data

# Scale the app so we have one instance for each volume.
fly scale count 3
```

Once you are done with your application, you can tear it down with the following:

```sh
fly apps destroy "$APPNAME"
```

## Usage

The included application only has two endpoints:

1. `POST /` - creates a new record
2. `GET /` - returns a list of all records

First, we'll add some records:

```sh
curl -X POST https://${APPNAME}.fly.dev/ -d 'foo'
curl -X POST https://${APPNAME}.fly.dev/ -d 'bar'
curl -X POST https://${APPNAME}.fly.dev/ -d 'baz'
```

Then we can retrieve a list of those records:

```sh
curl https://${APPNAME}.fly.dev/
```

This should return the following:

```
Record #1: "foo"
Record #2: "bar"
Record #3: "baz"
```

### Viewing replica data

Fly.io automatically routes requests to the nearest instance so we'll need to
log into our instances if we want to verify the replicated data. First, you'll
need to set up access to your private network over Wireguard:

https://fly.io/docs/reference/private-networking/#private-network-vpn

Once set up, we can cURL against a specific region's server:

```sh
curl lhr.${APPNAME}.internal:8080
```

You can also SSH into the node and verify with the SQLite CLI. Be sure to use
the `-readonly` flag to ensure you do not accidentally alter the database on
the replica.

```sh
fly ssh console lhr.${APPNAME}.internal

sqlite3 -readonly /data/db

sqlite> SELECT * FROM records;
1|foo
2|bar
3|baz
```

