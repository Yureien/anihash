# Anihash

Anihash is a self-hosted server that acts as a proxy between your applications and [AniDB](https://anidb.net/). It is designed to query AniDB for file information using ed2k hashes and file sizes. It caches this information in a local database to reduce redundant queries, respect AniDB's API rate limits, and provide a fast, local source for your media file data.

## Features

- **Caching:** Stores file information locally to minimize API calls to AniDB.
- **Queueing:** Pending requests for new files are queued and processed in the background.
- **Simple API:** A straightforward HTTP API to query for file information.
- **Docker Support:** Ready to be deployed as a Docker container.
- **CLI:** Includes a command-line tool for easy interaction (see `anilookup`).
- **Automatic Scanning:** Anihash can scan directories for new video files and automatically process them.

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

server:
  host: 0.0.0.0
  port: 8080

database:
  sqlite:
    path: /data/anihash.db # Recommended path for Docker
    # path: anihash.db # For local non-docker usage

scanner:
  # Path to scan for video files. Leave empty or remove to disable.
  scan_path: /path/to/your/media
```

### Parameters

-   `anidb`:
    -   `user`: Your AniDB API username.
    -   `password`: Your AniDB API password.
    -   `address`: The AniDB UDP API address.
-   `server`:
    -   `host`: The host address for the server to listen on.
    -   `port`: The port for the server to listen on.
-   `database`:
    -   `sqlite.path`: The path to the SQLite database file.
-   `scanner` (optional):
    -   `scan_path`: The path to a directory to scan for video files. If this is set, anihash will scan the directory on startup and watch for new files to automatically process them.

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

Anihash provides a simple HTTP API to query for file information. You can also access an interactive API documentation with forms by navigating to the root URL of the server (e.g., `http://localhost:8080`).

#### `GET /query/ed2k`

This endpoint allows you to query file information using the file's size and ed2k hash. This is the canonical way to query file information, and will fetch from AniDB if the file is not in the database.

**Query Parameters:**

-   `size` (integer, required): The size of the file in bytes.
-   `ed2k` (string, required): The ed2k hash of the file.

**Example Request:**

```sh
curl "http://localhost:8080/query/ed2k?size=12345678&ed2k=abcdef1234567890abcdef1234567890"
```

**Example Response (File Available):**
If the file is found in the cache, the server returns the file data.
```json
{
  "file": {
    "FileID": 12345,
    "Ed2k": "abcdef1234567890abcdef1234567890",
    "Size": 12345678,
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

#### `GET /query/hash`

This endpoint allows you to query file information using the file's SHA1 or MD5 hash. This endpoint will only search the local database, and will not fetch from AniDB.

**Query Parameters:**

-   `hash` (string, required): The SHA1 or MD5 hash of the file.

**Example Request:**

```sh
curl "http://localhost:8080/query/hash?hash=8c88c204d48243952f1b8949f4c042079f0da2e5"
```

**Example Response (File Found):**
If the file is found in the local database, the server returns the file data.
```json
{
  "file": {
    "FileID": 54321,
    "Ed2k": "f0e9d8c7b6a54321f0e9d8c7b6a54321",
    "Size": 987654321,
    // ... other fields
  },
  "state": {
    "State": "FILE_AVAILABLE"
  }
}
```

**Example Response (File Not Found):**
If the file is not found in the local database, the server returns a "not found" state.
```json
{
  "file": null,
  "state": {
    "State": "FILE_NOT_FOUND"
  }
}
```

## File Scanner

Anihash can optionally scan a directory on your filesystem to find video files, hash them, and add them to the local database. This is useful for pre-populating the cache with your entire media library.

### How it works

1.  **Initial Scan:** When the server starts, it walks through the entire `scan_path` directory and processes any video files it finds.
2.  **Watching for Changes:** After the initial scan, it uses a file watcher to monitor the directory for any new or modified files, processing them as they appear. This means you can add new files to your library and anihash will automatically pick them up without a restart.

To enable this feature, add the `scanner` section to your `config.yaml` and provide a `scan_path`.

### CLI Tool (`anilookup`)

For command-line interaction with the anihash server, please refer to the `anilookup` tool. Instructions can be found in its README file: [anilookup/README.md](anilookup/README.md).
