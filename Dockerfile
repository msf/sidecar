# Build Stage
#FROM 506714715093.dkr.ecr.us-east-1.amazonaws.com/mirrorverse/docker.io/library/golang:1.14-buster AS build-stage
From golang:1.15-buster AS build-stage

ARG USER
ARG TOKEN

RUN apt-get update && apt-get install -y \
        protobuf-compiler \
        ca-certificates

RUN go env -w "GOPRIVATE=gitlab.com/Unbabel"
RUN echo -e "machine gitlab.com\nlogin ${USER}\npassword ${TOKEN}" > ~/.netrc

WORKDIR /app

COPY Makefile go.mod go.sum /app/
RUN make setup

COPY . /app/
RUN make

# Final Stage
# Use the official Alpine image for a lean production container.
# https://hub.docker.com/_/alpine
# https://docs.docker.com/develop/develop-images/multistage-build/#use-multi-stage-builds
FROM alpine:3
RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=build-stage /app/bin/sender /app/bin/sidecar /app/bin/web /app/
RUN chmod +x /app/

CMD ["/app/sender"]
