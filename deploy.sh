#!/bin/bash

# BC Ferries API Deployment Script
# Run this from ~/bc-ferries-api on the server

echo "🚢 Starting BC Ferries API deployment..."
echo ""

# Pull latest code from master
echo "📥 Pulling latest code from master..."
git pull origin master

if [ $? -ne 0 ]; then
    echo "❌ Git pull failed. Please check for conflicts."
    exit 1
fi

echo ""
echo "🛑 Stopping containers..."
docker compose down

echo ""
echo "🔨 Building and starting containers..."
docker compose up -d --build

if [ $? -ne 0 ]; then
    echo "❌ Docker compose failed. Check logs for errors."
    exit 1
fi

echo ""
echo "✅ Deployment complete!"
echo ""
echo "📋 Container status:"
docker compose ps

echo ""
echo "📝 Recent API logs (Ctrl+C to exit):"
echo ""
docker compose logs -f api
