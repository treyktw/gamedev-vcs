// app/api/v1/files/[projectId]/[...filePath]/route.ts
import { NextRequest, NextResponse } from 'next/server';

export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ projectId: string; filePath: string[] }> }
) {
  const { projectId, filePath } = await params;
  const fullFilePath = filePath.join('/');
  
  try {
    const response = await fetch(`${process.env.VCS_BACKEND_URL}/api/v1/files/${projectId}/${encodeURIComponent(fullFilePath)}`, {
      headers: {
        'Authorization': request.headers.get('Authorization') || '',
      },
    });

    if (!response.ok) {
      return NextResponse.json(
        { error: 'Failed to fetch file content from backend' },
        { status: response.status }
      );
    }

    // Handle both text and binary files
    const contentType = response.headers.get('content-type');
    if (contentType?.includes('text') || contentType?.includes('json')) {
      const text = await response.text();
      return new NextResponse(text, {
        headers: { 'Content-Type': contentType },
      });
    } else {
      const buffer = await response.arrayBuffer();
      return new NextResponse(buffer, {
        headers: { 'Content-Type': contentType || 'application/octet-stream' },
      });
    }
  } catch (error) {
    console.error('File content API error:', error);
    return NextResponse.json(
      { error: 'Backend not available. File content endpoint not implemented yet.' },
      { status: 503 }
    );
  }
}