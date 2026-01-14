import { NextResponse } from 'next/server';

/**
 * System Features API (Mock)
 * 
 * GET /api/auth/system-features
 * 
 * Returns available system features/flags.
 * In a real app, this would return feature toggles from config/database.
 */
export async function GET() {
  // Mock: Return default feature flags
  return NextResponse.json({
    data: {
      sso_enforced_for_signin: false,
      sso_enforced_for_signin_protocol: '',
      sso_enforced_for_web: false,
      sso_enforced_for_web_protocol: '',
      enable_marketplace: true,
      enable_web_sso_switch_component: false,
      enable_email_code_login: false,
      enable_email_password_login: true,
      enable_social_oauth_login: false,
      is_allow_create_workspace: true,
      is_allow_register: true,
      license: {
        status: 'active',
        expired_at: null,
      },
    },
  });
}
