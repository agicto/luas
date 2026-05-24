import Link from 'next/link';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { 
  ArrowRight, 
  Shield, 
  Globe, 
  Terminal, 
  Sparkles, 
  Palette,
  Atom,
  Database,
  ShieldCheck
} from 'lucide-react';

/**
 * Luas Homepage
 * Unified design language with Auth and Console sections
 */
export default function HomePage() {
  return (
    <div className="flex flex-col">
      {/* Hero Section - with glow effects like Auth */}
      <section className="relative overflow-hidden bg-bg-canvas">
        {/* Animated Background Elements */}
        <div className="absolute inset-0 pointer-events-none">
          {/* Large gradient orbs */}
          <div className="absolute top-[-20%] left-[-10%] w-[50%] h-[50%] rounded-full bg-primary/15 blur-3xl" />
          <div className="absolute bottom-[-30%] right-[-15%] w-[60%] h-[60%] rounded-full bg-primary-deeper/10 blur-3xl" />
          <div className="absolute top-[30%] right-[20%] w-[30%] h-[30%] rounded-full bg-primary/5 blur-2xl" />
          
          {/* Decorative circles */}
          <div className="absolute top-[15%] right-[25%] w-32 h-32 rounded-full border border-primary/10" />
          <div className="absolute top-[18%] right-[27%] w-24 h-24 rounded-full border border-primary/5" />
          <div className="absolute bottom-[20%] left-[8%] w-40 h-40 rounded-full border border-primary/10" />
          
          {/* Grid Pattern */}
          <div 
            className="absolute inset-0 opacity-[0.02]"
            style={{
              backgroundImage: `linear-gradient(var(--primary) 1px, transparent 1px), linear-gradient(90deg, var(--primary) 1px, transparent 1px)`,
              backgroundSize: '60px 60px',
            }}
          />
        </div>

        <div className="container relative py-20 md:py-28 lg:py-36">
          <div className="mx-auto max-w-4xl text-center">
            {/* Badge */}
            <Badge variant="secondary" className="mb-6 px-4 py-1.5 text-sm font-medium">
              <Sparkles className="mr-1.5 h-3.5 w-3.5" />
              AI-First Frontend Scaffold
            </Badge>

            {/* Title */}
            <h1 className="text-4xl font-bold tracking-tight md:text-5xl lg:text-6xl">
              Build Modern Web Apps{' '}
              <span className="bg-linear-to-r from-primary to-primary-deeper bg-clip-text text-transparent">
                with AI Power
              </span>
            </h1>

            {/* Description */}
            <p className="mt-6 text-lg text-text-muted md:text-xl max-w-2xl mx-auto">
              A production-ready Next.js 16 scaffold with TypeScript, Tailwind CSS 4, 
              premium UI components, and everything you need for AI-assisted development.
            </p>

            {/* CTA Buttons */}
            <div className="mt-10 flex flex-wrap items-center justify-center gap-4">
              <Link href="/register">
                <Button size="lg" className="h-12 px-8 gap-2 shadow-button-primary">
                  Get Started
                  <ArrowRight className="h-4 w-4" />
                </Button>
              </Link>
              <Link href="/console">
                <Button variant="outline" size="lg" className="h-12 px-8">
                  View Demo
                </Button>
              </Link>
            </div>
          </div>
        </div>
      </section>

      {/* Features Section */}
      <section className="py-20 md:py-28 bg-bg-subtle/30">
        <div className="container">
          <div className="text-center mb-12">
            <h2 className="text-3xl font-bold tracking-tight md:text-4xl">
              Everything You Need
            </h2>
            <p className="mt-4 text-text-muted max-w-2xl mx-auto">
              A complete scaffold with authentication, dashboard, i18n, and premium UI components.
            </p>
          </div>

          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
            {featuresData.map((feature, index) => (
              <Card 
                key={index} 
                className="group bg-bg-surface border-border/50 hover:shadow-premium hover:-translate-y-1 transition-all duration-300"
              >
                <CardContent className="p-6">
                  <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-primary/10 text-primary group-hover:bg-primary group-hover:text-primary-foreground transition-all duration-300">
                    <feature.icon className="h-6 w-6" />
                  </div>
                  <h3 className="mt-4 text-lg font-semibold">{feature.title}</h3>
                  <p className="mt-2 text-sm text-text-muted leading-relaxed">
                    {feature.description}
                  </p>
                </CardContent>
              </Card>
            ))}
          </div>
        </div>
      </section>

      {/* Tech Stack Section */}
      <section className="py-20 md:py-28">
        <div className="container">
          <div className="text-center mb-12">
            <h2 className="text-3xl font-bold tracking-tight md:text-4xl">
              Modern Tech Stack
            </h2>
            <p className="mt-4 text-text-muted">
              Built with the latest technologies for optimal developer experience.
            </p>
          </div>

          <div className="flex flex-wrap items-center justify-center gap-6 md:gap-10">
            {techStackData.map((tech, index) => (
              <div 
                key={index}
                className="flex items-center gap-3 px-5 py-3 rounded-xl bg-bg-surface border border-border/50 hover:border-primary/30 hover:shadow-md transition-all duration-300"
              >
                {tech.icon}
                <span className="font-medium">{tech.name}</span>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* CTA Section - Using primary gradient like Auth */}
      <section className="relative overflow-hidden bg-linear-to-br from-primary via-primary/95 to-primary-deeper py-20 md:py-28">
        {/* Background decorations */}
        <div className="absolute inset-0 pointer-events-none">
          <div className="absolute top-[-20%] right-[-10%] w-[40%] h-[40%] rounded-full bg-white/10 blur-3xl" />
          <div className="absolute bottom-[-20%] left-[-10%] w-[30%] h-[30%] rounded-full bg-primary-deeper/30 blur-2xl" />
        </div>

        <div className="container relative text-center text-white">
          <h2 className="text-3xl font-bold tracking-tight md:text-4xl lg:text-5xl">
            Ready to Build?
          </h2>
          <p className="mt-4 text-white/80 max-w-xl mx-auto md:text-lg">
            Start building your next project with Luas. 
            Authentication, dashboard, and premium components included.
          </p>
          <div className="mt-10 flex flex-wrap items-center justify-center gap-4">
            <Link href="/register">
              <Button size="lg" variant="secondary" className="h-12 px-8 gap-2">
                Get Started Free
                <ArrowRight className="h-4 w-4" />
              </Button>
            </Link>
            <a 
              href="https://github.com/zgiai/luas" 
              target="_blank" 
              rel="noopener noreferrer"
            >
              <Button 
                size="lg" 
                variant="outline" 
                className="h-12 px-8 bg-transparent border-white/30 text-white hover:bg-white/10 hover:text-white"
              >
                View on GitHub
              </Button>
            </a>
          </div>
        </div>
      </section>
    </div>
  );
}

// Features data
const featuresData = [
  {
    title: 'Authentication',
    description: 'Complete auth flow with login, registration, and protected routes.',
    icon: Shield,
  },
  {
    title: 'Admin Console',
    description: 'Beautiful dashboard with charts, tables, and data visualization.',
    icon: Terminal,
  },
  {
    title: 'Internationalization',
    description: 'Built-in i18n support with type-safe translations.',
    icon: Globe,
  },
  {
    title: 'Premium UI',
    description: 'Elegant components with glassmorphism and micro-animations.',
    icon: Palette,
  },
];

// Tech stack data
const techStackData = [
  {
    name: 'Next.js 16',
    icon: (
      <svg viewBox="0 0 24 24" className="h-6 w-6" fill="currentColor">
        <path d="M11.572 0c-.176 0-.31.001-.358.007a19.76 19.76 0 01-.364.033C7.443.346 4.25 2.185 2.228 5.012a11.875 11.875 0 00-2.119 5.243c-.096.659-.108.854-.108 1.747s.012 1.089.108 1.748c.652 4.506 3.86 8.292 8.209 9.695.779.25 1.6.422 2.534.525.363.04 1.935.04 2.299 0 1.611-.178 2.977-.577 4.323-1.264.207-.106.247-.134.219-.158-.02-.013-.9-1.193-1.955-2.62l-1.919-2.592-2.404-3.558a338.739 338.739 0 00-2.422-3.556c-.009-.002-.018 1.579-.023 3.51-.007 3.38-.01 3.515-.052 3.595a.426.426 0 01-.206.214c-.075.037-.14.044-.495.044H7.81l-.108-.068a.438.438 0 01-.157-.171l-.05-.106.006-4.703.007-4.705.072-.092a.645.645 0 01.174-.143c.096-.047.134-.051.54-.051.478 0 .558.018.682.154.035.038 1.337 1.999 2.895 4.361a10760.433 10760.433 0 004.735 7.17l1.9 2.879.096-.063a12.317 12.317 0 002.466-2.163 11.944 11.944 0 002.824-6.134c.096-.66.108-.854.108-1.748 0-.893-.012-1.088-.108-1.747-.652-4.506-3.859-8.292-8.208-9.695a12.597 12.597 0 00-2.499-.523A33.119 33.119 0 0011.573 0z"/>
      </svg>
    ),
  },
  {
    name: 'React 19',
    icon: <Atom className="h-6 w-6" />,
  },
  {
    name: 'TypeScript',
    icon: (
      <svg viewBox="0 0 24 24" className="h-6 w-6" fill="currentColor">
        <path d="M1.125 0C.502 0 0 .502 0 1.125v21.75C0 23.498.502 24 1.125 24h21.75c.623 0 1.125-.502 1.125-1.125V1.125C24 .502 23.498 0 22.875 0zm17.363 9.75c.612 0 1.154.037 1.627.111a6.38 6.38 0 0 1 1.306.34v2.458a3.95 3.95 0 0 0-.643-.361 5.093 5.093 0 0 0-.717-.26 5.453 5.453 0 0 0-1.426-.2c-.3 0-.573.028-.819.086a2.1 2.1 0 0 0-.623.242c-.17.104-.3.229-.393.374a.888.888 0 0 0-.14.49c0 .196.053.373.156.529.104.156.252.304.443.444s.423.276.696.41c.273.135.582.274.926.416.47.197.892.407 1.266.628.374.222.695.473.963.753.268.279.472.598.614.957.142.359.214.776.214 1.253 0 .657-.125 1.21-.373 1.656a3.033 3.033 0 0 1-1.012 1.085 4.38 4.38 0 0 1-1.487.596c-.566.12-1.163.18-1.79.18a9.916 9.916 0 0 1-1.84-.164 5.544 5.544 0 0 1-1.512-.493v-2.63a5.033 5.033 0 0 0 3.237 1.2c.333 0 .624-.03.872-.09.249-.06.456-.144.623-.25.166-.108.29-.234.373-.38a1.023 1.023 0 0 0-.074-1.089 2.12 2.12 0 0 0-.537-.5 5.597 5.597 0 0 0-.807-.444 27.72 27.72 0 0 0-1.007-.436c-.918-.383-1.602-.852-2.053-1.405-.45-.553-.676-1.222-.676-2.005 0-.614.123-1.141.369-1.582.246-.441.58-.804 1.004-1.089a4.494 4.494 0 0 1 1.47-.629 7.536 7.536 0 0 1 1.77-.201zm-15.113.188h9.563v2.166H9.506v9.646H6.789v-9.646H3.375z"/>
      </svg>
    ),
  },
  {
    name: 'Tailwind CSS 4',
    icon: (
      <svg viewBox="0 0 24 24" className="h-6 w-6" fill="currentColor">
        <path d="M12.001,4.8c-3.2,0-5.2,1.6-6,4.8c1.2-1.6,2.6-2.2,4.2-1.8c0.913,0.228,1.565,0.89,2.288,1.624 C13.666,10.618,15.027,12,18.001,12c3.2,0,5.2-1.6,6-4.8c-1.2,1.6-2.6,2.2-4.2,1.8c-0.913-0.228-1.565-0.89-2.288-1.624 C16.337,6.182,14.976,4.8,12.001,4.8z M6.001,12c-3.2,0-5.2,1.6-6,4.8c1.2-1.6,2.6-2.2,4.2-1.8c0.913,0.228,1.565,0.89,2.288,1.624 c1.177,1.194,2.538,2.576,5.512,2.576c3.2,0,5.2-1.6,6-4.8c-1.2,1.6-2.6,2.2-4.2,1.8c-0.913-0.228-1.565-0.89-2.288-1.624 C10.337,13.382,8.976,12,6.001,12z"/>
      </svg>
    ),
  },
  {
    name: 'Zustand',
    icon: <Database className="h-6 w-6" />,
  },
  {
    name: 'React Query',
    icon: (
      <svg viewBox="0 0 24 24" className="h-6 w-6" fill="currentColor">
        <path d="M6.306 14.691c-.834.96 -.76 2.296-.088 3.011 1.232 1.313 4.268 1.106 6.949-.674.27-.179.533-.37.79-.573a13.18 13.18 0 0 1-1.61-.903c-.28.13-.556.247-.835.346-1.725.614-3.422.607-4.528-.177-.477-.339-.804-.844-.678-1.03zm6.234-6.022c-.18-.072-.365-.142-.553-.206a14.007 14.007 0 0 1 .938-1.594c-2.036-3.016-4.758-4.623-6.466-3.657-.912.515-1.311 1.65-1.255 3.054.013.318.046.649.097.987.188-.02.379-.034.572-.043a13.13 13.13 0 0 1 .168-1.238c.12-.577.396-1.129.896-1.412 1.176-.664 3.221.277 4.915 2.347.156.19.307.388.454.595-.242.174-.475.36-.699.556-.197-.232-.4-.454-.61-.662-.77-.766-1.58-1.37-2.373-1.78-.232-.12-.458-.218-.676-.293zm5.612 1.082c-1.145-.212-2.47-.262-3.86-.15a13.2 13.2 0 0 1-.853 1.65 14.007 14.007 0 0 1 2.68.27c1.074.23 2.012.59 2.75 1.082.307.204.577.449.795.73.252.324.354.641.252.858-.3.635-1.826.87-3.795.529-.273-.047-.547-.104-.82-.169-.058.19-.12.38-.188.567.32.07.648.13.98.177 2.355.334 4.348-.01 4.94-.997.327-.546.226-1.252-.227-1.905-.5-.722-1.304-1.286-2.33-1.701a11.1 11.1 0 0 0-.324-.04z"/>
      </svg>
    ),
  },
  {
    name: 'Zod Validation',
    icon: <ShieldCheck className="h-6 w-6" />,
  },
  {
    name: 'Radix UI',
    icon: (
      <svg viewBox="0 0 25 25" className="h-6 w-6" fill="currentColor">
        <path d="M12 25C7.58173 25 4 21.4183 4 17C4 12.5817 7.58173 9 12 9V25Z"/>
        <path d="M12 0H4V8H12V0Z"/>
        <path d="M17 8C19.2091 8 21 6.20914 21 4C21 1.79086 19.2091 0 17 0C14.7909 0 13 1.79086 13 4C13 6.20914 14.7909 8 17 8Z"/>
      </svg>
    ),
  },
];
