import { NextRequest, NextResponse } from 'next/server';

export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ projectId: string }> }
) {
  const { projectId } = await params;
  
  try {
    const response = await fetch(`${process.env.VCS_BACKEND_URL}/api/v1/projects/${projectId}`, {
      headers: {
        'Authorization': request.headers.get('Authorization') || '',
        'X-User-ID': request.headers.get('X-User-ID') || '2c974a19-41b6-4236-a1b3-2f0bd23f363a',
        'X-User-Name': request.headers.get('X-User-Name') || 'CLI User',
        'Content-Type': 'application/json',
      },
    });

    if (!response.ok) {
      return NextResponse.json(
        { error: 'Failed to fetch project from backend' },
        { status: response.status }
      );
    }

    const data = await response.json();
    return NextResponse.json(data);
  } catch (error) {
    console.error('Project API error:', error);
    return NextResponse.json(
      { error: 'Backend not available. Please ensure Go server is running on port 8080.' },
      { status: 503 }
    );
  }
}