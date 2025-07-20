"use client";

import { ClientSideSuspense, LiveblocksProvider, RoomProvider } from "@liveblocks/react/suspense";
import { use } from "react";
import RepositoryBrowser from "@/components/repo/RepositoryBrowser";

interface PageProps {
  params: Promise<{ projectId: string }>;
}

function RepositoryPage({ params }: PageProps) {
  const { projectId } = use(params);

  return (
    <LiveblocksProvider authEndpoint="/api/liveblocks-auth">
      <RoomProvider id={`project-${projectId}`}>
        <ClientSideSuspense fallback={<RepositoryLoader />}>
          <RepositoryBrowser projectId={projectId} />
        </ClientSideSuspense>
      </RoomProvider>
    </LiveblocksProvider>
  );
}

function RepositoryLoader() {
  return (
    <div className="h-screen flex items-center justify-center">
      <div className="flex flex-col items-center gap-4">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
        <p className="text-muted-foreground">Loading repository...</p>
      </div>
    </div>
  );
}

export default RepositoryPage;