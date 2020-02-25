# FROM golang:1.13
# WORKDIR /go/src/app
# COPY . .
# RUN go get -d -v ./...
# RUN go install -v ./...
# CMD ["dataservice"]

FROM golang:alpine
WORKDIR /home
COPY bin/dataservice .
ENTRYPOINT ["/home/dataservice"]
