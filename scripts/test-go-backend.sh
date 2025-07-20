#!/bin/bash

echo "🚀 Testing Go Backend with Drizzle Schema..."

# Set environment variables if not already set
if [ -z "$DATABASE_URL" ]; then
    echo "⚠️  DATABASE_URL not set. Please set it to your Neon database URL."
    echo "Example: export DATABASE_URL='postgres://user:pass@ep-xxx.region.aws.neon.tech/dbname?sslmode=require'"
    exit 1
fi

echo "📊 Database URL: $DATABASE_URL"

# Build the Go application
echo "🔨 Building Go application..."
go build -o bin/vcs-server ./cmd/vcs-server

if [ $? -ne 0 ]; then
    echo "❌ Build failed!"
    exit 1
fi

echo "✅ Build successful!"

# Run the server
echo "🚀 Starting server..."
echo "Press Ctrl+C to stop"
./bin/vcs-server 