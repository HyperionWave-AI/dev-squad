import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import './index.css'
import App from './App.tsx'

// Enhanced accessibility and color contrast setup
const setupAccessibility = () => {
  // Set up proper color scheme detection
  const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
  const prefersHighContrast = window.matchMedia('(prefers-contrast: high)').matches;
  const prefersReducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
  
  // Apply accessibility classes to document
  const documentElement = document.documentElement;
  
  if (prefersDark) {
    documentElement.classList.add('dark-mode');
  }
  
  if (prefersHighContrast) {
    documentElement.classList.add('high-contrast');
  }
  
  if (prefersReducedMotion) {
    documentElement.classList.add('reduced-motion');
  }
  
  // Set up color scheme meta tag for better browser integration
  const colorSchemeMetaTag = document.querySelector('meta[name="color-scheme"]');
  if (!colorSchemeMetaTag) {
    const meta = document.createElement('meta');
    meta.name = 'color-scheme';
    meta.content = prefersDark ? 'dark light' : 'light dark';
    document.head.appendChild(meta);
  }
  
  // Set up theme color meta tag for mobile browsers
  const themeColorMetaTag = document.querySelector('meta[name="theme-color"]');
  if (!themeColorMetaTag) {
    const meta = document.createElement('meta');
    meta.name = 'theme-color';
    meta.content = prefersDark ? '#111827' : '#ffffff';
    document.head.appendChild(meta);
  }
  
  // Listen for changes in user preferences
  window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', (e) => {
    if (e.matches) {
      documentElement.classList.add('dark-mode');
    } else {
      documentElement.classList.remove('dark-mode');
    }
  });
  
  window.matchMedia('(prefers-contrast: high)').addEventListener('change', (e) => {
    if (e.matches) {
      documentElement.classList.add('high-contrast');
    } else {
      documentElement.classList.remove('high-contrast');
    }
  });
  
  window.matchMedia('(prefers-reduced-motion: reduce)').addEventListener('change', (e) => {
    if (e.matches) {
      documentElement.classList.add('reduced-motion');
    } else {
      documentElement.classList.remove('reduced-motion');
    }
  });
};

// Set up viewport meta tag for proper mobile rendering
const setupViewport = () => {
  const viewportMetaTag = document.querySelector('meta[name="viewport"]');
  if (!viewportMetaTag) {
    const meta = document.createElement('meta');
    meta.name = 'viewport';
    meta.content = 'width=device-width, initial-scale=1.0, viewport-fit=cover, user-scalable=yes';
    document.head.appendChild(meta);
  }
};

// Set up proper font loading for better typography
const setupFontLoading = () => {
  // Preload system fonts for better performance
  const fontPreloadLink = document.createElement('link');
  fontPreloadLink.rel = 'preconnect';
  fontPreloadLink.href = 'https://fonts.googleapis.com';
  document.head.appendChild(fontPreloadLink);
  
  // Set up font display swap for better loading performance
  const style = document.createElement('style');
  style.textContent = `
    @font-face {
      font-family: system-ui;
      font-display: swap;
    }
  `;
  document.head.appendChild(style);
};

// Initialize accessibility and performance features
setupAccessibility();
setupViewport();
setupFontLoading();

// Enhanced root rendering with better error handling
const rootElement = document.getElementById('root');

if (!rootElement) {
  throw new Error('Root element not found. Please ensure there is a div with id="root" in your HTML.');
}

// Add ARIA label to root for screen readers
rootElement.setAttribute('aria-label', 'Hyperion Application');
rootElement.setAttribute('role', 'application');

// Create and render the application
const root = createRoot(rootElement);

root.render(
  <StrictMode>
    <BrowserRouter basename="/ui">
      <App />
    </BrowserRouter>
  </StrictMode>
);

// Add global keyboard navigation support
document.addEventListener('keydown', (event) => {
  // Skip to main content with Alt+M
  if (event.altKey && event.key === 'm') {
    event.preventDefault();
    const mainContent = document.querySelector('main, [role="main"], .main-content');
    if (mainContent instanceof HTMLElement) {
      mainContent.focus();
      mainContent.scrollIntoView({ behavior: 'smooth' });
    }
  }
  
  // Toggle high contrast mode with Alt+H
  if (event.altKey && event.key === 'h') {
    event.preventDefault();
    document.documentElement.classList.toggle('high-contrast');
  }
  
  // Focus management for modal dialogs
  if (event.key === 'Escape') {
    const activeModal = document.querySelector('[role="dialog"][aria-modal="true"]');
    if (activeModal) {
      const closeButton = activeModal.querySelector('[aria-label*="close"], [aria-label*="Close"]');
      if (closeButton instanceof HTMLElement) {
        closeButton.click();
      }
    }
  }
});

// Add focus management for better keyboard navigation
document.addEventListener('focusin', (event) => {
  const target = event.target as HTMLElement;
  
  // Ensure focused elements are visible
  if (target && target.scrollIntoView) {
    target.scrollIntoView({
      behavior: 'smooth',
      block: 'nearest',
      inline: 'nearest'
    });
  }
});

// Performance monitoring for Core Web Vitals
if (typeof window !== 'undefined' && 'PerformanceObserver' in window) {
  // Monitor Largest Contentful Paint (LCP)
  const observer = new PerformanceObserver((list) => {
    for (const entry of list.getEntries()) {
      if (entry.entryType === 'largest-contentful-paint') {
        console.log('LCP:', entry.startTime);
      }
      if (entry.entryType === 'first-input') {
        console.log('FID:', entry.startTime);
      }
    }
  });
  
  try {
    observer.observe({ entryTypes: ['largest-contentful-paint', 'first-input'] });
  } catch (e) {
    // Fallback for browsers that don't support these metrics
    console.log('Performance monitoring not available in this browser');
  }
}

// Service worker registration for better caching (if available)
if ('serviceWorker' in navigator && process.env.NODE_ENV === 'production') {
  window.addEventListener('load', () => {
    navigator.serviceWorker.register('/sw.js')
      .then((registration) => {
        console.log('SW registered: ', registration);
      })
      .catch((registrationError) => {
        console.log('SW registration failed: ', registrationError);
      });
  });
}