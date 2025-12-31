import { NextResponse } from 'next/server';
import { cookies } from 'next/headers';
import { authConfig, decodeMockToken, mockUsers } from '@/config/auth';

/**
 * Auth Me API (Mock)
 * 
 * GET /api/auth/me
 * 
 * Returns current user info based on access token.
 * Response format: { data: { user } }
 */
export async function GET() {
  try {
    const cookieStore = await cookies();
    const accessToken = cookieStore.get(authConfig.cookies.accessToken)?.value;

    if (!accessToken) {
      return NextResponse.json(
        { message: 'Not authenticated', code: 'UNAUTHORIZED' },
        { status: 401 }
      );
    }

    // Decode and validate token
    const decoded = decodeMockToken(accessToken);
    if (!decoded || decoded.type !== 'access') {
      return NextResponse.json(
        { message: 'Invalid or expired token', code: 'INVALID_TOKEN' },
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

    // Return user data in expected format
    const userData = {
      id: user.id,
      email: user.email,
      name: user.name,
      role: user.role,
      avatar: user.avatar || null,
    };

    return NextResponse.json({
      data: { user: userData },
    });

  } catch (error) {
    console.error('Auth me error:', error);
    return NextResponse.json(
      { message: 'Internal server error', code: 'INTERNAL_ERROR' },
      { status: 500 }
    );
  }
}
