FROM ubuntu:17.10
RUN apt-get update && \
    apt-get install -y --no-install-recommends curl ca-certificates jq iproute2 && \
    curl -sLf https://get.docker.com/builds/Linux/x86_64/docker-1.10.3 > /usr/bin/docker && \
    chmod +x /usr/bin/docker
ARG VERSION=dev
ENV AGENT_IMAGE rancher/agent:${VERSION}
COPY agent run.sh /usr/bin/
ENTRYPOINT ["run.sh"]
