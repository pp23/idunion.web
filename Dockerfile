FROM golang:1.23.1-alpine3.20 AS build

WORKDIR /app

RUN apk --no-cache add build-base

COPY go.mod go.sum ./
RUN go mod tidy
RUN go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.4.1

COPY . .

RUN make build-release

RUN mkdir -p /etc && \
    echo 'nobody:x:65534:65534:nobody:/:' > /etc/passwd && \
    echo 'nobody:x:65534:' > /etc/group

FROM scratch

# in case the application requires these variables
ENV USER=appuser
ENV HOME=/home/$USER

# actual user
USER nobody:nobody

COPY --from=build /etc/passwd /etc/passwd
COPY --from=build /etc/group /etc/group
COPY --from=build /app/out/bin/htmx /app/htmx

# copy web content.
# As of now, web files need to be in the same location where the htmx app gets started to get found.
COPY --from=build /app/static /app/static
COPY --from=build /app/templates /app/templates

EXPOSE 3000

WORKDIR /app

ENTRYPOINT ["./htmx"]
