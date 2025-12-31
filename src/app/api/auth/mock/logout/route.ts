import { NextResponse } from 'next/server';
import { cookies } from 'next/headers';
import { authConfig } from '@/config/auth';

/**
 * Mock Logout API
 * 
 * POST /api/auth/mock/logout
 * 
 * Clears auth cookies.
 */
export async function POST() {
  try {
    const cookieStore = await cookies();

    // Clear access token cookie
    cookieStore.delete(authConfig.cookies.accessToken);

    // Clear refresh token cookie
    cookieStore.delete(authConfig.cookies.refreshToken);

    return NextResponse.json({ 
      data: { success: true },
    });

  } catch (error) {
    console.error('Mock logout error:', error);
    return NextResponse.json(
      { message: 'Internal server error', code: 'INTERNAL_ERROR' },
      { status: 500 }
    );
  }
}
