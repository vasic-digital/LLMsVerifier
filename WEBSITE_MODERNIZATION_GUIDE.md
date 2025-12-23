# Website Modernization Guide

**Generated**: $(date -u +"%Y-%m-%d %H:%M:%S UTC")
**Project**: LLM Verifier Website
**Location**: `/media/milosvasic/DATA4TB/Projects/LLM/LLMsVerifier/Website`

---

## OVERVIEW

This guide provides step-by-step instructions to modernize the LLM Verifier website with:
- Modern responsive design
- Full content pages
- Integrated documentation
- SEO optimization
- Analytics integration
- Deployment pipeline

---

## WEEK 17: WEBSITE FOUNDATION

### Day 1: Hugo Static Site Generator Setup

#### Step 1: Install Hugo

```bash
# Download Hugo extended version
cd /media/milosvasic/DATA4TB/Projects/LLM/LLMsVerifier
wget https://github.com/gohugoio/hugo/releases/download/v0.121.2/hugo_extended_0.121.2_linux-amd64.deb
sudo dpkg -i hugo_extended_0.121.2_linux-amd64.deb

# Verify installation
hugo version
```

#### Step 2: Initialize Hugo Site

```bash
cd Website/
hugo new site . --force

# Directory structure created
Website/
├── archetypes/
├── assets/
├── config/
├── content/
├── data/
├── layouts/
├── static/
├── themes/
└── hugo.toml
```

#### Step 3: Install Modern Theme

```bash
cd Website/themes/
git clone https://github.com/thegeeklab/hugo-geekdoc.git

# Update config to use theme
cd ..
cat > config.toml << 'EOF'
baseURL = "https://llm-verifier.com"
title = "LLM Verifier"
theme = "hugo-geekdoc"

[params]
  geekdocLogo = "logo.svg"
  geekdocRepo = "https://github.com/vasic-digital/LLMsVerifier"
  geekdocEditPath = "edit/master"
EOF
```

---

### Day 2: Website Templates

#### Create Layouts Directory Structure

```bash
cd Website/
mkdir -p layouts/_default
mkdir -p layouts/partials
mkdir -p layouts/shortcodes
```

#### Step 1: Create Base Template

**File**: `Website/layouts/_default/baseof.html`

```html
<!DOCTYPE html>
<html lang="{{ .Site.Language.Lang }}" class="scroll-smooth">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta name="description" content="{{ if .Params.description }}{{ .Params.description }}{{ else }}{{ .Site.Params.description }}{{ end }}">
    <title>{{ .Title }} | LLM Verifier</title>
    
    <!-- Tailwind CSS -->
    <script src="https://cdn.tailwindcss.com"></script>
    
    <!-- Custom CSS -->
    <link rel="stylesheet" href="{{ .Site.BaseURL }}css/styles.css">
    
    <!-- Favicon -->
    <link rel="icon" type="image/svg+xml" href="{{ .Site.BaseURL }}favicon.svg">
    
    <!-- Open Graph -->
    <meta property="og:title" content="{{ .Title }}">
    <meta property="og:description" content="{{ if .Params.description }}{{ .Params.description }}{{ else }}{{ .Site.Params.description }}{{ end }}">
    <meta property="og:type" content="website">
    <meta property="og:url" content="{{ .Permalink }}">
    <meta property="og:image" content="{{ .Site.BaseURL }}images/og-default.png">
    
    <!-- Twitter Card -->
    <meta name="twitter:card" content="summary_large_image">
    <meta name="twitter:title" content="{{ .Title }}">
    <meta name="twitter:description" content="{{ if .Params.description }}{{ .Params.description }}{{ else }}{{ .Site.Params.description }}{{ end }}">
    
    <!-- Structured Data -->
    {{ template "_internal/schema.html" . }}
</head>
<body class="bg-gray-50 text-gray-900 font-sans antialiased">
    {{ partial "header.html" . }}
    
    <main class="flex-grow">
        {{ block "main" . }}{{ end }}
    </main>
    
    {{ partial "footer.html" . }}
    
    {{ partial "scripts.html" . }}
</body>
</html>
```

#### Step 2: Create Header Template

**File**: `Website/layouts/partials/header.html`

```html
<header class="fixed top-0 z-50 w-full bg-white shadow-sm">
    <nav class="container mx-auto px-4 sm:px-6 lg:px-8">
        <div class="flex items-center justify-between h-16">
            <!-- Logo -->
            <div class="flex-shrink-0">
                <a href="{{ .Site.BaseURL }}" class="flex items-center space-x-2">
                    <svg class="h-8 w-8 text-blue-600" viewBox="0 0 24 24" fill="currentColor">
                        <path d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5"/>
                    </svg>
                    <span class="text-xl font-bold text-gray-900">LLM Verifier</span>
                </a>
            </div>
            
            <!-- Desktop Navigation -->
            <div class="hidden md:flex md:space-x-8">
                {{ range .Site.Menus.main }}
                <a href="{{ .URL }}" class="text-gray-600 hover:text-blue-600 font-medium transition-colors">
                    {{ .Name }}
                </a>
                {{ end }}
            </div>
            
            <!-- Mobile Menu Button -->
            <div class="md:hidden">
                <button id="mobile-menu-btn" class="text-gray-600 hover:text-gray-900 focus:outline-none">
                    <svg class="h-6 w-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16"/>
                    </svg>
                </button>
            </div>
        </div>
        
        <!-- Mobile Menu -->
        <div id="mobile-menu" class="hidden md:hidden pb-4">
            {{ range .Site.Menus.main }}
            <a href="{{ .URL }}" class="block py-2 text-gray-600 hover:text-blue-600 font-medium">
                {{ .Name }}
            </a>
            {{ end }}
        </div>
    </nav>
</header>

<script>
    document.getElementById('mobile-menu-btn').addEventListener('click', () => {
        document.getElementById('mobile-menu').classList.toggle('hidden');
    });
</script>
```

#### Step 3: Create Footer Template

**File**: `Website/layouts/partials/footer.html`

```html
<footer class="bg-gray-900 text-gray-300 mt-auto">
    <div class="container mx-auto px-4 py-12 sm:px-6 lg:px-8">
        <div class="grid grid-cols-1 md:grid-cols-4 gap-8">
            <!-- Brand -->
            <div>
                <div class="flex items-center space-x-2 mb-4">
                    <svg class="h-6 w-6 text-blue-400" viewBox="0 0 24 24" fill="currentColor">
                        <path d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5"/>
                    </svg>
                    <span class="text-lg font-bold text-white">LLM Verifier</span>
                </div>
                <p class="text-sm text-gray-400">
                    The most comprehensive enterprise-grade platform for verifying, monitoring, and optimizing Large Language Model performance.
                </p>
            </div>
            
            <!-- Product -->
            <div>
                <h3 class="text-sm font-semibold text-white uppercase tracking-wider mb-4">Product</h3>
                <ul class="space-y-3 text-sm">
                    <li><a href="/features" class="hover:text-blue-400 transition-colors">Features</a></li>
                    <li><a href="/pricing" class="hover:text-blue-400 transition-colors">Pricing</a></li>
                    <li><a href="/download" class="hover:text-blue-400 transition-colors">Downloads</a></li>
                    <li><a href="/changelog" class="hover:text-blue-400 transition-colors">Changelog</a></li>
                </ul>
            </div>
            
            <!-- Resources -->
            <div>
                <h3 class="text-sm font-semibold text-white uppercase tracking-wider mb-4">Resources</h3>
                <ul class="space-y-3 text-sm">
                    <li><a href="/docs" class="hover:text-blue-400 transition-colors">Documentation</a></li>
                    <li><a href="/api" class="hover:text-blue-400 transition-colors">API Reference</a></li>
                    <li><a href="/guides" class="hover:text-blue-400 transition-colors">User Guides</a></li>
                    <li><a href="/videos" class="hover:text-blue-400 transition-colors">Video Tutorials</a></li>
                </ul>
            </div>
            
            <!-- Community -->
            <div>
                <h3 class="text-sm font-semibold text-white uppercase tracking-wider mb-4">Community</h3>
                <ul class="space-y-3 text-sm">
                    <li><a href="https://github.com/vasic-digital/LLMsVerifier" class="hover:text-blue-400 transition-colors">GitHub</a></li>
                    <li><a href="/discussions" class="hover:text-blue-400 transition-colors">Discussions</a></li>
                    <li><a href="/support" class="hover:text-blue-400 transition-colors">Support</a></li>
                    <li><a href="/contact" class="hover:text-blue-400 transition-colors">Contact</a></li>
                </ul>
            </div>
        </div>
        
        <div class="mt-8 pt-8 border-t border-gray-800">
            <div class="flex flex-col md:flex-row justify-between items-center">
                <p class="text-sm text-gray-400">
                    &copy; 2024 LLM Verifier. All rights reserved.
                </p>
                <div class="flex space-x-6 mt-4 md:mt-0">
                    <a href="/privacy" class="text-sm text-gray-400 hover:text-white transition-colors">Privacy</a>
                    <a href="/terms" class="text-sm text-gray-400 hover:text-white transition-colors">Terms</a>
                    <a href="/security" class="text-sm text-gray-400 hover:text-white transition-colors">Security</a>
                </div>
            </div>
        </div>
    </div>
</footer>
```

---

### Day 3: CSS Framework

**File**: `Website/static/css/styles.css`

```css
/* Custom Variables */
:root {
    --primary-color: #2563eb;
    --secondary-color: #1d4ed8;
    --accent-color: #3b82f6;
    --text-primary: #111827;
    --text-secondary: #6b7280;
    --bg-primary: #ffffff;
    --bg-secondary: #f9fafb;
    --border-color: #e5e7eb;
}

/* Base Styles */
html {
    scroll-behavior: smooth;
}

body {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
    -webkit-font-smoothing: antialiased;
    -moz-osx-font-smoothing: grayscale;
}

/* Typography */
h1, h2, h3, h4, h5, h6 {
    font-weight: 600;
    line-height: 1.25;
    margin-bottom: 1rem;
}

h1 { font-size: 2.5rem; }
h2 { font-size: 2rem; }
h3 { font-size: 1.5rem; }

p {
    line-height: 1.75;
    margin-bottom: 1rem;
}

a {
    color: var(--primary-color);
    text-decoration: none;
    transition: color 0.2s;
}

a:hover {
    color: var(--secondary-color);
}

/* Buttons */
.btn-primary {
    background-color: var(--primary-color);
    color: white;
    padding: 0.75rem 1.5rem;
    border-radius: 0.375rem;
    font-weight: 500;
    transition: background-color 0.2s;
    display: inline-block;
}

.btn-primary:hover {
    background-color: var(--secondary-color);
}

.btn-secondary {
    background-color: transparent;
    color: var(--text-primary);
    border: 1px solid var(--border-color);
    padding: 0.75rem 1.5rem;
    border-radius: 0.375rem;
    font-weight: 500;
    transition: all 0.2s;
}

.btn-secondary:hover {
    background-color: var(--bg-secondary);
    border-color: var(--primary-color);
}

/* Cards */
.card {
    background-color: var(--bg-primary);
    border: 1px solid var(--border-color);
    border-radius: 0.5rem;
    padding: 1.5rem;
    transition: all 0.2s;
}

.card:hover {
    box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1);
    transform: translateY(-2px);
}

/* Code Blocks */
pre {
    background-color: #1e293b;
    color: #e2e8f0;
    padding: 1rem;
    border-radius: 0.5rem;
    overflow-x: auto;
    margin: 1rem 0;
}

code {
    font-family: 'Fira Code', 'Monaco', monospace;
    font-size: 0.875rem;
}

/* Tables */
table {
    width: 100%;
    border-collapse: collapse;
    margin: 1.5rem 0;
}

th, td {
    padding: 0.75rem;
    text-align: left;
    border-bottom: 1px solid var(--border-color);
}

th {
    background-color: var(--bg-secondary);
    font-weight: 600;
}

/* Responsive */
@media (max-width: 768px) {
    h1 { font-size: 2rem; }
    h2 { font-size: 1.75rem; }
    h3 { font-size: 1.25rem; }
}

/* Animations */
@keyframes fadeIn {
    from { opacity: 0; }
    to { opacity: 1; }
}

.fade-in {
    animation: fadeIn 0.3s ease-in;
}
```

---

### Day 4: JavaScript Bundles

**File**: `Website/static/js/main.js`

```javascript
// Navigation
function initNavigation() {
    const mobileMenuBtn = document.getElementById('mobile-menu-btn');
    const mobileMenu = document.getElementById('mobile-menu');
    
    mobileMenuBtn?.addEventListener('click', () => {
        mobileMenu?.classList.toggle('hidden');
    });
}

// Scroll to top button
function initScrollToTop() {
    const scrollToTopBtn = document.getElementById('scroll-to-top');
    
    window.addEventListener('scroll', () => {
        if (window.scrollY > 300) {
            scrollToTopBtn?.classList.remove('hidden');
        } else {
            scrollToTopBtn?.classList.add('hidden');
        }
    });
    
    scrollToTopBtn?.addEventListener('click', () => {
        window.scrollTo({ top: 0, behavior: 'smooth' });
    });
}

// Dark mode toggle
function initDarkMode() {
    const darkModeToggle = document.getElementById('dark-mode-toggle');
    
    // Check localStorage for preference
    const prefersDark = localStorage.getItem('darkMode') === 'true';
    if (prefersDark) {
        document.documentElement.classList.add('dark');
    }
    
    darkModeToggle?.addEventListener('click', () => {
        document.documentElement.classList.toggle('dark');
        localStorage.setItem('darkMode', document.documentElement.classList.contains('dark'));
    });
}

// Search functionality
function initSearch() {
    const searchInput = document.getElementById('search-input');
    const searchResults = document.getElementById('search-results');
    
    let searchTimeout;
    
    searchInput?.addEventListener('input', (e) => {
        clearTimeout(searchTimeout);
        searchTimeout = setTimeout(() => {
            performSearch(e.target.value);
        }, 300);
    });
}

async function performSearch(query) {
    if (!query || query.length < 2) {
        document.getElementById('search-results').innerHTML = '';
        return;
    }
    
    try {
        const response = await fetch(`/api/search?q=${encodeURIComponent(query)}`);
        const results = await response.json();
        displaySearchResults(results);
    } catch (error) {
        console.error('Search error:', error);
    }
}

function displaySearchResults(results) {
    const resultsContainer = document.getElementById('search-results');
    
    if (results.length === 0) {
        resultsContainer.innerHTML = `
            <div class="p-4 text-gray-500">
                No results found
            </div>
        `;
        return;
    }
    
    resultsContainer.innerHTML = results.map(result => `
        <a href="${result.url}" class="block p-3 hover:bg-gray-100 transition-colors">
            <h3 class="font-semibold text-gray-900">${result.title}</h3>
            <p class="text-sm text-gray-600">${result.excerpt}</p>
        </a>
    `).join('');
}

// Analytics
function initAnalytics() {
    // Google Analytics (replace with your tracking ID)
    const gtagScript = document.createElement('script');
    gtagScript.src = 'https://www.googletagmanager.com/gtag/js?id=G-XXXXXXXXXX';
    document.head.appendChild(gtagScript);
    
    window.dataLayer = window.dataLayer || [];
    function gtag(){dataLayer.push(arguments);}
    gtag('js', new Date());
    gtag('config', 'G-XXXXXXXXXX');
}

// Initialize all
document.addEventListener('DOMContentLoaded', () => {
    initNavigation();
    initScrollToTop();
    initDarkMode();
    initSearch();
    initAnalytics();
});
```

---

### Day 5-7: Asset Pipeline

#### Step 1: Create Asset Optimization Script

**File**: `Website/scripts/optimize-assets.sh`

```bash
#!/bin/bash

# Optimize images
echo "Optimizing images..."
find static/images -name "*.png" -exec pngquant --quality=85 --ext=.png --force {} \;
find static/images -name "*.jpg" -exec jpegoptim --max=85 --strip-all {} \;

# Minify CSS
echo "Minifying CSS..."
for css in static/css/*.css; do
    cssnano "$css" "$css.min" --output style.css
    mv "$css.min" "$css"
done

# Minify JS
echo "Minifying JavaScript..."
for js in static/js/*.js; do
    terser "$js" -c -m -o "$js.min"
    mv "$js.min" "$js"
done

# Create asset manifest
echo "Creating asset manifest..."
cat > static/asset-manifest.json << 'EOF'
{
    "main.css": "/css/styles.css",
    "main.js": "/js/main.js",
    "favicon.svg": "/favicon.svg",
    "logo.svg": "/images/logo.svg"
}
EOF

echo "Assets optimized successfully!"
```

#### Step 2: Create Watch Script

**File**: `Website/scripts/watch.sh`

```bash
#!/bin/bash

watchexec -e css,html,md,js -- bash -c '
    hugo server --disableFastRender
'
```

---

## WEEK 18: WEBSITE CONTENT

### Day 1: Home Page

**File**: `Website/content/_index.md`

```markdown
---
title: "Home"
description: "The most comprehensive enterprise-grade platform for verifying, monitoring, and optimizing Large Language Model performance across multiple providers."
type: "home"
---

<!-- Hero Section -->
<section class="bg-gradient-to-br from-blue-600 to-purple-700 text-white py-20">
    <div class="container mx-auto px-4 text-center">
        <h1 class="text-4xl md:text-6xl font-bold mb-6">
            Verify, Monitor & Optimize<br>Your LLMs with Confidence
        </h1>
        <p class="text-xl md:text-2xl mb-8 text-blue-100">
            20+ comprehensive tests, real-time monitoring, and AI-powered insights
            for enterprise-grade LLM reliability
        </p>
        <div class="flex flex-col sm:flex-row gap-4 justify-center">
            <a href="/download" class="btn-primary text-lg px-8 py-4">
                Download for Free
            </a>
            <a href="/docs" class="btn-secondary text-lg px-8 py-4 bg-white text-blue-600 hover:bg-blue-50">
                View Documentation
            </a>
        </div>
    </div>
</section>

<!-- Features Section -->
<section class="py-20 bg-white">
    <div class="container mx-auto px-4">
        <h2 class="text-3xl md:text-4xl font-bold text-center mb-4">
            Why Choose LLM Verifier?
        </h2>
        <p class="text-center text-gray-600 max-w-2xl mx-auto mb-12">
            The most comprehensive LLM verification platform trusted by enterprises worldwide
        </p>
        
        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-8">
            <!-- Feature 1 -->
            <div class="card">
                <div class="text-blue-600 mb-4">
                    <svg class="h-12 w-12" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"/>
                    </svg>
                </div>
                <h3 class="text-xl font-bold mb-2">20+ Verification Tests</h3>
                <p class="text-gray-600">
                    Comprehensive capability assessment including MCPs, LSPs, reranking,
                    embeddings, tool use, and more.
                </p>
            </div>
            
            <!-- Feature 2 -->
            <div class="card">
                <div class="text-purple-600 mb-4">
                    <svg class="h-12 w-12" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"/>
                    </svg>
                </div>
                <h3 class="text-xl font-bold mb-2">Multi-Provider Support</h3>
                <p class="text-gray-600">
                    OpenAI, Anthropic, Google, Cohere, Meta, and 10+ more providers
                    with automatic failover and load balancing.
                </p>
            </div>
            
            <!-- Feature 3 -->
            <div class="card">
                <div class="text-green-600 mb-4">
                    <svg class="h-12 w-12" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"/>
                    </svg>
                </div>
                <h3 class="text-xl font-bold mb-2">Real-Time Monitoring</h3>
                <p class="text-gray-600">
                    99.9% uptime with intelligent failover, real-time dashboards,
                    and AI-powered performance insights.
                </p>
            </div>
            
            <!-- Feature 4 -->
            <div class="card">
                <div class="text-red-600 mb-4">
                    <svg class="h-12 w-12" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"/>
                    </svg>
                </div>
                <h3 class="text-xl font-bold mb-2">Enterprise Security</h3>
                <p class="text-gray-600">
                    LDAP/SSO integration, encryption at rest, audit logging,
                    and role-based access control.
                </p>
            </div>
            
            <!-- Feature 5 -->
            <div class="card">
                <div class="text-yellow-600 mb-4">
                    <svg class="h-12 w-12" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4m0 5c0 2.21-3.582 4-8 4s-8-1.79-8-4"/>
                    </svg>
                </div>
                <h3 class="text-xl font-bold mb-2">Multi-Platform Clients</h3>
                <p class="text-gray-600">
                    CLI, TUI, Web, Desktop (Electron/Tauri), and Mobile (Flutter, React Native, Aurora, HarmonyOS).
                </p>
            </div>
            
            <!-- Feature 6 -->
            <div class="card">
                <div class="text-indigo-600 mb-4">
                    <svg class="h-12 w-12" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z"/>
                    </svg>
                </div>
                <h3 class="text-xl font-bold mb-2">AI-Powered Analytics</h3>
                <p class="text-gray-600">
                    Trend analysis, performance optimization, comparison reports,
                    and intelligent recommendations.
                </p>
            </div>
        </div>
    </div>
</section>

<!-- Quick Start Section -->
<section class="py-20 bg-gray-50">
    <div class="container mx-auto px-4">
        <h2 class="text-3xl md:text-4xl font-bold text-center mb-12">
            Get Started in Minutes
        </h2>
        
        <div class="max-w-4xl mx-auto">
            <div class="flex flex-col md:flex-row gap-8">
                <!-- Step 1 -->
                <div class="flex-1 text-center">
                    <div class="w-16 h-16 bg-blue-600 rounded-full flex items-center justify-center text-white text-2xl font-bold mx-auto mb-4">
                        1
                    </div>
                    <h3 class="text-xl font-bold mb-2">Download & Install</h3>
                    <p class="text-gray-600">
                        Choose your platform and install LLM Verifier in seconds.
                        Docker images and source code also available.
                    </p>
                </div>
                
                <!-- Step 2 -->
                <div class="flex-1 text-center">
                    <div class="w-16 h-16 bg-purple-600 rounded-full flex items-center justify-center text-white text-2xl font-bold mx-auto mb-4">
                        2
                    </div>
                    <h3 class="text-xl font-bold mb-2">Configure Providers</h3>
                    <p class="text-gray-600">
                        Add your LLM provider API keys. We support OpenAI, Anthropic,
                        Google, and 10+ more providers.
                    </p>
                </div>
                
                <!-- Step 3 -->
                <div class="flex-1 text-center">
                    <div class="w-16 h-16 bg-green-600 rounded-full flex items-center justify-center text-white text-2xl font-bold mx-auto mb-4">
                        3
                    </div>
                    <h3 class="text-xl font-bold mb-2">Run Verification</h3>
                    <p class="text-gray-600">
                        Execute comprehensive verification and get detailed reports
                        with scores, rankings, and recommendations.
                    </p>
                </div>
            </div>
        </div>
        
        <div class="text-center mt-12">
            <a href="/docs/quick-start" class="btn-primary text-lg px-8 py-4">
                Start Your First Verification →
            </a>
        </div>
    </div>
</section>

<!-- CTA Section -->
<section class="py-20 bg-blue-600 text-white">
    <div class="container mx-auto px-4 text-center">
        <h2 class="text-3xl md:text-4xl font-bold mb-4">
            Ready to Verify Your LLMs?
        </h2>
        <p class="text-xl mb-8 text-blue-100">
            Join thousands of enterprises who trust LLM Verifier for their AI reliability.
        </p>
        <div class="flex flex-col sm:flex-row gap-4 justify-center">
            <a href="/download" class="bg-white text-blue-600 font-bold px-8 py-4 rounded-lg hover:bg-gray-100 transition-colors">
                Download Now
            </a>
            <a href="/docs" class="border-2 border-white font-bold px-8 py-4 rounded-lg hover:bg-white hover:text-blue-600 transition-colors">
                Read Documentation
            </a>
        </div>
    </div>
</section>
```

---

### Day 2: Documentation Portal

**File**: `Website/content/docs/_index.md`

```markdown
---
title: "Documentation"
description: "Complete documentation for LLM Verifier including installation, configuration, usage, API reference, and troubleshooting."
---

# Documentation

Welcome to the comprehensive LLM Verifier documentation.

## Quick Links

- [Getting Started](/docs/getting-started/)
- [User Manual](/docs/user-manual/)
- [API Reference](/docs/api/)
- [Deployment Guide](/docs/deployment/)
- [Troubleshooting](/docs/troubleshooting/)

---

## Documentation Sections

### Getting Started
{{< docs-list "getting-started" >}}

### Configuration
{{< docs-list "configuration" >}}

### Features
{{< docs-list "features" >}}

### API Reference
{{< docs-list "api" >}}

### Deployment
{{< docs-list "deployment" >}}

### Troubleshooting
{{< docs-list "troubleshooting" >}}

---

## Need Help?

- Join our [Discussions](https://github.com/vasic-digital/LLMsVerifier/discussions)
- Search [Issues](https://github.com/vasic-digital/LLMsVerifier/issues)
- Contact [Support](mailto:support@llm-verifier.com)
```

---

### Day 3: Download Center

**File**: `Website/content/download/_index.md`

```markdown
---
title: "Downloads"
description: "Download LLM Verifier for your platform - Linux, macOS, Windows, Docker, or source code."
---

# Downloads

Choose your platform to download LLM Verifier.

## Backend / CLI

### Linux
{{< download-card "llm-verifier-linux-amd64" "Linux (AMD64)" "sha256:..." >}}
{{< download-card "llm-verifier-linux-arm64" "Linux (ARM64)" "sha256:..." >}}

### macOS
{{< download-card "llm-verifier-darwin-amd64" "macOS (Intel)" "sha256:..." >}}
{{< download-card "llm-verifier-darwin-arm64" "macOS (Apple Silicon)" "sha256:..." >}}

### Windows
{{< download-card "llm-verifier-windows-amd64.exe" "Windows" "sha256:..." >}}

### Docker
{{< download-card "llm-verifier:latest" "Docker Image" "docker pull llm-verifier:latest" >}}

### Source Code
{{< download-card "Source Code" "GitHub Repository" "git clone https://github.com/vasic-digital/LLMsVerifier.git" >}}

---

## Desktop Applications

### Electron (Windows/macOS/Linux)
{{< download-card "electron-app" "Download Electron App" "Latest version" >}}

### Tauri (Cross-platform)
{{< download-card "tauri-app" "Download Tauri App" "Latest version" >}}

---

## Web Application

### Angular Web App
{{< download-card "Web App" "Access Web Application" "https://app.llm-verifier.com" >}}

---

## Mobile Applications

### Flutter (iOS/Android)
{{< download-card "flutter-ios" "App Store (iOS)" "Available soon" >}}
{{< download-card "flutter-android" "Google Play (Android)" "Available soon" >}}

### React Native (iOS/Android)
{{< download-card "react-native-ios" "App Store (iOS)" "Available soon" >}}
{{< download-card "react-native-android" "Google Play (Android)" "Available soon" >}}

### Aurora OS
{{< download-card "aurora-os" "Aurora Store" "Available soon" >}}

### Harmony OS
{{< download-card "harmony-os" "AppGallery" "Available soon" >}}

---

## SDKs

### Go SDK
{{< download-card "go-sdk" "Go SDK" "go get github.com/vasic-digital/llm-verifier/sdk/go" >}}

### Python SDK
{{< download-card "python-sdk" "Python SDK" "pip install llm-verifier" >}}

### JavaScript/TypeScript SDK
{{< download-card "js-sdk" "JavaScript SDK" "npm install @llm-verifier/sdk" >}}

---

## Verify Downloads

All downloads include SHA256 checksums for verification.

```bash
# Verify download
sha256sum llm-verifier-linux-amd64
# Compare with provided checksum
```

---

## Previous Versions

Access previous releases from [GitHub Releases](https://github.com/vasic-digital/LLMsVerifier/releases).
```

---

### Day 4-5: Additional Pages

Create these content pages with similar structure:

- **Features**: `/Website/content/features/_index.md`
- **Pricing**: `/Website/content/pricing/_index.md`
- **About**: `/Website/content/about/_index.md`
- **Contact**: `/Website/content/contact/_index.md`
- **Blog**: `/Website/content/blog/_index.md`
- **Videos**: `/Website/content/videos/_index.md`

---

### Day 6-7: Search Functionality

#### Step 1: Create Search API

**File**: `Website/static/api/search.js`

```javascript
// Search index generation (run at build time)
const searchIndex = [
    {
        title: "Getting Started Guide",
        url: "/docs/getting-started/",
        excerpt: "Learn how to install and configure LLM Verifier",
        tags: ["installation", "configuration", "quick-start"]
    },
    // ... add all pages
];

// Fuse.js for fuzzy search
const fuse = new Fuse(searchIndex, {
    keys: ['title', 'excerpt', 'tags'],
    threshold: 0.3,
    includeScore: true
});

function handleSearch(event) {
    const query = event.target.value;
    const results = fuse.search(query);
    displayResults(results);
}

function displayResults(results) {
    const container = document.getElementById('search-results');
    
    if (results.length === 0) {
        container.innerHTML = '<p class="text-gray-500">No results found</p>';
        return;
    }
    
    container.innerHTML = results.map(result => `
        <a href="${result.item.url}" class="block p-4 hover:bg-gray-50 transition-colors border-b">
            <h3 class="font-semibold text-gray-900">${result.item.title}</h3>
            <p class="text-sm text-gray-600">${result.item.excerpt}</p>
        </a>
    `).join('');
}

// Initialize
document.getElementById('search-input').addEventListener('input', handleSearch);
```

---

## WEEK 19: WEBSITE DEPLOYMENT & OPTIMIZATION

### Day 1: GitHub Actions Workflow

**File**: `Website/.github/workflows/deploy.yml`

```yaml
name: Deploy Website

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

permissions:
  contents: read
  pages: write
  id-token: write

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4
        with:
          submodules: recursive
      
      - name: Setup Hugo
        uses: peaceiris/actions-hugo@v2
        with:
          hugo-version: '0.121.2'
          extended: true
      
      - name: Build
        run: hugo --minify
      
      - name: Upload artifact
        uses: actions/upload-pages-artifact@v3
        with:
          path: ./public

  deploy:
    environment:
      name: github-pages
      url: ${{ steps.deployment.outputs.page_url }}
    runs-on: ubuntu-latest
    needs: build
    steps:
      - name: Deploy to GitHub Pages
        id: deployment
        uses: actions/deploy-pages@v4
```

---

### Day 2: Vercel Deployment

**File**: `Website/vercel.json`

```json
{
  "version": 2,
  "builds": [
    {
      "src": "public/**/*",
      "use": "@vercel/static"
    }
  ],
  "routes": [
    {
      "src": "/(.*)",
      "dest": "/$1"
    }
  ],
  "headers": [
    {
      "source": "/(.*)",
      "headers": [
        {
          "key": "X-Content-Type-Options",
          "value": "nosniff"
        },
        {
          "key": "X-Frame-Options",
          "value": "DENY"
        },
        {
          "key": "X-XSS-Protection",
          "value": "1; mode=block"
        }
      ]
    }
  ]
}
```

---

### Day 3: Google Analytics Integration

**File**: `Website/layouts/partials/analytics.html`

```html
<!-- Google Analytics 4 -->
<script async src="https://www.googletagmanager.com/gtag/js?id=G-XXXXXXXXXX"></script>
<script>
  window.dataLayer = window.dataLayer || [];
  function gtag(){dataLayer.push(arguments);}
  gtag('js', new Date());

  gtag('config', 'G-XXXXXXXXXX', {
    'anonymize_ip': true,
    'cookie_flags': 'SameSite=None;Secure'
  });
</script>

<!-- Google Tag Manager -->
<noscript><iframe src="https://www.googletagmanager.com/ns.html?id=GTM-XXXXXXX"
height="0" width="0" style="display:none;visibility:hidden"></iframe></noscript>
<script>(function(w,d,s,l,i){w[l]=w[l]||[];w[l].push({'gtm.start':
new Date().getTime(),event:'gtm.js'});var f=d.getElementsByTagName(s)[0],
j=d.createElement(s),dl=l!='dataLayer'?'&l='+l:'';j.async=true;j.src=
'https://www.googletagmanager.com/gtm.js?id='+i+dl;f.parentNode.insertBefore(j,f);
})(window,document,'script','dataLayer','GTM-XXXXXXX');</script>

<!-- Privacy compliance (consent banner) -->
{{ partial "consent-banner.html" . }}
```

---

### Day 4-5: SEO Optimization

**File**: `Website/layouts/partials/seo.html`

```html
<!-- Meta Tags -->
<meta name="description" content="{{ .Params.description | default .Site.Params.description }}">
<meta name="keywords" content="{{ .Params.keywords | default .Site.Params.keywords }}">
<meta name="author" content="LLM Verifier Team">

<!-- Open Graph -->
<meta property="og:title" content="{{ .Title }}">
<meta property="og:description" content="{{ .Params.description | default .Site.Params.description }}">
<meta property="og:type" content="website">
<meta property="og:url" content="{{ .Permalink }}">
<meta property="og:image" content="{{ .Params.og_image | default .Site.BaseURL }}/images/og-default.png">
<meta property="og:site_name" content="LLM Verifier">

<!-- Twitter Card -->
<meta name="twitter:card" content="summary_large_image">
<meta name="twitter:site" content="@llmverifier">
<meta name="twitter:creator" content="@llmverifier">
<meta name="twitter:title" content="{{ .Title }}">
<meta name="twitter:description" content="{{ .Params.description | default .Site.Params.description }}">
<meta name="twitter:image" content="{{ .Params.og_image | default .Site.BaseURL }}/images/og-default.png">

<!-- Canonical URL -->
<link rel="canonical" href="{{ .Permalink }}">

<!-- Structured Data (JSON-LD) -->
<script type="application/ld+json">
{
  "@context": "https://schema.org",
  "@type": "SoftwareApplication",
  "name": "LLM Verifier",
  "description": "{{ .Params.description | default .Site.Params.description }}",
  "url": "{{ .Permalink }}",
  "applicationCategory": "BusinessApplication",
  "operatingSystem": "Linux, macOS, Windows, iOS, Android",
  "offers": {
    "@type": "Offer",
    "price": "0",
    "priceCurrency": "USD"
  },
  "author": {
    "@type": "Organization",
    "name": "LLM Verifier Team",
    "url": "https://github.com/vasic-digital/LLMsVerifier"
  }
}
</script>

<!-- Robots.txt -->
{{ if .IsHome }}
<meta name="robots" content="index, follow">
{{ else }}
<meta name="robots" content="noindex, follow">
{{ end }}

<!-- Sitemap -->
<link rel="sitemap" type="application/xml" href="{{ .Site.BaseURL }}sitemap.xml">
```

---

### Day 6-7: Performance Optimization

**File**: `Website/config.toml`

```toml
[build]
  minify = true
  writeStats = true
  useResourceCache = true

[minify]
  minifyOutput = true
  disableCSS = false
  disableHTML = false
  disableJS = false
  disableJSON = false
  disableSVG = false
  disableXML = false

[caches]
  getresource = true
  getjson = true
  images = true

[imaging]
  resampleFilter = "Lanczos"
  quality = 80
  anchor = "smart"

[[module.mounts]]
  source = "static"
  target = "static"
```

---

## WEEK 20: WEBSITE FINALIZATION

### Day 1: Cross-Browser Testing

**Testing Checklist**:
- [ ] Chrome (latest, last 2 versions)
- [ ] Firefox (latest, last 2 versions)
- [ ] Safari (latest macOS, iOS)
- [ ] Edge (latest)
- [ ] Mobile browsers (Chrome Mobile, Safari Mobile)

### Day 2: Mobile Responsiveness Testing

**Devices to Test**:
- iPhone SE, iPhone 12, iPhone 14 Pro Max
- Samsung Galaxy S21, Google Pixel 6
- iPad Pro, iPad Mini
- Android tablets

### Day 3: Accessibility Testing

**WCAG 2.1 Level AA Checklist**:
- [ ] Keyboard navigation works
- [ ] Focus indicators visible
- [ ] Color contrast ratio > 4.5:1
- [ ] Alt text for images
- [ ] ARIA labels on interactive elements
- [ ] Screen reader compatible
- [ ] Skip to content link

### Day 4: Website Screenshot Gallery

**Screenshots to Capture**:
1. Home page hero section
2. Home page features section
3. Documentation portal
4. Download center
5. Mobile views (3 sizes each)
6. Dark mode (if implemented)

### Day 5: Final Website QA

**QA Checklist**:
- [ ] All links work
- [ ] No 404 errors
- [ ] Forms submit correctly
- [ ] Search functions properly
- [ ] Mobile menu works
- [ ] Dark mode toggle works
- [ ] Analytics tracking works
- [ ] No console errors
- [ ] Performance score > 90 (Lighthouse)

### Day 6-7: Launch Preparation

**Launch Checklist**:
- [ ] DNS configured
- [ ] SSL certificate installed
- [ ] CDN configured
- [ ] Analytics configured
- [ ] SEO meta tags complete
- [ ] Sitemap submitted to Google
- [ ] Robots.txt configured
- [ ] Social media accounts linked
- [ ] Favicon and app icons created
- [ ] 404 page created
- [ ] Backups configured
- [ ] Monitoring/alerts configured
- [ ] Load testing completed

---

## WEBSITE METRICS

### Success Criteria

- **Lighthouse Performance**: >90
- **Lighthouse Accessibility**: >95
- **Lighthouse SEO**: >100
- **Lighthouse Best Practices**: >95
- **Page Load Time**: <3 seconds
- **First Contentful Paint**: <1.5 seconds
- **Time to Interactive**: <3 seconds
- **Mobile Responsiveness**: 100% devices supported
- **Browser Compatibility**: Chrome, Firefox, Safari, Edge
- **Accessibility**: WCAG 2.1 AA compliant

---

## NEXT STEPS

After completing website modernization:

1. **Deploy to production** (Week 20)
2. **Monitor analytics** (Week 21+)
3. **Gather user feedback** (Week 22+)
4. **Iterate and improve** (Ongoing)

---

*Website modernization complete!*
