/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      // Enhanced mobile-first breakpoints aligned with Material-UI
      screens: {
        'xs': '0px',
        'sm': '600px',   // Material-UI sm breakpoint
        'md': '900px',   // Material-UI md breakpoint  
        'lg': '1200px',  // Material-UI lg breakpoint
        'xl': '1536px',  // Material-UI xl breakpoint
        // Additional mobile breakpoints for better control
        'mobile-sm': '375px',  // Small mobile devices
        'mobile-lg': '414px',  // Large mobile devices
        'tablet-sm': '768px',  // Small tablets
        'tablet-lg': '1024px', // Large tablets
      },
      // Enhanced responsive spacing that matches our CSS custom properties
      spacing: {
        'xs': '0.25rem',    // 4px
        'sm': '0.5rem',     // 8px
        'md': '1rem',       // 16px
        'lg': '1.5rem',     // 24px
        'xl': '2rem',       // 32px
        '2xl': '3rem',      // 48px
        '3xl': '4rem',      // 64px
        // Mobile-specific spacing
        'mobile-xs': '0.125rem', // 2px
        'mobile-sm': '0.375rem', // 6px
        'mobile-md': '0.75rem',  // 12px
        'mobile-lg': '1.25rem',  // 20px
        // Touch-friendly spacing
        'touch-sm': '2.75rem',   // 44px - minimum touch target
        'touch-md': '3rem',      // 48px - comfortable touch target
        'touch-lg': '3.5rem',    // 56px - large touch target
      },
      // Mobile-optimized typography scale
      fontSize: {
        'xs': ['0.75rem', { lineHeight: '1rem' }],
        'sm': ['0.875rem', { lineHeight: '1.25rem' }],
        'base': ['1rem', { lineHeight: '1.5rem' }],
        'lg': ['1.125rem', { lineHeight: '1.75rem' }],
        'xl': ['1.25rem', { lineHeight: '1.75rem' }],
        '2xl': ['1.5rem', { lineHeight: '2rem' }],
        '3xl': ['1.875rem', { lineHeight: '2.25rem' }],
        '4xl': ['2.25rem', { lineHeight: '2.5rem' }],
        // Mobile-specific font sizes
        'mobile-xs': ['0.6875rem', { lineHeight: '0.875rem' }], // 11px
        'mobile-sm': ['0.8125rem', { lineHeight: '1.125rem' }], // 13px
        'mobile-base': ['0.9375rem', { lineHeight: '1.375rem' }], // 15px
        'mobile-lg': ['1.0625rem', { lineHeight: '1.5rem' }], // 17px
      },
      // Touch-friendly sizing with better mobile support
      minHeight: {
        'touch': '44px',           // iOS minimum
        'touch-comfortable': '48px', // Comfortable touch
        'touch-large': '56px',     // Material Design large
        'mobile-header': '56px',   // Mobile header height
        'tablet-header': '64px',   // Tablet header height
        'mobile-nav': '60px',      // Mobile navigation height
      },
      minWidth: {
        'touch': '44px',
        'touch-comfortable': '48px', 
        'touch-large': '56px',
        'mobile-button': '88px',   // Minimum button width on mobile
      },
      maxWidth: {
        'mobile': '100vw',
        'mobile-content': 'calc(100vw - 2rem)', // Account for padding
        'tablet': '768px',
        'desktop': '1200px',
        'wide': '1600px',
      },
      // Enhanced colors with better mobile contrast
      colors: {
        primary: {
          50: '#eff6ff',
          100: '#dbeafe', 
          200: '#bfdbfe',
          300: '#93c5fd',
          400: '#60a5fa',
          500: '#3b82f6', // Main primary color
          600: '#2563eb',
          700: '#1d4ed8',
          800: '#1e40af',
          900: '#1e3a8a',
          950: '#172554', // Extra dark for high contrast
        },
        secondary: {
          50: '#faf5ff',
          100: '#f3e8ff',
          200: '#e9d5ff', 
          300: '#d8b4fe',
          400: '#c084fc',
          500: '#a855f7',
          600: '#9333ea', // Secondary color
          700: '#7c3aed',
          800: '#6b21a8',
          900: '#581c87',
          950: '#3b0764', // Extra dark for high contrast
        },
        // Mobile-optimized grays with better contrast
        gray: {
          50: '#f9fafb',
          100: '#f3f4f6',
          200: '#e5e7eb',
          300: '#d1d5db',
          400: '#9ca3af',
          500: '#6b7280',
          600: '#4b5563',
          700: '#374151',
          800: '#1f2937',
          900: '#111827',
          950: '#030712',
        }
      },
      // Mobile-optimized animations
      animation: {
        'fade-in': 'fadeIn 0.2s ease-in-out',
        'fade-in-slow': 'fadeIn 0.4s ease-in-out',
        'slide-in': 'slideIn 0.3s ease-out',
        'slide-in-mobile': 'slideInMobile 0.25s ease-out',
        'bounce-subtle': 'bounceSubtle 0.6s ease-in-out',
        'pulse-subtle': 'pulseSubtle 2s ease-in-out infinite',
        'shake': 'shake 0.5s ease-in-out',
      },
      keyframes: {
        fadeIn: {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' },
        },
        slideIn: {
          '0%': { transform: 'translateX(-100%)' },
          '100%': { transform: 'translateX(0)' },
        },
        slideInMobile: {
          '0%': { transform: 'translateX(-100%)', opacity: '0' },
          '100%': { transform: 'translateX(0)', opacity: '1' },
        },
        bounceSubtle: {
          '0%, 100%': { transform: 'translateY(0)' },
          '50%': { transform: 'translateY(-2px)' },
        },
        pulseSubtle: {
          '0%, 100%': { opacity: '1' },
          '50%': { opacity: '0.8' },
        },
        shake: {
          '0%, 100%': { transform: 'translateX(0)' },
          '10%, 30%, 50%, 70%, 90%': { transform: 'translateX(-2px)' },
          '20%, 40%, 60%, 80%': { transform: 'translateX(2px)' },
        }
      },
      // Enhanced box shadows for mobile depth perception
      boxShadow: {
        'soft': '0 1px 2px rgba(0, 0, 0, 0.05)',
        'medium': '0 4px 6px rgba(0, 0, 0, 0.07)',
        'strong': '0 10px 15px rgba(0, 0, 0, 0.1)',
        'mobile': '0 2px 4px rgba(0, 0, 0, 0.1)',
        'mobile-strong': '0 4px 8px rgba(0, 0, 0, 0.15)',
        'focus': '0 0 0 3px rgba(59, 130, 246, 0.1)',
        'focus-strong': '0 0 0 3px rgba(59, 130, 246, 0.2)',
      },
      // Mobile-optimized border radius
      borderRadius: {
        'mobile': '6px',
        'mobile-lg': '8px',
        'mobile-xl': '12px',
        'touch': '8px', // Good for touch targets
      },
      // Z-index scale for proper layering on mobile
      zIndex: {
        'mobile-nav': '40',
        'mobile-drawer': '50',
        'mobile-modal': '60',
        'mobile-toast': '70',
        'mobile-tooltip': '80',
      },
      // Mobile-specific container sizes
      container: {
        center: true,
        padding: {
          DEFAULT: '1rem',
          'sm': '1.5rem',
          'lg': '2rem',
          'xl': '3rem',
        },
        screens: {
          'sm': '600px',
          'md': '900px',
          'lg': '1200px',
          'xl': '1536px',
        },
      },
      // Aspect ratios for responsive media
      aspectRatio: {
        'mobile-card': '16 / 10',
        'mobile-hero': '16 / 9',
        'mobile-square': '1 / 1',
      },
      // Mobile-optimized line heights
      lineHeight: {
        'mobile-tight': '1.1',
        'mobile-normal': '1.4',
        'mobile-relaxed': '1.6',
      },
      // Letter spacing for mobile readability
      letterSpacing: {
        'mobile-tight': '-0.01em',
        'mobile-normal': '0',
        'mobile-wide': '0.02em',
      },
      // Mobile-specific transitions
      transitionDuration: {
        'mobile': '200ms',
        'mobile-slow': '300ms',
      },
      transitionTimingFunction: {
        'mobile': 'cubic-bezier(0.4, 0, 0.2, 1)',
        'mobile-bounce': 'cubic-bezier(0.68, -0.55, 0.265, 1.55)',
      },
    },
  },
  plugins: [
    // Custom plugin for mobile-specific utilities
    function({ addUtilities, theme }) {
      const newUtilities = {
        // Mobile-safe text sizing (prevents zoom on iOS)
        '.text-mobile-safe': {
          fontSize: '16px',
        },
        // Touch-friendly interactive elements
        '.touch-target': {
          minHeight: theme('minHeight.touch'),
          minWidth: theme('minWidth.touch'),
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
        },
        '.touch-target-comfortable': {
          minHeight: theme('minHeight.touch-comfortable'),
          minWidth: theme('minWidth.touch-comfortable'),
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
        },
        // Mobile-optimized scrolling
        '.scroll-mobile': {
          '-webkit-overflow-scrolling': 'touch',
          overscrollBehavior: 'contain',
        },
        // Mobile-safe viewport units
        '.h-mobile-screen': {
          height: '100vh',
          height: '100dvh', // Dynamic viewport height for mobile browsers
        },
        '.min-h-mobile-screen': {
          minHeight: '100vh',
          minHeight: '100dvh',
        },
        // Mobile-optimized focus states
        '.focus-mobile': {
          '&:focus': {
            outline: '2px solid',
            outlineColor: theme('colors.primary.500'),
            outlineOffset: '2px',
          },
        },
        // Mobile-specific hiding/showing
        '.mobile-only': {
          '@media (min-width: 600px)': {
            display: 'none',
          },
        },
        '.desktop-only': {
          '@media (max-width: 599px)': {
            display: 'none',
          },
        },
        // Safe area padding for mobile devices with notches
        '.safe-area-padding': {
          paddingTop: 'env(safe-area-inset-top)',
          paddingRight: 'env(safe-area-inset-right)',
          paddingBottom: 'env(safe-area-inset-bottom)',
          paddingLeft: 'env(safe-area-inset-left)',
        },
        '.safe-area-margin': {
          marginTop: 'env(safe-area-inset-top)',
          marginRight: 'env(safe-area-inset-right)',
          marginBottom: 'env(safe-area-inset-bottom)',
          marginLeft: 'env(safe-area-inset-left)',
        },
      }
      addUtilities(newUtilities)
    }
  ],
  // Prevent conflicts with Material-UI but enable more features for mobile
  corePlugins: {
    preflight: false, // Disable Tailwind's base styles to avoid conflicts with Material-UI
  },
}