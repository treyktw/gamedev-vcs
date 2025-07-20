import { NextRequest, NextResponse } from 'next/server'
import { authRateLimit } from '@/lib/redis'

export async function withRateLimit(
  request: NextRequest,
  rateLimit: typeof authRateLimit,
  errorMessage = 'Too many requests'
) {
  const ip = request.headers.get('x-forwarded-for') || 
    request.headers.get('x-forwarded-for') || 
    request.headers.get('x-real-ip') || 
    'unknown'

  const { success, limit, reset, remaining } = await rateLimit.limit(ip)

  if (!success) {
    return NextResponse.json(
      { 
        error: errorMessage,
        retryAfter: Math.round((reset - Date.now()) / 10)
      },
      { 
        status: 429,
        headers: {
          'X-RateLimit-Limit': limit.toString(),
          'X-RateLimit-Remaining': remaining.toString(),
          'X-RateLimit-Reset': reset.toString(),
          'Retry-After': Math.round((reset - Date.now()) / 10).toString()
        }
      }
    )
  }

  return null // Continue
}