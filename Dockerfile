FROM plugins/base:multiarch

ADD release/linux/amd64/drone-gcs-cache /bin/
ENTRYPOINT ["/bin/drone-gcs-cache"]
