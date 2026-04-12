#!/bin/bash

CONTAINER_CMD=$(which podman || which docker)

$CONTAINER_CMD tag localhost/access-control-status-bridge:latest registry.bristolhackspace.org/access-control-status-bridge:latest

$CONTAINER_CMD push registry.bristolhackspace.org/access-control-status-bridge:latest

