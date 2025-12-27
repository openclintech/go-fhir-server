# ---- build stage ----
FROM golang:1.25-alpine AS build
WORKDIR /src

# cache deps
COPY go.mod ./
RUN go mod download

# copy source
COPY . ./

# build a static-ish binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/server ./cmd/server

# ---- run stage ----
FROM gcr.io/distroless/static-debian12
WORKDIR /
COPY --from=build /out/server /server

# Cloud Run uses PORT env var
EXPOSE 8080

CMD ["/server"]
