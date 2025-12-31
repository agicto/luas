import { NextRequest, NextResponse } from 'next/server';
import { cookies } from 'next/headers';
import { 
  authConfig, 
  mockUsers, 
  generateMockToken, 
  isRefreshModeEnabled 
} from '@/config/auth';

interface LoginBody {
  email: string;
  password: string;
  remember?: boolean;
}

/**
 * Mock Login API
 * 
 * POST /api/auth/mock/login
 * 
 * Authenticates against mock users and issues tokens.
 * Supports both 'basic' and 'refresh' token modes.
 * 
 * Response format matches what auth service expects:
 * { data: { user: User } }
 */
export async function POST(req: NextRequest) {
  try {
    const body = (await req.json()) as LoginBody;

    if (!body?.email || !body?.password) {
      return NextResponse.json(
        { message: 'Email and password are required', code: 'VALIDATION_ERROR' },
        { status: 400 }
      );
    }

    // Find mock user
    const user = mockUsers.find(
      (u) => u.email === body.email && u.password === body.password
    );

    if (!user) {
      return NextResponse.json(
        { message: 'Invalid email or password', code: 'INVALID_CREDENTIALS' },
        { status: 401 }
      );
    }

    // Generate tokens
    const accessToken = generateMockToken(user.id, 'access');
    const refreshToken = isRefreshModeEnabled() 
      ? generateMockToken(user.id, 'refresh') 
      : undefined;

    // Set cookies
    const cookieStore = await cookies();
    const maxAge = body.remember 
      ? authConfig.refreshTokenExpiry 
      : authConfig.accessTokenExpiry;

    cookieStore.set(authConfig.cookies.accessToken, accessToken, {
      httpOnly: true,
      secure: process.env.NODE_ENV === 'production',
      sameSite: 'lax',
      path: '/',
      maxAge,
    });

    if (refreshToken) {
      cookieStore.set(authConfig.cookies.refreshToken, refreshToken, {
        httpOnly: true,
        secure: process.env.NODE_ENV === 'production',
        sameSite: 'lax',
        path: '/',
        maxAge: authConfig.refreshTokenExpiry,
      });
    }

    // Return user data in expected format: { data: { user } }
    const userData = {
      id: user.id,
      email: user.email,
      name: user.name,
      role: user.role,
      avatar: user.avatar || null,
    };

    return NextResponse.json({ 
      data: { user: userData },
    }, { status: 200 });

  } catch (error) {
    console.error('Mock login error:', error);
    return NextResponse.json(
      { message: 'Internal server error', code: 'INTERNAL_ERROR' },
      { status: 500 }
    );
  }
}
