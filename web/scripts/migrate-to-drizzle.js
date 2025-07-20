#!/usr/bin/env node

/**
 * Migration script to help migrate from Prisma to Drizzle
 * This script will:
 * 1. Generate Drizzle migrations
 * 2. Provide instructions for data migration
 */

const { execSync } = require('child_process');
const fs = require('fs');
const path = require('path');

console.log('🚀 Starting migration from Prisma to Drizzle...\n');

// Step 1: Generate Drizzle migrations
console.log('📝 Generating Drizzle migrations...');
try {
  execSync('npm run db:generate', { stdio: 'inherit' });
  console.log('✅ Drizzle migrations generated successfully!\n');
} catch (error) {
  console.error('❌ Failed to generate migrations:', error.message);
  process.exit(1);
}

// Step 2: Check if migrations directory exists
const migrationsDir = path.join(__dirname, '../drizzle/migrations');
if (fs.existsSync(migrationsDir)) {
  const migrations = fs.readdirSync(migrationsDir).filter(file => file.endsWith('.sql'));
  console.log(`📁 Found ${migrations.length} migration files:`);
  migrations.forEach(migration => {
    console.log(`   - ${migration}`);
  });
  console.log('');
}

// Step 3: Provide next steps
console.log('📋 Next steps:');
console.log('1. Review the generated migrations in drizzle/migrations/');
console.log('2. Update your DATABASE_URL to point to your Neon database');
console.log('3. Run: npm run db:migrate');
console.log('4. Update your Go backend to use the new schema');
console.log('5. Test your application');
console.log('');

console.log('🔧 Additional setup:');
console.log('- Install @neondatabase/serverless: npm install @neondatabase/serverless');
console.log('- Update your .env file with the new DATABASE_URL');
console.log('- Remove Prisma dependencies: npm uninstall @prisma/client prisma');
console.log('');

console.log('✨ Migration setup complete!'); 