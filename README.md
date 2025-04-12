# Agent Service

## Make
### Build
```bash
make -f ./scripts/Makefile build
```

### Run
```bash
make -f ./scripts/Makefile run
```

### Clean
```bash
make -f ./scripts/Makefile clean
```

## Docker

### Build
```bash
docker build -t agent-service -f ./.docker/Dockerfile .
```

### Run
```bash
docker run -p 8080:8080 agent-service
```