import { NextRequest, NextResponse } from 'next/server';

export async function POST(
  request: NextRequest,
  { params }: { params: Promise<{ projectId: string; filePath: string[] }> }
) {
  const { projectId, filePath } = await params;
  const fullFilePath = filePath.join('/');
  
  try {
    const body = await request.json();
    const response = await fetch(`${process.env.VCS_BACKEND_URL}/api/v1/locks/${projectId}/${encodeURIComponent(fullFilePath)}`, {
      method: 'POST',
      headers: {
        'Authorization': request.headers.get('Authorization') || '',
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(body),
    });

    if (!response.ok) {
      return NextResponse.json(
        { error: 'Failed to lock file' },
        { status: response.status }
      );
    }

    const data = await response.json();
    return NextResponse.json(data);
  } catch (error) {
    console.error('File lock API error:', error);
    return NextResponse.json(
      { error: 'Backend not available. File locking endpoint not implemented yet.' },
      { status: 503 }
    );
  }
}

export async function DELETE(
  request: NextRequest,
  { params }: { params: Promise<{ projectId: string; filePath: string[] }> }
) {
  const { projectId, filePath } = await params;
  const fullFilePath = filePath.join('/');
  
  try {
    const response = await fetch(`${process.env.VCS_BACKEND_URL}/api/v1/locks/${projectId}/${encodeURIComponent(fullFilePath)}`, {
      method: 'DELETE',
      headers: {
        'Authorization': request.headers.get('Authorization') || '',
        'Content-Type': 'application/json',
      },
    });

    if (!response.ok) {
      return NextResponse.json(
        { error: 'Failed to unlock file' },
        { status: response.status }
      );
    }

    const data = await response.json();
    return NextResponse.json(data);
  } catch (error) {
    console.error('File unlock API error:', error);
    return NextResponse.json(
      { error: 'Backend not available. File unlocking endpoint not implemented yet.' },
      { status: 503 }
    );
  }
}
