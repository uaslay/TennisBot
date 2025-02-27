FROM golang:latest

COPY ./bin/TennisBot $GOPATH/bin

RUN mkdir -p /resources
COPY ./resources/* /resources

CMD ["TennisBot"]