#!/bin/bash

echo "🗄️  Setting up Database with Drizzle Schema..."

# Set environment variables if not already set
if [ -z "$DATABASE_URL" ]; then
    echo "⚠️  DATABASE_URL not set. Please set it to your Neon database URL."
    echo "Example: export DATABASE_URL='postgres://user:pass@ep-xxx.region.aws.neon.tech/dbname?sslmode=require'"
    exit 1
fi

echo "📊 Database URL: $DATABASE_URL"

# Create bin directory if it doesn't exist
mkdir -p bin

# Build the Go application
echo "🔨 Building Go application..."
go build -o bin/vcs-server ./cmd/vcs-server

if [ $? -ne 0 ]; then
    echo "❌ Build failed!"
    exit 1
fi

echo "✅ Build successful!"

# Run migrations
echo "🔄 Running database migrations..."
./bin/vcs-server migrate

if [ $? -ne 0 ]; then
    echo "❌ Migration failed!"
    exit 1
fi

echo "✅ Migrations completed!"

# Seed the database
echo "🌱 Seeding database..."
./bin/vcs-server seed

if [ $? -ne 0 ]; then
    echo "❌ Seeding failed!"
    exit 1
fi

echo "✅ Database setup completed!"
echo "🚀 You can now run: ./scripts/test-go-backend.sh" 