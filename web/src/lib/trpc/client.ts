// lib/trpc/client.ts - tRPC client configuration
'use client';

import { createTRPCReact } from '@trpc/react-query';
import { httpBatchLink, loggerLink } from '@trpc/client';
import { QueryClient } from '@tanstack/react-query';
import type { AppRouter } from './index';

// Create tRPC React client
export const trpc = createTRPCReact<AppRouter>();

// Create query client with default options
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 5 * 60 * 1000, // 5 minutes
      gcTime: 10 * 60 * 1000, // 10 minutes (formerly cacheTime)
      retry: (failureCount, error: any) => {
        // Don't retry on auth errors
        if (error?.data?.code === 'UNAUTHORIZED') {
          return false;
        }
        return failureCount < 3;
      },
    },
    mutations: {
      retry: false,
    },
  },
});

// Create tRPC client
export const trpcClient = trpc.createClient({
  links: [
    // Logger in development
    ...(process.env.NODE_ENV === 'development'
      ? [
          loggerLink({
            enabled: () => true,
          }),
        ]
      : []),
    // HTTP batch link
    httpBatchLink({
      url: '/api/trpc',
      async headers() {
        return {
          // Add any additional headers here
        };
      },
    }),
  ],
});

// Custom hooks for common patterns
export function useTRPCErrorHandler() {
  return (error: any) => {
    if (error?.data?.code === 'UNAUTHORIZED') {
      // Redirect to login
      window.location.href = '/auth/signin';
    } else {
      // Log error for debugging
      console.error('tRPC Error:', error);
    }
  };
}

// Optimistic updates helper
export function useOptimisticUpdate() {
  const utils = trpc.useUtils();
  
  return {
    // Optimistically update project list
    updateProject: (projectId: string, updates: any) => {
      utils.projects.list.setData(undefined, (old) => {
        if (!old) return old;
        return old.map(project => 
          project.id === projectId 
            ? { ...project, ...updates }
            : project
        );
      });
    },
    
    // Optimistically add new project
    addProject: (newProject: any) => {
      utils.projects.list.setData(undefined, (old) => {
        if (!old) return [newProject];
        return [newProject, ...old];
      });
    },
    
    // Optimistically remove project
    removeProject: (projectId: string) => {
      utils.projects.list.setData(undefined, (old) => {
        if (!old) return old;
        return old.filter(project => project.id !== projectId);
      });
    },
    
    // Invalidate queries
    invalidateProjects: () => utils.projects.list.invalidate(),
    invalidateProject: (projectId: string) => utils.projects.byId.invalidate({ id: projectId }),
  };
}

// Real-time subscription helpers
export function useRealtimeSubscription(projectId: string) {
  const utils = trpc.useUtils();
  
  // Periodically refetch presence and activity
  const { data: presence } = trpc.collaboration.presence.useQuery(
    { projectId },
    {
      refetchInterval: 5000, // 5 seconds
      enabled: !!projectId,
    }
  );
  
  const { data: activity } = trpc.analytics.activity.useQuery(
    { projectId, limit: 20 },
    {
      refetchInterval: 10000, // 10 seconds
      enabled: !!projectId,
    }
  );
  
  return { presence, activity };
}

// File upload with progress
export function useFileUploadWithProgress() {
  const uploadFile = async (
    projectId: string, 
    file: File, 
    onProgress?: (progress: number) => void
  ) => {
    // Create FormData for file upload
    const formData = new FormData();
    formData.append('file', file);
    formData.append('project_id', projectId);
    
    // Upload directly to backend (bypass tRPC for file uploads)
    const xhr = new XMLHttpRequest();
    
    return new Promise((resolve, reject) => {
      xhr.upload.addEventListener('progress', (e) => {
        if (e.lengthComputable && onProgress) {
          onProgress((e.loaded / e.total) * 100);
        }
      });
      
      xhr.addEventListener('load', () => {
        if (xhr.status === 200) {
          resolve(JSON.parse(xhr.responseText));
        } else {
          reject(new Error(`Upload failed: ${xhr.statusText}`));
        }
      });
      
      xhr.addEventListener('error', () => {
        reject(new Error('Upload failed'));
      });
      
      xhr.open('POST', `${process.env.NEXT_PUBLIC_VCS_API_URL}/api/v1/files/upload?project=${projectId}`);
      xhr.send(formData);
    });
  };
  
  return {
    uploadFile,
    isLoading: false, // No tRPC mutation state
    error: null,
  };
}

// Batch operations helper
export function useBatchOperations() {
  const utils = trpc.useUtils();
  
  return {
    // Batch lock multiple files
    lockFiles: async (projectId: string, filePaths: string[]) => {
      const promises = filePaths.map(filePath =>
        utils.client.locks.lock.mutate({ projectId, filePath })
      );
      
      return Promise.allSettled(promises);
    },
    
    // Batch unlock multiple files
    unlockFiles: async (projectId: string, filePaths: string[]) => {
      const promises = filePaths.map(filePath =>
        utils.client.locks.unlock.mutate({ projectId, filePath })
      );
      
      return Promise.allSettled(promises);
    },
  };
}