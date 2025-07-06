# ---- Build Stage ----
FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build the application, creating a static binary.
# CGO_ENABLED=0 is used to build a statically linked binary.
# -ldflags="-w -s" strips debug information, reducing the binary size.
RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o /anihash .

# ---- Runtime Stage ----
FROM alpine:latest

WORKDIR /app

COPY --from=builder /anihash /app/anihash

# The application requires a config.yaml file to run.
# This command copies it from your project root into the container.
# You will need to create this file before building the image.
COPY config.yaml /app/config.yaml

# Expose the port that the server will listen on.
# This value is for documentation and to help automation.
# The actual port is determined by your config.yaml.
# We are using 8080 as a common default.
# When you run the container, map this port to a host port, for example:
# docker run -p 8080:8080 my-image
EXPOSE 8080

# The application may use an SQLite database.
# To ensure data is not lost when the container is stopped,
# you should mount a volume to the path where the database is stored.
# You can specify this path in your config.yaml. For example, if you set
# the database path to /app/data/anihash.db, you can run the container with:
# docker run -v my-app-data:/app/data my-image
# The following instruction is commented out as the path is dynamic.
# VOLUME /app/data

# Set the command to run when the container starts.
CMD ["/app/anihash"]
