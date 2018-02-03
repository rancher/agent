FROM ubuntu:17.10
RUN apt-get update && \
    apt-get install -y --no-install-recommends curl
ARG VERSION=dev
ENV AGENT_IMAGE rancher/agent:${VERSION}
COPY agent run.sh /usr/bin/
CMD ["run.sh"]
