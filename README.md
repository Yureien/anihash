# Anihash

Anihash is a self-hosted server that acts as a proxy between your applications and [AniDB](https://anidb.net/). It is designed to query AniDB for file information using ed2k hashes and file sizes. It caches this information in a local database to reduce redundant queries, respect AniDB's API rate limits, and provide a fast, local source for your media file data.

## Features

- **Caching:** Stores file information locally to minimize API calls to AniDB.
- **Queueing:** Pending requests for new files are queued and processed in the background.
- **Simple API:** A straightforward HTTP API to query for file information.
- **Docker Support:** Ready to be deployed as a Docker container.
- **CLI:** Includes a command-line tool for easy interaction (see `anilookup`).

## Installation

You can run Anihash either from source or using Docker.

### From Source

**Prerequisites:**
- [Go](https://go.dev/doc/install) (version 1.22 or newer)

**Steps:**

1.  Clone the repository:
    ```sh
    git clone https://github.com/yureien/anihash.git
    cd anihash
    ```

2.  Create a `config.yaml` file by copying the example or creating your own. See the [Configuration](#configuration) section for details.

3.  Build the application:
    ```sh
    go build -o anihash .
    ```

4.  Run the server:
    ```sh
    ./anihash
    ```

### With Docker

1.  Clone the repository:
    ```sh
    git clone https://github.com/yureien/anihash.git
    cd anihash
    ```

2.  Create a `config.yaml` file. See the [Configuration](#configuration) section for details.

3.  Build the Docker image:
    ```sh
    docker build -t anihash .
    ```

4.  Run the Docker container:
    ```sh
    docker run -d \
      -p 8080:8080 \
      -v $(pwd)/config.yaml:/app/config.yaml \
      -v anihash-data:/data \
      --name anihash \
      anihash
    ```
    This command will:
    - Run the container in detached mode (`-d`).
    - Map port `8080` of the container to port `8080` on the host.
    - Mount your local `config.yaml` into the container.
    - Create a Docker volume named `anihash-data` to persist the SQLite database.
    - Name the container `anihash`.

    **Note:** For the volume mount to work correctly, make sure the `database.sqlite.path` in your `config.yaml` points to a path inside the mounted volume, for example `/data/anihash.db`.

## Configuration

Anihash is configured using a `config.yaml` file in the same directory as the executable.

Here is an example configuration:
```yaml
anidb:
  address: api.anidb.net:9000
  user: "your-anidb-username"
  password: "your-anidb-password"
  client_name: "myapp"
  client_version: 1

server:
  host: 0.0.0.0
  port: 8080

database:
  sqlite:
    path: /data/anihash.db # Recommended path for Docker
    # path: anihash.db # For local non-docker usage
```

### Parameters

-   `anidb`:
    -   `user`: Your AniDB API username.
    -   `password`: Your AniDB API password.
    -   `client_name`: The client name to identify with AniDB.
    -   `client_version`: The client version to identify with AniDB.
    -   `address`: The AniDB UDP API address.
-   `server`:
    -   `host`: The host address for the server to listen on.
    -   `port`: The port for the server to listen on.
-   `database`:
    -   `sqlite.path`: The path to the SQLite database file.

## Usage

### Running the Server

Once you have your `config.yaml` ready, simply run the executable or the Docker container. The server will start and begin listening for requests.

```sh
./anihash
```
or for docker:
```sh
docker start anihash
```

### API

Anihash provides a simple HTTP API to query for file information.

#### `GET /query`

This endpoint retrieves file information from the database or queues a request to AniDB if the file is not yet known.

**Query Parameters:**

-   `ed2k` (string, required): The ed2k hash of the file.
-   `size` (integer, required): The size of the file in bytes.

**Example Request:**

```sh
curl "http://localhost:8080/query?ed2k=0123456789abcdef0123456789abcdef&size=123456789"
```

**Example Response (File Available):**
If the file is found in the cache, the server returns the file data.
```json
{
  "file": {
    "FileID": 12345,
    "Ed2k": "0123456789abcdef0123456789abcdef",
    "Size": 123456789,
    // ... other fields
  },
  "state": {
    "State": "FILE_AVAILABLE"
  }
}
```

**Example Response (File Pending):**
If the file is not in the cache, the server queues a request to AniDB and returns a pending state. You can query again later to get the full file data.
```json
{
  "file": null,
  "state": {
    "State": "FILE_PENDING"
  }
}
```

### CLI Tool (`anilookup`)

For command-line interaction with the anihash server, please refer to the `anilookup` tool. Instructions can be found in its README file: [anilookup/README.md](anilookup/README.md).
