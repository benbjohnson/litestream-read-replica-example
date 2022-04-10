# Import Litestream docker image as a stage so we can pull the binary from it.
FROM litestream/litestream:0.4.0-beta.2 AS litestream

# Build our application in the second stage so we can pull the binary in the last stage.
# Our Go build uses 'static' flags so that we can run in alpine.
FROM golang:1.17 as builder
COPY . /src/litestream-read-replica-example
WORKDIR /src/litestream-read-replica-example
RUN --mount=type=cache,target=/root/.cache/go-build \
	--mount=type=cache,target=/go/pkg \
	go build -ldflags '-s -w -extldflags "-static"' -tags osusergo,netgo,sqlite_omit_load_extension -o /usr/local/bin/litestream-read-replica-example .


# This is where our final stage of the Docker build begins.
# We are using alpine to make our image as small as possible. 
FROM alpine

# Set environment variables.
ENV DSN "/data/db"

EXPOSE 8080

# These lines copy the binaries from the previous builds.
COPY --from=builder /usr/local/bin/litestream-read-replica-example /usr/local/bin/litestream-read-replica-example
COPY --from=litestream /usr/local/bin/litestream /usr/local/bin/litestream

# Copy both our primary configuration & replica configuration.
# We will choose which one we use based at runtime.
ADD etc/litestream.primary.yml /etc/litestream.primary.yml
ADD etc/litestream.replica.yml /etc/litestream.replica.yml
ADD run.sh /run.sh

# We add sqlite so we can create an empty database for our replica to start with.
# Add bash & cURL in case we want to ssh into our image. These are not necessary.
RUN apk add sqlite bash ca-certificates curl

# Set up directory to hold database. This may be mounted by the host.
RUN mkdir -p /data

# This script determines the replication mode, restores a back up if necessary,
# and starts Litestream and the application.
CMD /run.sh
