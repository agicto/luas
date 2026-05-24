# Luas Web

一个面向平台控制面的 Next.js 控制台，负责 GitHub 连接、仓库导入、服务创建、环境变量管理、部署触发与实时日志展示。

## ✨ Frontend-First AI Features

- 🎨 **Component-Driven**: Extensive UI component library with Radix UI and custom designs
- 🚀 **Performance Optimized**: Next.js 16.2.1 App Router with automatic code splitting
- 🌙 **Theme System**: Beautiful dark/light themes with CSS variables
- 📱 **Mobile-First**: Responsive design for all screen sizes
- 🔍 **TypeScript**: Full type safety and excellent DX (TypeScript 5.9+)
- ⚡ **Hot Reload**: Instant development feedback with Next.js Turbopack
- 🤖 **AI-Ready**: Clean patterns for AI code generation and "vibe coding"
- 🔐 **Auth Integration**: Mock auth routes with httpOnly session cookies and protected demo pages
- 📊 **State Management**: Zustand 5.0 for predictable, granular state handling
- 🌍 **I18n**: Full internationalization support with `next-intl`
- 🎨 **Styleguide Explorer**: Pre-built component gallery and design system playground
- 🛠️ **Developer Tools**: Pre-configured ESLint 9, Prettier, and Vitest

## 🤖 AI Developer Experience

- **AI-Friendly Code Structure**: Clean, predictable patterns that AI tools (like Windsurf, Cursor, Bolt) understand
- **Smart Component Design**: Components designed for AI generation and modification, utilizing Atomic Design principles
- **Type Safety**: Comprehensive TypeScript types for better AI code completion and error prevention
- **Documentation**: Rich JSDoc comments for AI context understanding
- **Error Handling**: Standardized error handling patterns for AI debugging assistance

## 🆕 Latest Updates (v2.1.0)

- ✅ **Next.js 16.2.1** - Current pinned framework version
- ✅ **React 19.2.3** - Full support for React 19 features
- ✅ **Tailwind CSS 4.1.18** - Modern utility-first CSS
- ✅ **Next-Intl 4.6** - Comprehensive i18n solution
- ✅ **Zustand 5.0** - Optimized state management
- ✅ **Architecture Guide** - Comprehensive guide for building scalable AI-ready apps

👉 Check out the [Optimization Summary Report](docs/OPTIMIZATION_SUMMARY.md) for details.

## 🛠️ Frontend-Optimized Tech Stack

- **Framework**: Next.js 16.2.1 (App Router)
- **Library**: React 19.2.3
- **Language**: TypeScript 5.9.3 (Strict Mode)
- **Styling**: Tailwind CSS 4.1.18 + PostCSS
- **UI Components**: Radix UI + Lucide Icons
- **State Management**: Zustand 5.0.9
- **Data Fetching**: TanStack Query v5
- **Validation**: Zod 4.2
- **Theming**: Next-Themes 0.4
- **Testing**: Vitest 4.0 + Testing Library

## 🚀 Quick Start

### Prerequisites

- Node.js 20.11+
- pnpm 10+ (Recommended)

### Installation

1. **Clone the repository**

   ```bash
   git clone https://github.com/zgiai/luas.git
   cd luas/web
   ```

2. **Install dependencies**

   ```bash
   pnpm install
   ```

3. **Set up environment variables**

   ```bash
   cp .env.example .env.local
   # Edit .env.local with your configuration
   ```

   Platform console variables:

   ```bash
   NEXT_PUBLIC_DEPLOY_API_URL=http://localhost:8025/v1
   NEXT_PUBLIC_PLATFORM_API_URL=http://localhost:8025/v1
   ```

4. **Run the development server**

   ```bash
   pnpm dev
   ```

5. **Open your browser**

   Navigate to [http://localhost:3000](http://localhost:3000)

## 📁 Project Structure

```text
src/
├── app/                    # Next.js App Router
│   ├── (auth)/            # Authentication routes
│   ├── (protected)/       # Authenticated route group
│   │   ├── (console)/     # Console shell and business pages
│   │   └── (devtools)/    # Internal demo and playground pages
│   ├── (site)/            # Marketing/Public pages
│   └── api/               # API Route handlers
├── components/            # Shared UI and generic components
│   ├── ui/               # Base UI library (Shadcn-like)
│   ├── features/         # Shared feature-facing UI blocks
│   └── common/           # Shared layout components
├── features/             # Feature-first modules (auth, example, ...)
│   ├── auth/             # components, hooks, services, store, server, types
│   └── example/          # hooks, services, server, types
├── hooks/                # Shared generic hooks only
├── services/             # Compatibility exports for feature services
├── store/                # Shared global stores only
├── i18n/                 # Translation files
├── providers/            # React context providers
├── utils/                # Utility functions
└── types/                # Shared cross-feature types
```

## 📊 Features in Depth

### 🎨 **Styleguide**
A built-in styleguide available at `/styleguide` lives under the protected devtools route group, keeping internal playground pages separate from business pages.

### 🔐 **Authentication**
Complete auth flow out-of-the-box:
- Login/Register pages with validation
- Mock `/api/auth/*` routes with a demo account (`admin@example.com` / `admin123`)
- httpOnly session cookie bootstrap via `AuthProvider`
- `middleware.ts` + `AuthGuard` for protected routes

### 🌍 **Internationalization**
Powered by `next-intl`, supporting:
- Multi-language routing
- Type-safe translation keys
- Dynamic language switching

### 🚀 **Platform Console**
控制台首页现在是平台工作台，提供：
- GitHub Token 连接与仓库浏览
- 从仓库导入服务
- 选择部署目标与部署策略
- 环境变量编辑
- 部署记录与实时日志流
- 发布中心低层操作页

## 🚀 Deployment

### Vercel (Recommended)
Deployment is seamless on Vercel with zero configuration.

### Docker
```bash
docker build -t luas-web .
docker run -p 3000:3000 luas-web
```

## 🧪 Scripts

```bash
pnpm dev          # Start development with Turbopack
pnpm build        # Build for production
pnpm start        # Start production server
pnpm lint         # Run ESLint
pnpm type-check   # Run TypeScript checks
pnpm test         # Run unit tests
pnpm format       # Format with Prettier
```

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

---

Made with ❤️ by the Luas contributors
