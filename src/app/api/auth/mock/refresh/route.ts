import { NextRequest, NextResponse } from 'next/server';
import { cookies } from 'next/headers';
import {
  authConfig,
  mockUsers,
  generateMockToken,
  decodeMockToken,
  isRefreshModeEnabled,
} from '@/config/auth';

interface RefreshBody {
  refresh_token?: string;
}

/**
 * Mock Token Refresh API
 * 
 * POST /api/auth/mock/refresh
 * 
 * Refreshes access token using refresh token.
 * Only available when tokenMode is 'refresh'.
 */
export async function POST(req: NextRequest) {
  try {
    // Check if refresh mode is enabled
    if (!isRefreshModeEnabled()) {
      return NextResponse.json(
        { message: 'Token refresh is not enabled in basic mode', code: 'REFRESH_DISABLED' },
        { status: 400 }
      );
    }

    // Get refresh token from body or cookie
    const body = (await req.json().catch(() => ({}))) as RefreshBody;
    const cookieStore = await cookies();
    const refreshToken = 
      body.refresh_token || 
      cookieStore.get(authConfig.cookies.refreshToken)?.value;

    if (!refreshToken) {
      return NextResponse.json(
        { message: 'Refresh token is required', code: 'MISSING_TOKEN' },
        { status: 400 }
      );
    }

    // Decode and validate refresh token
    const decoded = decodeMockToken(refreshToken);
    if (!decoded || decoded.type !== 'refresh') {
      return NextResponse.json(
        { message: 'Invalid or expired refresh token', code: 'INVALID_TOKEN' },
        { status: 401 }
      );
    }

    // Find user
    const user = mockUsers.find((u) => u.id === decoded.userId);
    if (!user) {
      return NextResponse.json(
        { message: 'User not found', code: 'USER_NOT_FOUND' },
        { status: 401 }
      );
    }

    // Generate new tokens
    const newAccessToken = generateMockToken(user.id, 'access');
    const newRefreshToken = generateMockToken(user.id, 'refresh');

    // Update cookies
    cookieStore.set(authConfig.cookies.accessToken, newAccessToken, {
      httpOnly: true,
      secure: process.env.NODE_ENV === 'production',
      sameSite: 'lax',
      path: '/',
      maxAge: authConfig.accessTokenExpiry,
    });

    cookieStore.set(authConfig.cookies.refreshToken, newRefreshToken, {
      httpOnly: true,
      secure: process.env.NODE_ENV === 'production',
      sameSite: 'lax',
      path: '/',
      maxAge: authConfig.refreshTokenExpiry,
    });

    return NextResponse.json({
      data: {
        access_token: newAccessToken,
        refresh_token: newRefreshToken,
        expires_in: authConfig.accessTokenExpiry,
        token_type: 'Bearer',
      },
    }, { status: 200 });

  } catch (error) {
    console.error('Mock refresh error:', error);
    return NextResponse.json(
      { message: 'Internal server error', code: 'INTERNAL_ERROR' },
      { status: 500 }
    );
  }
}
