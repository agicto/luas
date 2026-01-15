import { NextRequest, NextResponse } from 'next/server';
import { cookies } from 'next/headers';
import { authConfig, mockUsers, generateMockToken } from '@/config/auth';

interface LoginBody {
  email: string;
  password: string;
  remember?: boolean;
}

/**
 * Auth Login API (Mock)
 * 
 * POST /api/auth/login
 * 
 * Authenticates against mock users and issues access token.
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

    // Generate access token
    const accessToken = generateMockToken(user.id, 'access');

    // Set cookie
    const cookieStore = await cookies();
    const maxAge = body.remember ? 604800 : authConfig.accessTokenExpiry;

    cookieStore.set(authConfig.cookies.accessToken, accessToken, {
      httpOnly: true,
      secure: process.env.NODE_ENV === 'production',
      sameSite: 'lax',
      path: '/',
      maxAge,
    });

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
    console.error('Auth login error:', error);
    return NextResponse.json(
      { message: 'Internal server error', code: 'INTERNAL_ERROR' },
      { status: 500 }
    );
  }
}
