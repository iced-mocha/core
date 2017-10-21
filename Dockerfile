FROM icedmocha/core-base:latest

WORKDIR /go/src/github.com/iced-mocha/core
COPY . /go/src/github.com/iced-mocha/core

RUN dep ensure -v && go install -v
RUN ./scripts/setup.sh

ENTRYPOINT ["core"]
