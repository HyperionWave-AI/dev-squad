# Hyperion Coordinator UI - Documentation Hub

Welcome to the comprehensive documentation for **ui2** (Hyperion Coordinator UI), the modern web interface for the Hyperion Parallel Squad platform.

---

## 📚 What is Hyperion Coordinator UI?

**Hyperion Coordinator UI (ui2)** is a React-based web application for managing AI development workflows, coordinating parallel development teams, and interacting with AI agents in real-time.

### Key Features

- **🎯 Task Management** - Visual Kanban board for human and agent tasks with drag-and-drop
- **💬 AI Chat** - Real-time chat interface with WebSocket streaming
- **📚 Knowledge Base** - Browse and search curated knowledge collections
- **🔍 Code Search** - Semantic code search with intelligent indexing
- **🛠️ Tool Management** - Discover and test HTTP tools
- **⚙️ Configuration** - Manage MCP servers and sub-agents
- **📱 Responsive Design** - Mobile-first design from 375px to 1920px
- **♿ Accessibility** - WCAG 2.1 AA compliant
- **🌙 Dark Mode** - System preference detection with manual toggle

### Technology Stack

- **React 19.1.1** with TypeScript 5.8.3
- **Material-UI 7.3.2** for component library
- **Tailwind CSS 4.1.13** for utility styling
- **Vite 7.1.7** for build tooling
- **Playwright 1.55.1** for E2E testing
- **@hello-pangea/dnd** for drag-and-drop functionality

---

## 📖 Documentation Index

### Core Documentation

| Document | Description | Audience |
|----------|-------------|----------|
| **[ARCHITECTURE.md](./ARCHITECTURE.md)** | System architecture, data flow, component hierarchy | Developers, Architects |
| **[COMPONENTS.md](./COMPONENTS.md)** | Complete component catalog with usage examples | Developers, Designers |
| **[API_INTEGRATION.md](./API_INTEGRATION.md)** | Service layer, API clients, WebSocket integration | Developers |
| **[DEVELOPER_GUIDE.md](./DEVELOPER_GUIDE.md)** | Setup, workflow, coding standards, common tasks | New Developers |
| **[DEPLOYMENT.md](./DEPLOYMENT.md)** | Build process, deployment strategies, CI/CD | DevOps, SRE |

### Specialized Documentation

| Document | Description | Audience |
|----------|-------------|----------|
| **[TESTING.md](./TESTING.md)** | E2E tests, unit tests, accessibility testing | QA, Developers |
| **[UI_UX_PATTERNS.md](./UI_UX_PATTERNS.md)** | Design system, Material-UI theme, responsive patterns | Designers, Developers |
| **[TROUBLESHOOTING.md](./TROUBLESHOOTING.md)** | Common issues, debugging strategies, FAQ | All Users |

### Quick References

| Document | Description |
|----------|-------------|
| **[QUICKSTART.md](../QUICKSTART.md)** | Get started in 5 minutes |
| **[package.json](../package.json)** | Dependencies and npm scripts |
| **[playwright.config.ts](../playwright.config.ts)** | Test configuration |
| **[vite.config.ts](../vite.config.ts)** | Build configuration |

---

## 🚀 Quick Start

### Installation

```bash
# Navigate to UI directory
cd ui

# Install dependencies
npm install

# Start development server
npm run dev

# Open http://localhost:5173/ui/
```

### Essential Commands

```bash
# Development
npm run dev              # Start dev server with HMR

# Testing
npm test                 # Run all E2E tests
npm run test:headed      # Run tests with browser UI
npm run test:accessibility  # WCAG compliance tests

# Building
npm run build            # Production build
npm run preview          # Preview production build

# Code Quality
npm run lint             # Check linting
npm run lint -- --fix    # Auto-fix issues
npx tsc --noEmit        # Type check
```

---

## 🎯 Navigation by Role

### For New Developers

**Getting Started:**
1. Read [QUICKSTART.md](../QUICKSTART.md) for immediate setup
2. Study [DEVELOPER_GUIDE.md](./DEVELOPER_GUIDE.md) for detailed workflow
3. Review [ARCHITECTURE.md](./ARCHITECTURE.md) for system understanding
4. Explore [COMPONENTS.md](./COMPONENTS.md) for UI components

**First Tasks:**
- Set up development environment
- Run tests: `npm test`
- Create a simple component
- Submit a pull request

### For Frontend Developers

**Core Reading:**
1. [ARCHITECTURE.md](./ARCHITECTURE.md) - Understand layered architecture
2. [COMPONENTS.md](./COMPONENTS.md) - Reusable component library
3. [API_INTEGRATION.md](./API_INTEGRATION.md) - Service layer patterns
4. [UI_UX_PATTERNS.md](./UI_UX_PATTERNS.md) - Design system

**Focus Areas:**
- Component development patterns
- State management (local + server)
- Material-UI theming
- Responsive design principles

### For UI/UX Designers

**Design Resources:**
1. [UI_UX_PATTERNS.md](./UI_UX_PATTERNS.md) - Complete design system
2. [COMPONENTS.md](./COMPONENTS.md) - Component catalog with examples
3. [ARCHITECTURE.md](./ARCHITECTURE.md) - User flows and interactions

**Key Topics:**
- Color palette and typography
- Material-UI theme customization
- Accessibility standards (WCAG 2.1 AA)
- Responsive breakpoints (375px → 1920px)
- Dark mode implementation

### For QA Engineers

**Testing Documentation:**
1. [TESTING.md](./TESTING.md) - Comprehensive test guide
2. [TROUBLESHOOTING.md](./TROUBLESHOOTING.md) - Debugging strategies
3. [DEVELOPER_GUIDE.md](./DEVELOPER_GUIDE.md) - Running tests

**Test Coverage:**
- 8 E2E test suites (Playwright)
- Unit tests (Vitest)
- Accessibility tests (Axe-core)
- Visual regression tests
- Cross-browser testing (Chromium, WebKit)
- Responsive testing (mobile, tablet, desktop)

### For DevOps/SRE

**Deployment Documentation:**
1. [DEPLOYMENT.md](./DEPLOYMENT.md) - Build and deploy processes
2. [ARCHITECTURE.md](./ARCHITECTURE.md) - System architecture
3. [TROUBLESHOOTING.md](./TROUBLESHOOTING.md) - Production issues

**Key Topics:**
- Vite build configuration
- Docker containerization
- CI/CD pipelines (GitHub Actions)
- Environment variables
- Production optimization

---

## 📋 Navigation by Task

### Adding a New Feature

1. **Plan** - Review [ARCHITECTURE.md](./ARCHITECTURE.md) for patterns
2. **Design** - Check [UI_UX_PATTERNS.md](./UI_UX_PATTERNS.md) for design system
3. **Develop** - Follow [DEVELOPER_GUIDE.md](./DEVELOPER_GUIDE.md) standards
4. **Test** - Write tests per [TESTING.md](./TESTING.md)
5. **Deploy** - Use process in [DEPLOYMENT.md](./DEPLOYMENT.md)

**Checklist:**
- [ ] Component follows design system
- [ ] TypeScript types defined
- [ ] Accessibility compliance (WCAG 2.1 AA)
- [ ] E2E tests written
- [ ] Responsive across breakpoints
- [ ] API integration via REST client
- [ ] Dark mode support

### Fixing a Bug

1. **Reproduce** - Use [TROUBLESHOOTING.md](./TROUBLESHOOTING.md) to debug
2. **Locate** - Check [ARCHITECTURE.md](./ARCHITECTURE.md) for code location
3. **Fix** - Follow [DEVELOPER_GUIDE.md](./DEVELOPER_GUIDE.md) standards
4. **Test** - Add regression test per [TESTING.md](./TESTING.md)
5. **Verify** - Run full test suite

**Common Issues:**
- Architecture violations → [TROUBLESHOOTING.md](./TROUBLESHOOTING.md#architecture-violations)
- Build failures → [TROUBLESHOOTING.md](./TROUBLESHOOTING.md#build-failures)
- Test failures → [TROUBLESHOOTING.md](./TROUBLESHOOTING.md#test-failures)
- Performance issues → [TROUBLESHOOTING.md](./TROUBLESHOOTING.md#performance-issues)

### Improving Performance

1. **Measure** - Use React DevTools Profiler
2. **Analyze** - Review [TROUBLESHOOTING.md](./TROUBLESHOOTING.md#performance-issues)
3. **Optimize** - Apply best practices from [DEVELOPER_GUIDE.md](./DEVELOPER_GUIDE.md#best-practices)
4. **Test** - Verify with performance tests

**Strategies:**
- Memoization (useMemo, useCallback)
- Code splitting (lazy loading)
- Bundle optimization
- Image optimization

### Implementing Accessibility

1. **Learn** - Read [UI_UX_PATTERNS.md](./UI_UX_PATTERNS.md#accessibility-wcag-21-aa)
2. **Implement** - Follow WCAG 2.1 AA guidelines
3. **Test** - Run `npm run test:accessibility`
4. **Fix** - Use [TROUBLESHOOTING.md](./TROUBLESHOOTING.md#accessibility-violations)

**Requirements:**
- Color contrast 4.5:1 (text)
- Keyboard navigation support
- ARIA labels and roles
- Screen reader compatibility
- Focus management

---

## 🗂️ Navigation by Topic

### Architecture & Design

- [System Architecture](./ARCHITECTURE.md#architecture-layers)
- [Component Hierarchy](./ARCHITECTURE.md#component-hierarchy)
- [Data Flow](./ARCHITECTURE.md#data-flow)
- [Routing Architecture](./ARCHITECTURE.md#routing-architecture)
- [State Management](./ARCHITECTURE.md#state-management)

### Components & UI

- [Component Catalog](./COMPONENTS.md)
- [Material-UI Theme](./UI_UX_PATTERNS.md#material-ui-theme)
- [Tailwind CSS](./UI_UX_PATTERNS.md#tailwind-css-configuration)
- [Design System](./UI_UX_PATTERNS.md#design-system-overview)
- [Dark Mode](./UI_UX_PATTERNS.md#dark-mode)

### API & Integration

- [REST API Client](./API_INTEGRATION.md#rest-api-architecture)
- [WebSocket Integration](./API_INTEGRATION.md#websocket-integration)
- [Service Layer](./API_INTEGRATION.md#service-modules)
- [Error Handling](./API_INTEGRATION.md#error-handling)
- [Type Definitions](./API_INTEGRATION.md#type-definitions)

### Testing & Quality

- [E2E Testing](./TESTING.md#playwright-e2e-tests)
- [Unit Testing](./TESTING.md#unit-tests-vitest)
- [Accessibility Testing](./TESTING.md#accessibility-tests)
- [Visual Regression](./TESTING.md#visual-regression-tests)
- [Test Commands](./TESTING.md#test-commands)

### Development

- [Setup & Installation](./DEVELOPER_GUIDE.md#installation)
- [Coding Standards](./DEVELOPER_GUIDE.md#coding-standards)
- [Common Tasks](./DEVELOPER_GUIDE.md#common-tasks)
- [Debugging](./DEVELOPER_GUIDE.md#debugging)
- [Best Practices](./DEVELOPER_GUIDE.md#best-practices)

### Deployment

- [Build Process](./DEPLOYMENT.md#build-process)
- [CI/CD Pipeline](./DEPLOYMENT.md#cicd-pipeline)
- [Docker Deployment](./DEPLOYMENT.md#docker-deployment)
- [Environment Variables](./DEPLOYMENT.md#environment-variables)
- [Production Optimization](./DEPLOYMENT.md#production-optimization)

---

## 🛠️ Quick Links

### Setup & Installation
👉 [Developer Guide - Installation](./DEVELOPER_GUIDE.md#installation)

### Architecture Overview
👉 [Architecture - System Overview](./ARCHITECTURE.md#system-overview)

### Component Catalog
👉 [Components - Complete Catalog](./COMPONENTS.md)

### API Reference
👉 [API Integration - Service Modules](./API_INTEGRATION.md#service-modules)

### Testing Guide
👉 [Testing - Running Tests](./TESTING.md#test-commands)

### Design System
👉 [UI/UX Patterns - Design System](./UI_UX_PATTERNS.md#design-system-overview)

### Troubleshooting
👉 [Troubleshooting - Quick Checklist](./TROUBLESHOOTING.md#quick-diagnostic-checklist)

---

## 📝 Contributing to Documentation

### When to Update Documentation

Update documentation when:

- ✅ Adding new components or pages
- ✅ Changing API interfaces
- ✅ Modifying architecture patterns
- ✅ Updating dependencies
- ✅ Adding new features
- ✅ Fixing bugs that affect usage
- ✅ Changing build or deployment process

### How to Update Documentation

1. **Identify affected files:**
   - New component? → Update [COMPONENTS.md](./COMPONENTS.md)
   - API change? → Update [API_INTEGRATION.md](./API_INTEGRATION.md)
   - Architecture change? → Update [ARCHITECTURE.md](./ARCHITECTURE.md)

2. **Follow the existing format:**
   - Use consistent markdown structure
   - Include code examples with syntax highlighting
   - Add cross-references to related sections
   - Update table of contents if needed

3. **Verify accuracy:**
   - Test code examples
   - Verify cross-references work
   - Check for broken links
   - Ensure version numbers are current

4. **Update metadata:**
   ```markdown
   **Last Updated**: 2025-11-04
   **Version**: ui2 (Hyperion Coordinator UI)
   ```

### Documentation Review Process

1. **Self-review** - Check for accuracy and completeness
2. **Peer review** - Have another developer review changes
3. **Test examples** - Verify all code examples work
4. **Update index** - Update this README.md if needed

### Documentation Style Guide

**Headers:**
- Use `#` for main title (once per file)
- Use `##` for major sections
- Use `###` for subsections
- Use `####` for detailed topics

**Code Blocks:**
```typescript
// Always specify language
import { Component } from '@mui/material';
```

**Links:**
- Internal: `[Link Text](./FILE.md#section)`
- External: `[Link Text](https://example.com)`

**Lists:**
- Use `-` for unordered lists
- Use `1.` for ordered lists
- Use `- [ ]` for checklists

**Emphasis:**
- **Bold** for important terms
- *Italic* for emphasis
- `Code` for inline code

---

## 📊 Documentation Maintenance

### Version History

| Version | Date | Changes |
|---------|------|---------|
| 1.0.0 | 2025-11-04 | Initial complete documentation set |

### Review Schedule

- **Weekly**: Check for outdated code examples
- **Monthly**: Verify all links and cross-references
- **Quarterly**: Full documentation audit
- **On Release**: Update version numbers and changelogs

### Keeping Docs in Sync with Code

**Automated Checks:**
- TypeScript compilation ensures type examples are valid
- ESLint catches code style violations
- Tests verify API examples work

**Manual Checks:**
- Code review includes documentation review
- Feature PRs must include documentation updates
- Breaking changes require documentation updates

**Documentation Quality Metrics:**
- All public APIs documented
- All components have usage examples
- All configuration options explained
- All error messages documented

---

## 🔗 External Resources

### React & TypeScript

- [React Documentation](https://react.dev/)
- [TypeScript Handbook](https://www.typescriptlang.org/docs/)
- [React TypeScript Cheatsheet](https://react-typescript-cheatsheet.netlify.app/)

### Material-UI

- [Material-UI Documentation](https://mui.com/)
- [Material-UI Theme](https://mui.com/material-ui/customization/theming/)
- [Material-UI Components](https://mui.com/material-ui/getting-started/)

### Tailwind CSS

- [Tailwind CSS Documentation](https://tailwindcss.com/docs)
- [Tailwind CSS Configuration](https://tailwindcss.com/docs/configuration)

### Testing

- [Playwright Documentation](https://playwright.dev/)
- [Vitest Documentation](https://vitest.dev/)
- [Testing Library](https://testing-library.com/docs/react-testing-library/intro/)

### Accessibility

- [WCAG 2.1 Guidelines](https://www.w3.org/WAI/WCAG21/quickref/)
- [MDN Accessibility](https://developer.mozilla.org/en-US/docs/Web/Accessibility)
- [Axe-core](https://github.com/dequelabs/axe-core)

### Build Tools

- [Vite Documentation](https://vitejs.dev/)
- [ESLint Rules](https://eslint.org/docs/rules/)
- [Prettier Configuration](https://prettier.io/docs/en/configuration.html)

---

## 💡 Tips for Documentation Users

### For Quick Reference

Use browser search (Ctrl+F / Cmd+F) to find specific topics:
- Search "TypeScript" → Find type definitions
- Search "responsive" → Find responsive design patterns
- Search "accessibility" → Find WCAG guidelines
- Search "test" → Find testing strategies

### For Learning

Follow this learning path:

**Day 1: Setup**
- [QUICKSTART.md](../QUICKSTART.md)
- [DEVELOPER_GUIDE.md - Installation](./DEVELOPER_GUIDE.md#installation)

**Day 2-3: Architecture**
- [ARCHITECTURE.md](./ARCHITECTURE.md)
- [COMPONENTS.md](./COMPONENTS.md)

**Day 4-5: Development**
- [API_INTEGRATION.md](./API_INTEGRATION.md)
- [UI_UX_PATTERNS.md](./UI_UX_PATTERNS.md)

**Week 2: Testing & Deployment**
- [TESTING.md](./TESTING.md)
- [DEPLOYMENT.md](./DEPLOYMENT.md)

### For Problem Solving

1. Check [TROUBLESHOOTING.md](./TROUBLESHOOTING.md) first
2. Search relevant documentation file
3. Check code examples in docs
4. Review test files for usage patterns
5. Check browser console for errors

---

## 📞 Getting Help

### Documentation Issues

If you find:
- 📝 Outdated information → Create a pull request to update
- 🐛 Broken links → Report in issue tracker
- ❓ Missing documentation → Suggest addition in issue tracker
- 💡 Improvements → Submit pull request

### Technical Issues

For technical problems:

1. **Check documentation:**
   - [TROUBLESHOOTING.md](./TROUBLESHOOTING.md)
   - Relevant topic documentation

2. **Debug:**
   - Browser DevTools console
   - React Developer Tools
   - Network tab for API issues

3. **Review examples:**
   - Test files in `ui/tests/`
   - Component examples in docs
   - Source code in `ui/src/`

---

## 🎯 Documentation Goals

### Current Status ✅

- ✅ Complete architecture documentation
- ✅ Comprehensive component catalog
- ✅ API integration guide
- ✅ Developer onboarding guide
- ✅ Deployment documentation
- ✅ Testing guide
- ✅ UI/UX patterns documented
- ✅ Troubleshooting guide
- ✅ Documentation hub (this file)

### Future Enhancements 🚀

- 📹 Video tutorials for common tasks
- 🎨 Interactive component playground
- 📊 Performance benchmarking guide
- 🔐 Security best practices
- 🌐 Internationalization guide
- 📱 Progressive Web App features

---

## 📜 License

This documentation is part of the Hyperion Parallel Squad project.

---

**Last Updated**: 2025-11-04
**Version**: ui2 (Hyperion Coordinator UI)
**Maintainer**: Hyperion Platform Team

---

**Happy Coding! 🚀**
