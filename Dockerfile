FROM golang:1.9

RUN go get -u github.com/golang/dep/cmd/dep && go install github.com/golang/dep/cmd/dep

WORKDIR /go/src/github.com/iced-mocha/core
COPY . /go/src/github.com/iced-mocha/core

RUN dep ensure -v && go install -v
RUN apt-get -y update && apt-get -y upgrade
RUN apt-get -y install sqlite3 libsqlite3-dev
RUN ./scripts/setup.sh

ENTRYPOINT ["core"]
