FROM rancher/agent-base:v0.3.0
ARG IMAGE
COPY register.py resolve_url.py run.sh /
ENTRYPOINT ["/run.sh"]
LABEL "io.rancher.container.system"="rancher-agent"
RUN curl -sL https://github.com/rancher/share-mnt/releases/download/v1.0.0/share-mnt.tar.gz | tar xvzf - -C /usr/bin
COPY agent agent.sh /usr/bin/
ENV RANCHER_AGENT_IMAGE $IMAGE
