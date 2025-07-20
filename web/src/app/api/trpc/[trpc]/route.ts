// app/api/trpc/[trpc]/route.ts - tRPC API route handler
import { fetchRequestHandler } from '@trpc/server/adapters/fetch';
import { appRouter, createContext } from '@/lib/trpc';

const handler = (req: Request) =>
  fetchRequestHandler({
    endpoint: '/api/trpc',
    req,
    router: appRouter,
    createContext: () => createContext(),
    onError: ({ error, path }) => {
      console.error(`tRPC Error on '${path}':`, error);
    },
  });

export { handler as GET, handler as POST };