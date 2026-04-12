#!/bin/bash

CONTAINER_CMD=$(which podman || which docker)

$CONTAINER_CMD build -t localhost/access-control-status-bridge:latest -f Dockerfile

