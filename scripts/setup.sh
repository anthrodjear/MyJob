#!/bin/bash
set -e

echo "Setting up AI Job Search Agent..."

# Check if .env exists
if [ ! -f .env ]; then
  echo "Creating .env from .env.example..."
  cp .env.example .env
  echo "Please edit .env with your API keys."
fi

# Start infrastructure
echo "Starting infrastructure services..."
docker compose up -d postgres redis ollama

# Wait for services
echo "Waiting for services to be ready..."
sleep 10

# Pull Ollama models
echo "Pulling Ollama models..."
docker compose exec ollama ollama pull mxbai-embed-large
docker compose exec ollama ollama pull qwen2.5:latest

echo "Setup complete!"
echo "Next steps:"
echo "  1. Edit .env with your API keys"
echo "  2. Run 'make migrate' to initialize the database"
echo "  3. Run 'make start' to start all services"
