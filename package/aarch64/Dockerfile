FROM ubuntu:18.04@sha256:4ffec8d4ab8efc7e08199bbcbb4bd403b0a29f84c2d617b4349783afa6551f32
RUN apt-get update && \
    apt-get install -y --no-install-recommends curl ca-certificates jq iproute2 && \
    curl -sLf https://download.docker.com/linux/static/stable/aarch64/docker-17.09.0-ce.tgz > /tmp/docker.tgz && \
    cd /tmp && tar -xf docker.tgz && mv docker/docker /usr/bin/docker && rm -rf /tmp/docker /tmp/docker.tgz && \
    chmod +x /usr/bin/docker
ARG VERSION=dev
ENV AGENT_IMAGE rancher/agent:${VERSION}
COPY agent run.sh /usr/bin/
ENTRYPOINT ["run.sh"]
