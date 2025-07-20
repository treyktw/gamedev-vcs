import { NextRequest, NextResponse } from 'next/server';
import { auth } from '@/auth';

export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ projectId: string }> }
) {
  const { projectId } = await params;
  
  try {
    // Get the authenticated session
    const session = await auth();
    
    if (!session?.user?.id) {
      return NextResponse.json(
        { error: 'Authentication required' },
        { status: 401 }
      );
    }

    const response = await fetch(`${process.env.VCS_BACKEND_URL}/api/v1/projects/${projectId}/members`, {
      headers: {
        'Authorization': `Bearer ${session.accessToken || ''}`,
        'X-User-ID': session.user.id,
        'X-User-Name': session.user.name || session.user.email || 'Unknown User',
        'Content-Type': 'application/json',
      },
    });

    if (!response.ok) {
      return NextResponse.json(
        { error: 'Failed to fetch team members from backend' },
        { status: response.status }
      );
    }

    const data = await response.json();
    return NextResponse.json(data);
  } catch (error) {
    console.error('Team members API error:', error);
    return NextResponse.json(
      { error: 'Backend not available. Team members endpoint not implemented yet.' },
      { status: 503 }
    );
  }
}