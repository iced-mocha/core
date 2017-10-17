FROM golang:1.9

RUN go get -u github.com/golang/dep/cmd/dep && go install github.com/golang/dep/cmd/dep

WORKDIR /go/src/github.com/iced-mocha/core
COPY . /go/src/github.com/iced-mocha/core

RUN dep ensure -v
RUN go install -v

ENTRYPOINT ["core"]
