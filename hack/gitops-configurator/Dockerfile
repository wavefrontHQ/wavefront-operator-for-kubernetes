FROM ubuntu
WORKDIR /

RUN apt-get -y update
RUN apt-get -y install git curl
RUN curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
RUN install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

COPY configurator.sh .
ENTRYPOINT ["bash"]
#ENTRYPOINT ["/configurator.sh"]
