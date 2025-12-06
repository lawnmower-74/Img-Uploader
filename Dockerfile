FROM golang:1.21

ENV GOPATH /go
ENV PATH $GOPATH/bin:/usr/local/go/bin:$PATH

WORKDIR /usr/src/app

COPY app/go.mod app/go.sum ./
RUN go mod download

COPY app/ .

# main.goをコンパイル
RUN CGO_ENABLED=0 go build -o /usr/local/bin/uploader ./main.go

# コンパイル済みファイルを実行
CMD ["/usr/local/bin/uploader"]