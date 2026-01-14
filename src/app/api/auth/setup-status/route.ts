import { NextResponse } from 'next/server';

/**
 * Auth Setup Status API (Mock)
 * 
 * GET /api/auth/setup-status
 * 
 * Returns the current setup status.
 * In a real app, this would check if initial setup has been completed.
 */
export async function GET() {
  // Mock: Always return that setup is finished
  // In a real app, this would check a database flag
  return NextResponse.json({
    data: {
      step: 'finished',
    },
  });
}
