# Adaptive Anti DoS
A self-adaptive system for mitigating DoS attacks to servers.

## How to Run
1. Install and configure [*docker swarm*](https://docs.docker.com/engine/swarm/)
2. Build images:
    ```bash
    docker build -f server/deploy/Dockerfile -t server:0.1 .
    docker build -f controller/deploy/Dockerfile -t controller:0.1 .
    ```
3. Deploy stack to swarm:
    ```bash
    docker stack deploy -d -c ./docker-compose.yaml aad
    ```
4. Send requests to the node's `4000` port or `localhost:4000` if your swarm is local:
    ```bash
    curl http://localhost:4000/a.png
    ```
   You can also run the load test program:
    ```bash
    go run ./loadtest/main.go
    ```
