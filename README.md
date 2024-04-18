# Quick Links 🔗
Quick Links is a dynamic backend service that efficiently manages and redirects URL paths.

## Prerequisites
Before you begin, ensure you have the following installed on your system:
- [Docker](https://docs.docker.com/get-docker/)
- optional: [Make](https://www.gnu.org/software/make/)

## Getting Started

### Quick Start
- To start up the application on port 80:
  ```bash
  docker compose --profile production up
  ```
  
  or with make
  ```bash
  make production
  ```

### Development
- To start up the development environment with hot-reload (port `8080`):
  ```bash
  make dev
  ```

### Debugging
- To start the debugging environment with remote debugging tools (application on port `8080`, debugging on port `8083`):
  ```bash
  make debug
  ```

### Management Commands
- To stop all containers but keep volumes:
  ```bash
  make down
  ```

- To stop all containers and remove all volumes (full cleanup):
  ```bash
  make remove
  ```
  