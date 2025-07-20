// app/dashboard/[projectId]/activity/page.tsx
"use client";

import { use } from "react";
import { ClientSideSuspense, LiveblocksProvider, RoomProvider } from "@liveblocks/react/suspense";
import ActivityFeed from "@/components/activity/activity-feed";

interface PageProps {
  params: Promise<{ projectId: string }>;
}

function ActivityPage({ params }: PageProps) {
  const { projectId } = use(params);

  return (
    <LiveblocksProvider authEndpoint="/api/liveblocks-auth">
      <RoomProvider id={`project-${projectId}`}>
        <ClientSideSuspense fallback={<ActivityLoader />}>
          <div className="h-full p-6">
            <ActivityFeed projectId={projectId} limit={100} showLiveActivity={true} />
          </div>
        </ClientSideSuspense>
      </RoomProvider>
    </LiveblocksProvider>
  );
}

function ActivityLoader() {
  return (
    <div className="h-full flex items-center justify-center">
      <div className="flex flex-col items-center gap-4">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
        <p className="text-muted-foreground">Loading activity feed...</p>
      </div>
    </div>
  );
}

export default ActivityPage;