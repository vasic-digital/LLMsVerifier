// LLMsVerifier Website JavaScript

document.addEventListener('DOMContentLoaded', function() {
    // Smooth scrolling for navigation links
    document.querySelectorAll('a[href^="#"]').forEach(anchor => {
        anchor.addEventListener('click', function(e) {
            e.preventDefault();
            const target = document.querySelector(this.getAttribute('href'));
            if (target) {
                target.scrollIntoView({
                    behavior: 'smooth',
                    block: 'start'
                });
            }
        });
    });

    // Navbar background change on scroll
    const header = document.querySelector('.header');
    window.addEventListener('scroll', () => {
        if (window.scrollY > 100) {
            header.style.background = 'rgba(26, 82, 118, 0.95)';
        } else {
            header.style.background = 'linear-gradient(135deg, #1a5276, #48c9b0)';
        }
    });

    // Animate stats on scroll
    const observerOptions = {
        threshold: 0.5
    };

    const observer = new IntersectionObserver((entries) => {
        entries.forEach(entry => {
            if (entry.isIntersecting) {
                entry.target.classList.add('visible');
                animateValue(entry.target);
            }
        });
    }, observerOptions);

    document.querySelectorAll('.stat-number').forEach(stat => {
        observer.observe(stat);
    });

    function animateValue(element) {
        const text = element.textContent;
        const isPercentage = text.includes('%');
        const hasPlus = text.includes('+');
        const numericValue = parseInt(text.replace(/[^0-9]/g, ''));

        if (isNaN(numericValue)) return;

        let current = 0;
        const duration = 2000;
        const increment = numericValue / (duration / 16);

        const timer = setInterval(() => {
            current += increment;
            if (current >= numericValue) {
                clearInterval(timer);
                current = numericValue;
            }
            let displayValue = Math.floor(current);
            if (hasPlus) displayValue += '+';
            if (isPercentage) displayValue += '%';
            element.textContent = displayValue;
        }, 16);
    }

    // Add loading animation for feature cards
    const cards = document.querySelectorAll('.feature-card, .doc-card');
    const cardObserver = new IntersectionObserver((entries) => {
        entries.forEach((entry, index) => {
            if (entry.isIntersecting) {
                setTimeout(() => {
                    entry.target.style.opacity = '1';
                    entry.target.style.transform = 'translateY(0)';
                }, index * 100);
            }
        });
    }, { threshold: 0.1 });

    cards.forEach(card => {
        card.style.opacity = '0';
        card.style.transform = 'translateY(20px)';
        card.style.transition = 'opacity 0.5s, transform 0.5s';
        cardObserver.observe(card);
    });

    console.log('LLMsVerifier website loaded successfully');
});
