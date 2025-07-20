"use client";

import { use } from "react";
import { ClientSideSuspense, LiveblocksProvider, RoomProvider } from "@liveblocks/react/suspense";
import CommitHistory from "@/components/commits/commit-history";

interface PageProps {
  params: Promise<{ projectId: string }>;
}

function CommitsPage({ params }: PageProps) {
  const { projectId } = use(params);

  return (
    <LiveblocksProvider authEndpoint="/api/liveblocks-auth">
      <RoomProvider id={`project-${projectId}`}>
        <ClientSideSuspense fallback={<CommitsLoader />}>
          <div className="h-full">
            <CommitHistory projectId={projectId} />
          </div>
        </ClientSideSuspense>
      </RoomProvider>
    </LiveblocksProvider>
  );
}

function CommitsLoader() {
  return (
    <div className="h-full flex items-center justify-center">
      <div className="flex flex-col items-center gap-4">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
        <p className="text-muted-foreground">Loading commit history...</p>
      </div>
    </div>
  );
}

export default CommitsPage;