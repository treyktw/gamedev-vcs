// lib/trpc/index.ts - tRPC setup with real Go backend
import { initTRPC, TRPCError } from '@trpc/server';
import { z } from 'zod';
import { auth } from '@/auth';

// Create context for tRPC
export async function createContext() {
  const session = await auth();
  
  return {
    session,
    user: session?.user,
    apiURL: process.env.NEXT_PUBLIC_VCS_API_URL || 'http://localhost:8080',
  };
}

type Context = Awaited<ReturnType<typeof createContext>>;

// Initialize tRPC
const t = initTRPC.context<Context>().create();

// Middleware for authentication
const authMiddleware = t.middleware(({ ctx, next }) => {
  if (!ctx.session?.user) {
    throw new TRPCError({
      code: 'UNAUTHORIZED',
      message: 'Authentication required',
    });
  }
  
  return next({
    ctx: {
      ...ctx,
      user: ctx.session.user,
    },
  });
});

// Helper function to make requests to Go backend
async function makeGoRequest<T>(
  ctx: Context,
  endpoint: string,
  options: RequestInit = {}
): Promise<T> {
  const url = `${ctx.apiURL}${endpoint}`;
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...options.headers as Record<string, string>,
  };

  // Add auth headers if user is authenticated
  if (ctx.session?.user?.id) {
    headers['X-User-ID'] = ctx.session.user.id;
    headers['X-User-Name'] = ctx.session.user.name || 'Unknown';
  }

  const response = await fetch(url, {
    ...options,
    headers,
  });

  if (!response.ok) {
    const errorData = await response.json().catch(() => ({ error: 'Unknown error' }));
    throw new TRPCError({
      code: 'INTERNAL_SERVER_ERROR',
      message: `Backend API Error: ${response.status} - ${errorData.error || response.statusText}`,
    });
  }

  return response.json();
}

// Router and procedure builders
export const router = t.router;
export const publicProcedure = t.procedure;
export const protectedProcedure = t.procedure.use(authMiddleware);

// Input validation schemas
const CreateProjectSchema = z.object({
  name: z.string().min(1, 'Project name is required'),
  description: z.string().optional(),
  isPrivate: z.boolean().default(true),
  organizationId: z.string().optional(),
});

const FileUploadSchema = z.object({
  projectId: z.string(),
  filePath: z.string().optional(),
  fileName: z.string(),
  fileSize: z.number(),
  contentType: z.string(),
});

const FileLockSchema = z.object({
  projectId: z.string(),
  filePath: z.string(),
  userName: z.string().optional(),
  sessionId: z.string().optional(),
});

// Main tRPC router
export const appRouter = router({
  // Health check
  health: publicProcedure.query(async ({ ctx }) => {
    try {
      const health = await makeGoRequest(ctx, '/health');
      return { status: 'healthy', health: health };
    } catch (error) {
      throw new TRPCError({
        code: 'INTERNAL_SERVER_ERROR',
        message: 'Backend health check failed',
        cause: error,
      });
    }
  }),

  // Projects
  projects: router({
    // Get all projects for user (REAL endpoint)
    list: protectedProcedure.query(async ({ ctx }) => {
      try {
        const result = await makeGoRequest<{ success: boolean; projects: any[] }>(
          ctx, 
          '/api/v1/projects'
        );
        return result.projects;
      } catch (error) {
        throw new TRPCError({
          code: 'INTERNAL_SERVER_ERROR',
          message: 'Failed to fetch projects',
          cause: error,
        });
      }
    }),

    // Create new project (REAL endpoint)
    create: protectedProcedure
      .input(CreateProjectSchema)
      .mutation(async ({ input, ctx }) => {
        try {
          const result = await makeGoRequest<{ success: boolean; project: any }>(
            ctx,
            '/api/v1/projects',
            {
              method: 'POST',
              body: JSON.stringify({
                name: input.name,
                description: input.description,
                is_private: input.isPrivate,
                organization_id: input.organizationId,
              }),
            }
          );
          return result.project;
        } catch (error) {
          throw new TRPCError({
            code: 'INTERNAL_SERVER_ERROR',
            message: 'Failed to create project',
            cause: error,
          });
        }
      }),

    // Get project by ID (REAL endpoint)
    byId: protectedProcedure
      .input(z.object({ id: z.string() }))
      .query(async ({ input, ctx }) => {
        try {
          const result = await makeGoRequest<{ success: boolean; project: any }>(
            ctx,
            `/api/v1/projects/${input.id}`
          );
          return result.project;
        } catch (error) {
          throw new TRPCError({
            code: 'NOT_FOUND',
            message: 'Project not found',
            cause: error,
          });
        }
      }),

    // Update project (REAL endpoint)
    update: protectedProcedure
      .input(z.object({
        id: z.string(),
        updates: CreateProjectSchema.partial(),
      }))
      .mutation(async ({ input, ctx }) => {
        try {
          const result = await makeGoRequest<{ success: boolean; project: any }>(
            ctx,
            `/api/v1/projects/${input.id}`,
            {
              method: 'PATCH',
              body: JSON.stringify(input.updates),
            }
          );
          return result.project;
        } catch (error) {
          throw new TRPCError({
            code: 'INTERNAL_SERVER_ERROR',
            message: 'Failed to update project',
            cause: error,
          });
        }
      }),

    // Delete project (REAL endpoint)
    delete: protectedProcedure
      .input(z.object({ id: z.string() }))
      .mutation(async ({ input, ctx }) => {
        try {
          await makeGoRequest<{ success: boolean }>(
            ctx,
            `/api/v1/projects/${input.id}`,
            {
              method: 'DELETE',
            }
          );
          return { success: true };
        } catch (error) {
          throw new TRPCError({
            code: 'INTERNAL_SERVER_ERROR',
            message: 'Failed to delete project',
            cause: error,
          });
        }
      }),
  }),

  // Storage management (REAL endpoints)
  storage: router({
    stats: protectedProcedure.query(async ({ ctx }) => {
      try {
        const result = await makeGoRequest<{ success: boolean; stats: any }>(
          ctx,
          '/api/v1/system/storage/stats'
        );
        return result.stats;
      } catch (error) {
        throw new TRPCError({
          code: 'INTERNAL_SERVER_ERROR',
          message: 'Failed to fetch storage stats',
          cause: error,
        });
      }
    }),

    cleanup: protectedProcedure
      .input(z.object({
        type: z.enum(['sessions', 'storage', 'all']).default('all'),
      }))
      .mutation(async ({ input, ctx }) => {
        try {
          await makeGoRequest<{ success: boolean }>(
            ctx,
            `/api/v1/system/cleanup?type=${input.type}`,
            {
              method: 'POST',
            }
          );
          return { success: true };
        } catch (error) {
          throw new TRPCError({
            code: 'INTERNAL_SERVER_ERROR',
            message: 'Failed to perform cleanup',
            cause: error,
          });
        }
      }),
  }),

  // File operations (REAL endpoints)
  files: router({
    // Get file list for project (REAL endpoint)
    list: protectedProcedure
      .input(z.object({ projectId: z.string() }))
      .query(async ({ input, ctx }) => {
        try {
          const result = await makeGoRequest<{ success: boolean; files: any[] }>(
            ctx,
            `/api/v1/projects/${input.projectId}/files`
          );
          return result.files || [];
        } catch (error) {
          throw new TRPCError({
            code: 'INTERNAL_SERVER_ERROR',
            message: 'Failed to fetch project files',
            cause: error,
          });
        }
      }),
  }),

  // File locking (REAL endpoints)
  locks: router({
    // Get locks for project (REAL endpoint)
    list: protectedProcedure
      .input(z.object({ projectId: z.string() }))
      .query(async ({ input, ctx }) => {
        try {
          const result = await makeGoRequest<{ success: boolean; locks: any[] }>(
            ctx,
            `/api/v1/locks/${input.projectId}`
          );
          return result.locks;
        } catch (error) {
          throw new TRPCError({
            code: 'INTERNAL_SERVER_ERROR',
            message: 'Failed to fetch locks',
            cause: error,
          });
        }
      }),

    // Lock file (REAL endpoint)
    lock: protectedProcedure
      .input(FileLockSchema)
      .mutation(async ({ input, ctx }) => {
        try {
          const userData = {
            user_name: input.userName || ctx.user.name || 'Unknown',
            session_id: input.sessionId || `session_${Date.now()}`,
          };
          
          const result = await makeGoRequest<any>(
            ctx,
            `/api/v1/locks/${input.projectId}/${encodeURIComponent(input.filePath)}`,
            {
              method: 'POST',
              body: JSON.stringify(userData),
            }
          );
          return result;
        } catch (error) {
          throw new TRPCError({
            code: 'INTERNAL_SERVER_ERROR',
            message: 'Failed to lock file',
            cause: error,
          });
        }
      }),

    // Unlock file (REAL endpoint)
    unlock: protectedProcedure
      .input(z.object({
        projectId: z.string(),
        filePath: z.string(),
      }))
      .mutation(async ({ input, ctx }) => {
        try {
          await makeGoRequest<{ success: boolean }>(
            ctx,
            `/api/v1/locks/${input.projectId}/${encodeURIComponent(input.filePath)}`,
            {
              method: 'DELETE',
            }
          );
          return { success: true };
        } catch (error) {
          throw new TRPCError({
            code: 'INTERNAL_SERVER_ERROR',
            message: 'Failed to unlock file',
            cause: error,
          });
        }
      }),
  }),

  // Team collaboration (REAL endpoints)
  collaboration: router({
    // Get team presence (REAL endpoint)
    presence: protectedProcedure
      .input(z.object({ projectId: z.string() }))
      .query(async ({ input, ctx }) => {
        try {
          const result = await makeGoRequest<{ success: boolean; presence: any[] }>(
            ctx,
            `/api/v1/collaboration/${input.projectId}/presence`
          );
          return result.presence;
        } catch (error) {
          throw new TRPCError({
            code: 'INTERNAL_SERVER_ERROR',
            message: 'Failed to fetch team presence',
            cause: error,
          });
        }
      }),

    // Update user presence (REAL endpoint)
    updatePresence: protectedProcedure
      .input(z.object({
        projectId: z.string(),
        status: z.enum(['online', 'editing', 'idle', 'offline']),
        currentFile: z.string().optional(),
      }))
      .mutation(async ({ input, ctx }) => {
        try {
          const data = {
            user_name: ctx.user.name || 'Unknown',
            status: input.status,
            current_file: input.currentFile || '',
          };
          
          await makeGoRequest<{ success: boolean }>(
            ctx,
            `/api/v1/collaboration/${input.projectId}/presence`,
            {
              method: 'POST',
              body: JSON.stringify(data),
            }
          );
          return { success: true };
        } catch (error) {
          throw new TRPCError({
            code: 'INTERNAL_SERVER_ERROR',
            message: 'Failed to update presence',
            cause: error,
          });
        }
      }),
  }),

  // Analytics (REAL endpoints)
  analytics: router({
    // Get activity feed (REAL endpoint)
    activity: protectedProcedure
      .input(z.object({
        projectId: z.string(),
        limit: z.number().min(1).max(100).default(50),
      }))
      .query(async ({ input, ctx }) => {
        try {
          const result = await makeGoRequest<{ success: boolean; activities: any[] }>(
            ctx,
            `/api/v1/analytics/activity/${input.projectId}?limit=${input.limit}`
          );
          return result.activities;
        } catch (error) {
          throw new TRPCError({
            code: 'INTERNAL_SERVER_ERROR',
            message: 'Failed to fetch activity feed',
            cause: error,
          });
        }
      }),

    // Get productivity metrics (REAL endpoint)
    productivity: protectedProcedure
      .input(z.object({
        projectId: z.string(),
        days: z.number().min(1).max(365).default(7),
      }))
      .query(async ({ input, ctx }) => {
        try {
          const result = await makeGoRequest<any>(
            ctx,
            `/api/v1/analytics/productivity/${input.projectId}?days=${input.days}`
          );
          return result;
        } catch (error) {
          throw new TRPCError({
            code: 'INTERNAL_SERVER_ERROR',
            message: 'Failed to fetch productivity metrics',
            cause: error,
          });
        }
      }),

    // Get team insights (REAL endpoint)
    insights: protectedProcedure
      .input(z.object({
        projectId: z.string(),
        days: z.number().min(1).max(365).default(30),
      }))
      .query(async ({ input, ctx }) => {
        try {
          const result = await makeGoRequest<any>(
            ctx,
            `/api/v1/analytics/insights/${input.projectId}?days=${input.days}`
          );
          return result;
        } catch (error) {
          throw new TRPCError({
            code: 'INTERNAL_SERVER_ERROR',
            message: 'Failed to fetch team insights',
            cause: error,
          });
        }
      }),
  }),
});

// Export type router for client
export type AppRouter = typeof appRouter;