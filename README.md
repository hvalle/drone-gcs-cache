# drone-gcs-cache

**This plugin is based on the [drone-s3-cache](https://github.com/drone-plugins/drone-s3-cache) plugin.**

Drone plugin that allows you to cache directories within the build workspace, this plugin works with Google Cloud Storage only.For the usage information and a listing of the available options please take a look at [the docs](http://plugins.drone.io/hvalle/drone-gcs-cache/).


## Build

Build the binary with the following commands:

```
go build
```

## Docker

Build the Docker image with the following commands:

```
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a -tags netgo -o release/linux/amd64/drone-gcs-cache
docker build --rm -t homerovalle/drone-gcs-cache .
```

## Usage

Execute from the working directory:

```
docker run --rm \
  -e PLUGIN_FLUSH=true \
  -e PLUGIN_JOSN_KEY="jsonkey" \
  -v $(pwd):$(pwd) \
  -w $(pwd) \
  homerovalle/drone-gcs-cache

docker run --rm \
  -e PLUGIN_RESTORE=true \
  -e PLUGIN_JOSN_KEY="jsonkey" \
  -e DRONE_REPO_OWNER="foo" \
  -e DRONE_REPO_NAME="bar" \
  -e DRONE_COMMIT_BRANCH="test" \
  -v $(pwd):$(pwd) \
  -w $(pwd) \
  homerovalle/drone-gcs-cache

docker run -it --rm \
  -v $(pwd):$(pwd) \
  -w $(pwd) \
  alpine:latest sh -c "mkdir -p cache && echo 'testing cache' >> cache/test && cat cache/test"

docker run --rm \
  -e PLUGIN_REBUILD=true \
  -e PLUGIN_MOUNT=".bundler" \
  -e PLUGIN_JOSN_KEY="jsonkey" \
  -e DRONE_REPO_OWNER="foo" \
  -e DRONE_REPO_NAME="bar" \
  -e DRONE_COMMIT_BRANCH="test" \
  -v $(pwd):$(pwd) \
  -w $(pwd) \
  homerovalle/drone-gcs-cache
```
