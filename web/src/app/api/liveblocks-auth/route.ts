//app/api/liveblocks-auth/route.ts - FIXED VERSION

import { Liveblocks } from "@liveblocks/node";
import { auth } from "@/auth";

const liveblocks = new Liveblocks({
  secret: "sk_dev_F9ccrfBHyvnkS5uB0OmZLA2PHz9GH-mKFqiBAhblz53GapAuOmprSsQF_stPhsQ7",
});

async function getUserFromDB(request: Request) {
  const session = await auth();
  
  if (!session?.user?.id) {
    throw new Error("Unauthorized");
  }

  return {
    id: session.user.id,
    organization: "default", // Use a default organization for all users
    group: "team", // Default group for team access
    metadata: {
      name: session.user.name || "Anonymous User",
      email: session.user.email || "user@example.com",
      image: session.user.image || null,
    }
  };
}

export async function POST(request: Request) {
  try {
    // Get the current user from your database
    const user = await getUserFromDB(request);

    // Start an auth session inside your endpoint
    const session = liveblocks.prepareSession(
      user.id,
      { userInfo: user.metadata } // Optional
    );

    // FIXED: Grant access to all project rooms for authenticated users
    // Pattern: project-{projectId} rooms get full access
    session.allow("project-*", session.FULL_ACCESS);
    
    // Also allow access with the organization pattern (fallback)
    session.allow(`${user.organization}:*`, session.READ_ACCESS);
    session.allow(`${user.organization}:${user.group}:*`, session.FULL_ACCESS);

    // Authorize the user and return the result
    const { status, body } = await session.authorize();
    return new Response(body, { status });
  } catch (error) {
    console.error("Liveblocks auth error:", error);
    
    // Return unauthorized for any authentication errors
    return new Response("Unauthorized", { status: 401 });
  }
}