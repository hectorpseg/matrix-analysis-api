FROM golang:alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/app .

FROM alpine
RUN adduser -D -H appuser
USER appuser
COPY --from=build /bin/app /bin/app
ENV PORT=8080
EXPOSE 8080
ENTRYPOINT ["/bin/app"]
