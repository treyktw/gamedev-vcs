// app/dashboard/[projectId]/team/page.tsx
"use client";

import { use } from "react";
import { ClientSideSuspense, LiveblocksProvider, RoomProvider } from "@liveblocks/react/suspense";
import TeamManagement from "@/components/team/team-management";

interface PageProps {
  params: Promise<{ projectId: string }>;
}

function TeamPage({ params }: PageProps) {
  const { projectId } = use(params);

  return (
    <LiveblocksProvider authEndpoint="/api/liveblocks-auth">
      <RoomProvider id={`project-${projectId}`}>
        <ClientSideSuspense fallback={<TeamLoader />}>
          <div className="h-full p-6">
            <TeamManagement projectId={projectId} />
          </div>
        </ClientSideSuspense>
      </RoomProvider>
    </LiveblocksProvider>
  );
}

function TeamLoader() {
  return (
    <div className="h-full flex items-center justify-center">
      <div className="flex flex-col items-center gap-4">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
        <p className="text-muted-foreground">Loading team...</p>
      </div>
    </div>
  );
}

export default TeamPage;
