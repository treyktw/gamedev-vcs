# Migration Guide: Prisma to Drizzle + Neon

This guide will help you migrate from Prisma to Drizzle with Neon database for better Go backend integration.

## Overview

We're migrating from:
- **Prisma** → **Drizzle ORM**
- **Local/Cloud PostgreSQL** → **Neon Database**
- **Redis sessions** → **Database sessions**
- **Prisma Client** → **Drizzle Client**

## Benefits

1. **Better Go Integration**: Drizzle has excellent Go support
2. **Type Safety**: Full TypeScript and Go type safety
3. **Performance**: Neon provides better performance and scalability
4. **Simplified Stack**: Remove Redis dependency
5. **Unified Schema**: Same schema for frontend and backend

## Prerequisites

1. Install required packages:
```bash
npm install drizzle-orm @neondatabase/serverless drizzle-kit @auth/drizzle-adapter
```

2. Set up a Neon database:
   - Go to [neon.tech](https://neon.tech)
   - Create a new project
   - Get your connection string

## Migration Steps

### 1. Update Environment Variables

Update your `.env` file:
```env
# Remove old Prisma URL
# DATABASE_URL="postgresql://..."

# Add new Neon URL
DATABASE_URL="postgres://user:password@ep-xxx-xxx-xxx.region.aws.neon.tech/dbname?sslmode=require"

# Remove Redis URL (no longer needed)
# REDIS_URL="redis://..."
```

### 2. Generate Drizzle Migrations

```bash
npm run db:generate
```

This will create migration files in `drizzle/migrations/`.

### 3. Run Migrations

```bash
npm run db:migrate
```

### 4. Update Go Backend

The Go backend has been updated to use the new Drizzle-compatible schema:

```go
// Use the new DrizzleDB instead of GORM
db, err := database.ConnectDrizzle()
if err != nil {
    log.Fatal(err)
}

// Run migrations
err = db.MigrateDrizzle()
if err != nil {
    log.Fatal(err)
}
```

### 5. Remove Prisma Dependencies

```bash
npm uninstall @prisma/client prisma @auth/prisma-adapter
```

### 6. Remove Redis Dependencies

```bash
npm uninstall @upstash/redis @upstash/ratelimit @auth/upstash-redis-adapter
```

## File Changes

### New Files Created:
- `drizzle/schema.ts` - Drizzle schema definition
- `drizzle.config.ts` - Drizzle configuration
- `lib/db.ts` - Database connection
- `lib/auth.ts` - Updated auth configuration
- `database/drizzle.go` - Go database connection
- `scripts/migrate-to-drizzle.js` - Migration helper script

### Files Updated:
- `package.json` - Updated dependencies
- `src/app/api/auth/[...nextauth]/route.ts` - Simplified auth route

### Files to Remove:
- `prisma/schema.prisma` - No longer needed
- `lib/redis.ts` - No longer needed
- `lib/rate-limiting-middleware.ts` - No longer needed

## Database Schema

The new schema maintains compatibility with your existing data structure:

### Key Tables:
- `users` - User accounts
- `accounts` - OAuth accounts
- `sessions` - User sessions
- `organizations` - Teams/companies
- `projects` - VCS projects
- `files` - Project files
- `commits` - Version control commits
- `branches` - Git-like branches

### Enums:
- `organization_role` - OWNER, ADMIN, MEMBER
- `team_role` - MAINTAINER, MEMBER
- `project_role` - ADMIN, WRITE, MEMBER, READ

## Testing the Migration

1. **Start the development server**:
```bash
npm run dev
```

2. **Test authentication**:
   - Try signing in with Google
   - Verify sessions are stored in the database

3. **Test Go backend**:
```bash
go run cmd/vcs-server/main.go
```

4. **Check database**:
```bash
npm run db:studio
```

## Troubleshooting

### Common Issues:

1. **Connection Errors**:
   - Verify your Neon connection string
   - Check SSL mode is set to `require`

2. **Migration Errors**:
   - Drop existing tables if needed
   - Run migrations in order

3. **Type Errors**:
   - Run `npm run build` to check for TypeScript errors
   - Update any remaining Prisma imports

### Rollback Plan:

If you need to rollback:
1. Restore your old `DATABASE_URL`
2. Reinstall Prisma: `npm install @prisma/client prisma`
3. Restore `prisma/schema.prisma`
4. Run `npx prisma generate`

## Performance Improvements

With Neon + Drizzle, you should see:
- **Faster queries** due to Neon's optimized PostgreSQL
- **Better connection pooling** with Drizzle
- **Reduced latency** with serverless connections
- **Simplified architecture** without Redis

## Next Steps

1. **Monitor Performance**: Check query performance in Neon dashboard
2. **Optimize Queries**: Use Drizzle's query builder for complex queries
3. **Add Indexes**: Monitor slow queries and add indexes as needed
4. **Backup Strategy**: Set up automated backups in Neon

## Support

If you encounter issues:
1. Check the [Drizzle documentation](https://orm.drizzle.team/)
2. Review [Neon documentation](https://neon.tech/docs)
3. Check the migration script output for errors

---

**Migration completed!** 🎉

Your application now uses Drizzle + Neon for better Go integration and performance. 