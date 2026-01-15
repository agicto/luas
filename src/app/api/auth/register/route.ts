import { NextRequest, NextResponse } from 'next/server';
import { cookies } from 'next/headers';
import { authConfig, mockUsers, generateMockToken } from '@/config/auth';

interface RegisterBody {
  email: string;
  name: string;
  password: string;
}

/**
 * Auth Register API (Mock)
 * 
 * POST /api/auth/register
 * 
 * Creates a new mock user and returns success.
 * In a real app, this would create a user in the database.
 */
export async function POST(req: NextRequest) {
  try {
    const body = (await req.json()) as RegisterBody;

    if (!body?.email || !body?.name || !body?.password) {
      return NextResponse.json(
        { message: 'Email, name, and password are required', code: 'VALIDATION_ERROR' },
        { status: 400 }
      );
    }

    // Check if email already exists in mock users
    const existingUser = mockUsers.find((u) => u.email === body.email);
    if (existingUser) {
      return NextResponse.json(
        { message: 'Email already registered', code: 'EMAIL_EXISTS' },
        { status: 409 }
      );
    }

    // Create new mock user (in real app, this would save to database)
    const newUser = {
      id: `mock-user-${Date.now()}`,
      email: body.email,
      name: body.name,
      role: 'user' as const,
    };

    // Generate access token and set cookie (auto-login after registration)
    const accessToken = generateMockToken(newUser.id, 'access');
    const cookieStore = await cookies();

    cookieStore.set(authConfig.cookies.accessToken, accessToken, {
      httpOnly: true,
      secure: process.env.NODE_ENV === 'production',
      sameSite: 'lax',
      path: '/',
      maxAge: authConfig.accessTokenExpiry,
    });

    return NextResponse.json({
      data: { user: newUser },
    }, { status: 201 });

  } catch (error) {
    console.error('Auth register error:', error);
    return NextResponse.json(
      { message: 'Internal server error', code: 'INTERNAL_ERROR' },
      { status: 500 }
    );
  }
}
