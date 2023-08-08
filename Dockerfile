FROM ubuntu

ENV GO_VERSION=1.19.2
ENV IGNITE_VERSION=0.25.1
ENV NODE_VERSION=18.x

ENV LOCAL=/usr/local
ENV GOROOT=$LOCAL/go
ENV HOME=/root
ENV GOPATH=$HOME/go
ENV PATH=$GOROOT/bin:$GOPATH/bin:$PATH

RUN mkdir -p $GOPATH/bin

RUN apt-get update -y
RUN apt update && apt install -y build-essential clang curl gcc jq wget zsh net-tools git

# Install Go
#RUN curl -L https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz | tar -C $LOCAL -xzf -
RUN curl -L https://studygolang.com/dl/golang/go${GO_VERSION}.linux-amd64.tar.gz | tar -C $LOCAL -xzf -

# Install Node
RUN curl -fsSL https://deb.nodesource.com/setup_${NODE_VERSION} | bash -
RUN apt-get install -y nodejs

# Install Ignite
RUN curl -L https://get.ignite.com/cli@v${IGNITE_VERSION}! | bash

RUN mkdir -p /sao-node
ADD . /sao-node
ENV GOPROXY=https://goproxy.io,direct
# ENV http_proxy=http://192.168.50.42:10807
# ENV https_proxy=http://192.168.50.42:10807;
# RUN cd /sao-consensus && ignite chain build
RUN cd /sao-node && make clean && make all
RUN rm -rf /sao-node
VOLUME /root/.sao-node

EXPOSE 5151 5152 5153
CMD ["sleep", "infinity"]

