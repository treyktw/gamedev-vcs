// app/api/v1/locks/[projectId]/route.ts
import { NextRequest, NextResponse } from 'next/server';

export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ projectId: string }> }
) {
  const { projectId } = await params;
  
  try {
    const response = await fetch(`${process.env.VCS_BACKEND_URL}/api/v1/locks/${projectId}`, {
      headers: {
        'Authorization': request.headers.get('Authorization') || '',
        'X-User-ID': request.headers.get('X-User-ID') || '2c974a19-41b6-4236-a1b3-2f0bd23f363a',
        'X-User-Name': request.headers.get('X-User-Name') || 'CLI User',
        'Content-Type': 'application/json',
      },
    });

    if (!response.ok) {
      return NextResponse.json(
        { error: 'Failed to fetch file locks from backend' },
        { status: response.status }
      );
    }

    const data = await response.json();
    return NextResponse.json(data);
  } catch (error) {
    console.error('File locks API error:', error);
    return NextResponse.json(
      { error: 'Backend not available. File locks endpoint not implemented yet.' },
      { status: 503 }
    );
  }
}