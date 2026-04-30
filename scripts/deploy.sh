#!/bin/bash
# deploy.sh - Deploy KrillinAI-CLI to aiark-agent

set -e

BINARY="krilin-ai"
HOST="aiark-agent.tail4227fa.ts.net"
REMOTE_DIR="/opt/krilin-ai"

echo "Cross-compiling for Linux AMD64..."
GOOS=linux GOARCH=amd64 go build -o "/tmp/${BINARY}" ./cmd/cli/

echo "Copying to ${HOST}:${REMOTE_DIR}/..."
scp "/tmp/${BINARY}" "${HOST}:${REMOTE_DIR}/"

echo "Setting permissions on remote..."
ssh "${HOST}" "chmod +x ${REMOTE_DIR}/${BINARY}"

echo "Testing remote doctor command..."
ssh "${HOST}" "${REMOTE_DIR}/${BINARY} doctor"

echo ""
echo "✅ Deploy complete!"
echo "Run on aiark-agent: ssh ${HOST} '${REMOTE_DIR}/${BINARY} run <video_url>'"