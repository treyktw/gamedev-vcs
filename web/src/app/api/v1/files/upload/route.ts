import { NextRequest, NextResponse } from 'next/server';
import { auth } from '@/auth';

export async function POST(request: NextRequest) {
  try {
    // Get the authenticated session
    const session = await auth();
    
    if (!session?.user?.id) {
      return NextResponse.json(
        { error: 'Authentication required' },
        { status: 401 }
      );
    }

    const formData = await request.formData();
    
    // Forward to Go backend with proper authentication
    const response = await fetch(`${process.env.VCS_BACKEND_URL}/api/v1/files/upload`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${session.accessToken || ''}`,
        'X-User-ID': session.user.id,
        'X-User-Name': session.user.name || session.user.email || 'Unknown User',
      },
      body: formData,
    });

    if (!response.ok) {
      return NextResponse.json(
        { error: 'Failed to upload files to backend' },
        { status: response.status }
      );
    }

    const data = await response.json();
    return NextResponse.json(data);
  } catch (error) {
    console.error('File upload API error:', error);
    return NextResponse.json(
      { error: 'Backend not available. File upload endpoint not implemented yet.' },
      { status: 503 }
    );
  }
}