/**
 * LLM Verifier Website - Main JavaScript
 * Version: 2.0.0
 */

(function() {
    'use strict';

    // Configuration
    const config = {
        // Analytics - Replace with actual IDs in production
        analytics: {
            googleAnalyticsId: null, // e.g., 'G-XXXXXXXXXX'
            clarityProjectId: null   // e.g., 'abcdefghij'
        }
    };

    // Initialize when DOM is ready
    document.addEventListener('DOMContentLoaded', function() {
        initNavigation();
        initSmoothScroll();
        initAnalytics();
    });

    /**
     * Initialize navigation functionality
     */
    function initNavigation() {
        const navbar = document.querySelector('.navbar');

        // Add scrolled class on scroll
        window.addEventListener('scroll', function() {
            if (window.scrollY > 50) {
                navbar.classList.add('scrolled');
            } else {
                navbar.classList.remove('scrolled');
            }
        });

        // Mobile menu toggle (if needed)
        const menuToggle = document.querySelector('.nav-menu-toggle');
        const navMenu = document.querySelector('.nav-menu');

        if (menuToggle && navMenu) {
            menuToggle.addEventListener('click', function() {
                navMenu.classList.toggle('active');
            });
        }
    }

    /**
     * Initialize smooth scrolling for anchor links
     */
    function initSmoothScroll() {
        document.querySelectorAll('a[href^="#"]').forEach(function(anchor) {
            anchor.addEventListener('click', function(e) {
                const targetId = this.getAttribute('href');
                if (targetId === '#') return;

                const target = document.querySelector(targetId);
                if (target) {
                    e.preventDefault();
                    target.scrollIntoView({
                        behavior: 'smooth',
                        block: 'start'
                    });
                }
            });
        });
    }

    /**
     * Initialize analytics services
     */
    function initAnalytics() {
        // Google Analytics 4
        if (config.analytics.googleAnalyticsId) {
            loadGoogleAnalytics(config.analytics.googleAnalyticsId);
        }

        // Microsoft Clarity
        if (config.analytics.clarityProjectId) {
            loadClarity(config.analytics.clarityProjectId);
        }

        // Log page view (basic analytics)
        logPageView();
    }

    /**
     * Load Google Analytics 4
     * @param {string} measurementId - GA4 Measurement ID
     */
    function loadGoogleAnalytics(measurementId) {
        const script = document.createElement('script');
        script.async = true;
        script.src = 'https://www.googletagmanager.com/gtag/js?id=' + measurementId;
        document.head.appendChild(script);

        window.dataLayer = window.dataLayer || [];
        function gtag() { window.dataLayer.push(arguments); }
        window.gtag = gtag;
        gtag('js', new Date());
        gtag('config', measurementId);
    }

    /**
     * Load Microsoft Clarity
     * @param {string} projectId - Clarity Project ID
     */
    function loadClarity(projectId) {
        (function(c, l, a, r, i, t, y) {
            c[a] = c[a] || function() { (c[a].q = c[a].q || []).push(arguments); };
            t = l.createElement(r); t.async = 1; t.src = "https://www.clarity.ms/tag/" + i;
            y = l.getElementsByTagName(r)[0]; y.parentNode.insertBefore(t, y);
        })(window, document, "clarity", "script", projectId);
    }

    /**
     * Log page view (basic analytics without external services)
     */
    function logPageView() {
        const pageData = {
            url: window.location.href,
            path: window.location.pathname,
            title: document.title,
            timestamp: new Date().toISOString(),
            referrer: document.referrer || 'direct',
            userAgent: navigator.userAgent
        };

        // Log to console in development
        console.log('LLM Verifier Page View:', pageData.path);
    }

    /**
     * Track custom events
     * @param {string} category - Event category
     * @param {string} action - Event action
     * @param {string} label - Event label (optional)
     */
    window.trackEvent = function(category, action, label) {
        // Google Analytics
        if (window.gtag) {
            window.gtag('event', action, {
                event_category: category,
                event_label: label
            });
        }

        // Console logging
        console.log('Event:', { category: category, action: action, label: label });
    };

})();
