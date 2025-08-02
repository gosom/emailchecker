function emailChecker() {
    return {
        email: '',
        result: null,
        loading: false,
        error: null,
        showSuggestions: false,
        showJson: false,
        responseTime: null,
        notification: '',
        
        suggestions: [
            'test_user@gmail.com',
            'p303200@uoa.gr', 
            'zerile@forexzig.com',
            'qwer1234ksnf45ms92@hotmail.com'
        ],

        init() {
            this.setupLoadingAnimation();
            
            document.addEventListener('keydown', (e) => {
                if (e.key === 'Escape') {
                    this.clearAll();
                }
                
                if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
                    e.preventDefault();
                    const emailInput = document.querySelector('input[type="email"]');
                    if (emailInput) {
                        emailInput.focus();
                        emailInput.select();
                    }
                }
            });
        },

        async checkEmail() {
            if (!this.email.trim()) return;
            
            this.clearStates();
            this.loading = true;
            const startTime = Date.now();
            
            try {
                const response = await fetch(`/check/${encodeURIComponent(this.email)}`);
                const data = await response.json();
                
                if (!response.ok) {
                    throw new Error(data.message || 'Failed to check email');
                }
                
                this.result = data;
                this.responseTime = Date.now() - startTime;
                
                setTimeout(() => {
                    document.querySelector('[x-show="result"]')?.scrollIntoView({ 
                        behavior: 'smooth', 
                        block: 'start' 
                    });
                }, 100);
                
            } catch (err) {
                this.error = err.message;
            } finally {
                this.loading = false;
            }
        },

        selectSuggestion(suggestion) {
            this.email = suggestion;
            this.showSuggestions = false;
            setTimeout(() => this.checkEmail(), 100);
        },

        clearAll() {
            this.email = '';
            this.clearStates();
        },

        clearStates() {
            this.result = null;
            this.error = null;
            this.showJson = false;
            this.responseTime = null;
            this.showSuggestions = false;
        },

        copyToClipboard() {
            if (!this.result) return;
            
            const jsonString = JSON.stringify(this.result, null, 2);
            
            if (navigator.clipboard && window.isSecureContext) {
                navigator.clipboard.writeText(jsonString).then(() => {
                    this.showNotification('📋 JSON copied to clipboard!');
                }).catch((err) => {
                    console.error('Clipboard API failed:', err);
                    this.fallbackCopy(jsonString);
                });
            } else {
                this.fallbackCopy(jsonString);
            }
        },

        fallbackCopy(text) {
            const textarea = document.createElement('textarea');
            textarea.value = text;
            textarea.style.position = 'fixed';
            textarea.style.left = '-9999px';
            textarea.style.opacity = '0';
            document.body.appendChild(textarea);
            
            try {
                textarea.focus();
                textarea.select();
                const successful = document.execCommand('copy');
                if (successful) {
                    this.showNotification('📋 JSON copied to clipboard!');
                } else {
                    this.showNotification('❌ Copy failed - please copy manually');
                }
            } catch (err) {
                console.error('Fallback copy failed:', err);
                this.showNotification('❌ Copy failed - please copy manually');
            } finally {
                document.body.removeChild(textarea);
            }
        },

        showNotification(message) {
            this.notification = message;
            setTimeout(() => {
                this.notification = '';
            }, 3000);
        },

        getRiskEmoji(riskLevel) {
            const emojis = {
                'low': '✅',
                'medium': '⚠️',
                'high': '🚫'
            };
            return emojis[riskLevel] || '❓';
        },

        getRiskClass(riskLevel) {
            return `risk-${riskLevel} px-4 py-2 rounded font-medium text-base`;
        },

        getBorderClass(riskLevel) {
            const classes = {
                'low': 'border-green-500',
                'medium': 'border-yellow-500', 
                'high': 'border-red-500'
            };
            return classes[riskLevel] || 'border-gray-500';
        },

        getQuickSummary() {
            if (!this.result) return '';
            
            const checks = [];
            if (this.result.disposable?.checked) {
                checks.push(`Disposable: ${this.result.disposable.value ? '❌' : '✅'}`);
            }
            if (this.result.dns?.checked) {
                checks.push(`DNS: ${this.result.dns.value?.has_mx ? '✅' : '❌'}`);
            }
            if (this.result.well_known?.checked) {
                checks.push(`Known provider: ${this.result.well_known.value ? '✅' : '❓'}`);
            }
            if (this.result.educational?.checked && this.result.educational.value) {
                checks.push(`Educational: ✅`);
            }
            
            return checks.length ? checks.join(' • ') : '';
        },

        formatJSON(obj) {
            if (!obj) return '';
            
            return JSON.stringify(obj, null, 2)
                .replace(/"([^"]+)":/g, '<span class="key">"$1"</span>:')
                .replace(/: "([^"]*)"/g, ': <span class="string">"$1"</span>')
                .replace(/: (\d+\.?\d*)/g, ': <span class="number">$1</span>')
                .replace(/: (true|false)/g, ': <span class="boolean">$1</span>')
                .replace(/: null/g, ': <span class="null">null</span>');
        },

        setupLoadingAnimation() {
            const dots = ['   ', '.  ', '.. ', '...'];
            const spinnerChars = ['⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'];
            let index = 0;
            
            setInterval(() => {
                const spinner = document.querySelector('.loading-spinner');
                const loadingText = document.querySelector('.loading-text');
                
                if (spinner && this.loading) {
                    spinner.textContent = spinnerChars[index];
                    if (loadingText) {
                        loadingText.textContent = `Analyzing email${dots[index % 4]}`;
                    }
                    index = (index + 1) % spinnerChars.length;
                }
            }, 100);
        }
    }
}
