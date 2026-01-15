import { NextRequest, NextResponse } from 'next/server';
import { cookies } from 'next/headers';
import { authConfig, generateMockToken } from '@/config/auth';

interface SetupBody {
  email: string;
  name: string;
  password: string;
}

/**
 * Auth Setup API (Mock)
 * 
 * POST /api/auth/setup
 * 
 * Initial system setup - creates the first admin user.
 * In a real app, this would only work once during initial setup.
 */
export async function POST(req: NextRequest) {
  try {
    const body = (await req.json()) as SetupBody;

    if (!body?.email || !body?.name || !body?.password) {
      return NextResponse.json(
        { message: 'Email, name, and password are required', code: 'VALIDATION_ERROR' },
        { status: 400 }
      );
    }

    // Create admin user (in real app, this would save to database)
    const adminUser = {
      id: `admin-${Date.now()}`,
      email: body.email,
      name: body.name,
      role: 'admin' as const,
    };

    // Generate access token and set cookie
    const accessToken = generateMockToken(adminUser.id, 'access');
    const cookieStore = await cookies();

    cookieStore.set(authConfig.cookies.accessToken, accessToken, {
      httpOnly: true,
      secure: process.env.NODE_ENV === 'production',
      sameSite: 'lax',
      path: '/',
      maxAge: authConfig.accessTokenExpiry,
    });

    return NextResponse.json({
      data: { user: adminUser },
    }, { status: 200 });

  } catch (error) {
    console.error('Auth setup error:', error);
    return NextResponse.json(
      { message: 'Internal server error', code: 'INTERNAL_ERROR' },
      { status: 500 }
    );
  }
}
