'use client';

import { useState } from 'react';
import { QueryClientProvider } from '@tanstack/react-query';
import { ReactQueryDevtools } from '@tanstack/react-query-devtools';
import { trpc, trpcClient, queryClient } from '@/lib/trpc/client';

export function TRPCProvider({ children }: { children: React.ReactNode }) {
  const [client] = useState(() => trpcClient);
  const [query] = useState(() => queryClient);

  return (
    <trpc.Provider client={client} queryClient={query}>
      <QueryClientProvider client={query}>
        {children}
        {process.env.NODE_ENV === 'development' && (
          <ReactQueryDevtools initialIsOpen={false} />
        )}
      </QueryClientProvider>
    </trpc.Provider>
  );
}