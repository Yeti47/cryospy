# End-to-End (E2E) Tests

This directory contains the End-to-End testing infrastructure for CryoSpy. The tests spin up a complete environment using Docker Compose, including the server, dashboard, and a capture client simulating a camera feed.

## Prerequisites

*   **Go 1.24+**
*   **Docker** and **Docker Compose** installed and running.
*   **FFmpeg** (optional, but recommended for faster test video generation). If not found locally, a Docker container will be used to generate the test video.

## Running the Tests

To run the E2E tests, execute the following command from the repository root:

```bash
go test -v ./tests/e2e
```

## How it Works

The `e2e_test.go` file performs the following steps:

1.  **Environment Setup**: Creates a temporary `testdata` directory in `tests/e2e/`.
2.  **Database Initialization**: Initializes a SQLite database with a test client and encryption keys.
3.  **Configuration Generation**: Generates `config.json` files for both the server and the client, pointing them to the correct paths inside the Docker containers.
4.  **Video Generation**: Generates a 70-second test video file (`test.mp4`) containing a timestamp and frame counter. This video is mounted into the client container to simulate a camera feed.
5.  **Container Orchestration**: Uses `docker-compose.yml` to build and start the `server` and `client` containers.
6.  **Execution**: The client container reads the video file, detects motion, and uploads clips to the server.
7.  **Cleanup**: Stops the containers and removes the `testdata` directory after the test completes (or fails).

## Troubleshooting

*   **Permission Issues**: If you encounter permission errors when cleaning up `testdata`, it might be because files created inside the container (like logs or database files) are owned by `root`. The test attempts to handle this, but you may need to manually remove the directory using `sudo rm -rf tests/e2e/testdata` if it persists.
*   **Docker Errors**: Ensure your user has permission to run Docker commands (e.g., is in the `docker` group).
