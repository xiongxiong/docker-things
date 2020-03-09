# FROM golang:1.13
# WORKDIR /go/src/app
# COPY . .
# RUN go get -d -v ./...
# RUN go install -v ./...
# CMD ["dataservice"]

FROM golang:alpine

# add dataservice
WORKDIR /home
COPY config/config.yml .
COPY bin/dataservice .

# add docker-compose-wait tool
ENV WAIT_VERSION 2.7.3
ADD https://github.com/ufoscout/docker-compose-wait/releases/download/$WAIT_VERSION/wait /home/wait
RUN chmod +x /wait

ENTRYPOINT ["/home/dataservice"]
