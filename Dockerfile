FROM vxlabs/glide as builder

WORKDIR $GOPATH/src/github.com/jbonachera/vault-unsealer
COPY glide* ./
RUN glide install -v
COPY . ./
RUN go test $(glide nv) && \
    go build -buildmode=exe -a -o /bin/vault-unsealer ./main.go

FROM alpine
COPY --from=builder /bin/vault-unsealer /bin/vault-unsealer

