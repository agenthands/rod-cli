(function() {
    return new Promise((resolve) => {
        // --- Styles ---
        const style = document.createElement('style');
        style.id = 'rod-annotator-styles';
        style.textContent = `
            @import url('https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap');
            
            #rod-annotator-panel {
                position: fixed;
                bottom: 24px;
                right: 24px;
                width: 340px;
                background: rgba(20, 20, 25, 0.85);
                backdrop-filter: blur(16px);
                -webkit-backdrop-filter: blur(16px);
                border: 1px solid rgba(255, 255, 255, 0.1);
                border-radius: 16px;
                box-shadow: 0 20px 40px rgba(0, 0, 0, 0.4), inset 0 1px 0 rgba(255, 255, 255, 0.1);
                color: #fff;
                font-family: 'Inter', system-ui, -apple-system, sans-serif;
                z-index: 2147483647;
                display: flex;
                flex-direction: column;
                overflow: hidden;
                animation: rod-slide-up 0.4s cubic-bezier(0.16, 1, 0.3, 1);
            }

            @keyframes rod-slide-up {
                from { opacity: 0; transform: translateY(20px) scale(0.95); }
                to { opacity: 1; transform: translateY(0) scale(1); }
            }

            #rod-annotator-header {
                padding: 16px 20px;
                border-bottom: 1px solid rgba(255, 255, 255, 0.08);
                display: flex;
                align-items: center;
                justify-content: space-between;
                background: linear-gradient(90deg, rgba(255,255,255,0.03) 0%, rgba(255,255,255,0) 100%);
            }

            #rod-annotator-header h2 {
                margin: 0;
                font-size: 15px;
                font-weight: 600;
                letter-spacing: -0.01em;
                display: flex;
                align-items: center;
                gap: 8px;
            }

            #rod-annotator-header h2::before {
                content: '';
                display: block;
                width: 8px;
                height: 8px;
                background: #4ade80;
                border-radius: 50%;
                box-shadow: 0 0 12px #4ade80;
            }

            #rod-annotator-body {
                padding: 16px 20px;
                max-height: 300px;
                overflow-y: auto;
            }

            #rod-annotator-body p {
                font-size: 13px;
                color: rgba(255, 255, 255, 0.6);
                margin: 0 0 12px 0;
                line-height: 1.5;
            }

            #rod-annotator-list {
                display: flex;
                flex-direction: column;
                gap: 8px;
            }

            .rod-annotation-item {
                background: rgba(255, 255, 255, 0.05);
                border: 1px solid rgba(255, 255, 255, 0.05);
                border-radius: 8px;
                padding: 10px 12px;
                display: flex;
                justify-content: space-between;
                align-items: center;
                font-size: 13px;
                transition: all 0.2s;
            }

            .rod-annotation-item:hover {
                background: rgba(255, 255, 255, 0.08);
                border-color: rgba(255, 255, 255, 0.1);
            }

            .rod-annotation-item-label {
                font-weight: 500;
                color: #e2e8f0;
            }

            .rod-annotation-item-delete {
                background: none;
                border: none;
                color: #f87171;
                cursor: pointer;
                opacity: 0.6;
                transition: opacity 0.2s;
                padding: 4px;
                display: flex;
                align-items: center;
                justify-content: center;
            }

            .rod-annotation-item-delete:hover {
                opacity: 1;
            }

            #rod-annotator-footer {
                padding: 16px 20px;
                border-top: 1px solid rgba(255, 255, 255, 0.08);
                display: flex;
                gap: 12px;
                background: rgba(0, 0, 0, 0.2);
            }

            .rod-btn {
                flex: 1;
                padding: 10px;
                border: none;
                border-radius: 8px;
                font-size: 13px;
                font-weight: 600;
                font-family: inherit;
                cursor: pointer;
                transition: all 0.2s;
            }

            .rod-btn-cancel {
                background: rgba(255, 255, 255, 0.08);
                color: #fff;
            }

            .rod-btn-cancel:hover {
                background: rgba(255, 255, 255, 0.15);
            }

            .rod-btn-done {
                background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%);
                color: #fff;
                box-shadow: 0 4px 12px rgba(99, 102, 241, 0.3);
            }

            .rod-btn-done:hover {
                box-shadow: 0 6px 16px rgba(99, 102, 241, 0.5);
                transform: translateY(-1px);
            }

            .rod-hover-highlight {
                position: absolute;
                border: 2px dashed #6366f1;
                background: rgba(99, 102, 241, 0.15);
                box-shadow: 0 0 0 2px rgba(99, 102, 241, 0.2);
                border-radius: 4px;
                pointer-events: none;
                z-index: 2147483646;
                transition: all 0.1s ease-out;
            }

            .rod-hover-tooltip {
                position: absolute;
                background: #6366f1;
                color: white;
                font-family: 'Inter', sans-serif;
                font-size: 11px;
                font-weight: 600;
                padding: 4px 8px;
                border-radius: 4px;
                top: -24px;
                left: -2px;
                white-space: nowrap;
                pointer-events: none;
            }

            .rod-persistent-highlight {
                position: absolute;
                border: 2px solid #10b981;
                background: rgba(16, 185, 129, 0.1);
                border-radius: 4px;
                pointer-events: none;
                z-index: 2147483645;
            }

            .rod-persistent-tooltip {
                position: absolute;
                background: #10b981;
                color: white;
                font-family: 'Inter', sans-serif;
                font-size: 11px;
                font-weight: 600;
                padding: 4px 8px;
                border-radius: 4px;
                top: -24px;
                left: -2px;
                white-space: nowrap;
                pointer-events: none;
            }
            
            #rod-prompt-overlay {
                position: fixed;
                top: 0; left: 0; right: 0; bottom: 0;
                background: rgba(0,0,0,0.6);
                backdrop-filter: blur(4px);
                z-index: 2147483648;
                display: flex;
                align-items: center;
                justify-content: center;
                opacity: 0;
                pointer-events: none;
                transition: opacity 0.2s;
            }
            
            #rod-prompt-overlay.active {
                opacity: 1;
                pointer-events: auto;
            }
            
            .rod-prompt-box {
                background: #1a1a20;
                border: 1px solid rgba(255,255,255,0.1);
                padding: 24px;
                border-radius: 16px;
                width: 320px;
                box-shadow: 0 24px 48px rgba(0,0,0,0.5);
                transform: scale(0.95);
                transition: transform 0.2s cubic-bezier(0.16, 1, 0.3, 1);
            }
            
            #rod-prompt-overlay.active .rod-prompt-box {
                transform: scale(1);
            }
            
            .rod-prompt-box h3 {
                margin: 0 0 16px 0;
                color: #fff;
                font-family: 'Inter', sans-serif;
                font-size: 16px;
                font-weight: 600;
            }
            
            .rod-prompt-box input {
                width: 100%;
                box-sizing: border-box;
                background: rgba(0,0,0,0.3);
                border: 1px solid rgba(255,255,255,0.2);
                padding: 12px;
                border-radius: 8px;
                color: #fff;
                font-family: inherit;
                font-size: 14px;
                margin-bottom: 20px;
                outline: none;
                transition: border-color 0.2s;
            }
            
            .rod-prompt-box input:focus {
                border-color: #6366f1;
            }
            
            .rod-prompt-actions {
                display: flex;
                gap: 12px;
            }
        `;
        document.head.appendChild(style);

        // --- State ---
        let annotations = [];
        let isPrompting = false;
        let currentTarget = null;
        let hoverBox = null;
        let hoverTooltip = null;

        // --- DOM Elements ---
        const panel = document.createElement('div');
        panel.id = 'rod-annotator-panel';
        panel.innerHTML = \`
            <div id="rod-annotator-header">
                <h2>Visual Annotation Mode</h2>
            </div>
            <div id="rod-annotator-body">
                <p>Hover and click elements to label them. Click Done when finished.</p>
                <div id="rod-annotator-list"></div>
            </div>
            <div id="rod-annotator-footer">
                <button class="rod-btn rod-btn-cancel" id="rod-btn-cancel">Cancel</button>
                <button class="rod-btn rod-btn-done" id="rod-btn-done">Done</button>
            </div>
        \`;
        document.body.appendChild(panel);
        
        const promptOverlay = document.createElement('div');
        promptOverlay.id = 'rod-prompt-overlay';
        promptOverlay.innerHTML = \`
            <div class="rod-prompt-box">
                <h3>Label Element</h3>
                <input type="text" id="rod-prompt-input" placeholder="e.g. submit_button" autocomplete="off" />
                <div class="rod-prompt-actions">
                    <button class="rod-btn rod-btn-cancel" id="rod-prompt-cancel">Cancel</button>
                    <button class="rod-btn rod-btn-done" id="rod-prompt-save">Save</button>
                </div>
            </div>
        \`;
        document.body.appendChild(promptOverlay);

        const promptInput = document.getElementById('rod-prompt-input');
        
        function renderList() {
            const list = document.getElementById('rod-annotator-list');
            list.innerHTML = '';
            
            // clear all persistent highlights
            document.querySelectorAll('.rod-persistent-highlight').forEach(el => el.remove());

            annotations.forEach((ann, i) => {
                const item = document.createElement('div');
                item.className = 'rod-annotation-item';
                item.innerHTML = \`
                    <span class="rod-annotation-item-label">\${ann.label}</span>
                    <button class="rod-annotation-item-delete" data-idx="\${i}">
                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg>
                    </button>
                \`;
                list.appendChild(item);
                
                // Draw persistent bounding box
                if(ann.rect) {
                    const box = document.createElement('div');
                    box.className = 'rod-persistent-highlight';
                    box.style.top = ann.rect.top + window.scrollY + 'px';
                    box.style.left = ann.rect.left + window.scrollX + 'px';
                    box.style.width = ann.rect.width + 'px';
                    box.style.height = ann.rect.height + 'px';
                    
                    const tooltip = document.createElement('div');
                    tooltip.className = 'rod-persistent-tooltip';
                    tooltip.textContent = ann.label;
                    box.appendChild(tooltip);
                    
                    document.body.appendChild(box);
                }
            });

            document.querySelectorAll('.rod-annotation-item-delete').forEach(btn => {
                btn.onclick = (e) => {
                    const idx = parseInt(e.currentTarget.getAttribute('data-idx'));
                    annotations.splice(idx, 1);
                    renderList();
                };
            });
        }

        // --- Interaction ---
        function generateSelector(el) {
            if (el.id) return '#' + el.id;
            if (el.className && typeof el.className === 'string') {
                const classes = el.className.trim().split(/\\s+/).filter(c => !c.startsWith('rod-'));
                if (classes.length > 0) return el.tagName.toLowerCase() + '.' + classes.join('.');
            }
            return el.tagName.toLowerCase();
        }

        function createHoverBox() {
            if(!hoverBox) {
                hoverBox = document.createElement('div');
                hoverBox.className = 'rod-hover-highlight';
                hoverTooltip = document.createElement('div');
                hoverTooltip.className = 'rod-hover-tooltip';
                hoverBox.appendChild(hoverTooltip);
                document.body.appendChild(hoverBox);
            }
        }

        function onMouseMove(e) {
            if (isPrompting) return;
            const target = e.target;
            if (panel.contains(target) || promptOverlay.contains(target)) {
                if(hoverBox) hoverBox.style.display = 'none';
                return;
            }
            
            createHoverBox();
            hoverBox.style.display = 'block';
            
            const rect = target.getBoundingClientRect();
            hoverBox.style.top = rect.top + window.scrollY + 'px';
            hoverBox.style.left = rect.left + window.scrollX + 'px';
            hoverBox.style.width = rect.width + 'px';
            hoverBox.style.height = rect.height + 'px';
            
            hoverTooltip.textContent = generateSelector(target);
        }

        function onClick(e) {
            if (isPrompting || panel.contains(e.target) || promptOverlay.contains(e.target)) return;
            e.preventDefault();
            e.stopPropagation();
            
            currentTarget = e.target;
            if(hoverBox) hoverBox.style.display = 'none';
            
            isPrompting = true;
            promptOverlay.classList.add('active');
            promptInput.value = '';
            promptInput.focus();
        }

        document.addEventListener('mousemove', onMouseMove, true);
        document.addEventListener('click', onClick, true);

        // --- Prompt Handlers ---
        function closePrompt() {
            isPrompting = false;
            promptOverlay.classList.remove('active');
            currentTarget = null;
        }

        document.getElementById('rod-prompt-save').onclick = () => {
            const label = promptInput.value.trim();
            if (label && currentTarget) {
                const rect = currentTarget.getBoundingClientRect();
                annotations.push({
                    label: label,
                    selector: generateSelector(currentTarget),
                    rect: {
                        top: rect.top,
                        left: rect.left,
                        width: rect.width,
                        height: rect.height
                    }
                });
                renderList();
            }
            closePrompt();
        };

        document.getElementById('rod-prompt-cancel').onclick = closePrompt;
        promptInput.onkeydown = (e) => {
            if (e.key === 'Enter') document.getElementById('rod-prompt-save').click();
            if (e.key === 'Escape') closePrompt();
        };

        // --- Completion ---
        function cleanup() {
            document.removeEventListener('mousemove', onMouseMove, true);
            document.removeEventListener('click', onClick, true);
            if(hoverBox) hoverBox.remove();
            document.querySelectorAll('.rod-persistent-highlight').forEach(el => el.remove());
            panel.remove();
            promptOverlay.remove();
            style.remove();
        }

        document.getElementById('rod-btn-cancel').onclick = () => {
            cleanup();
            resolve({ cancelled: true });
        };

        document.getElementById('rod-btn-done').onclick = () => {
            cleanup();
            resolve({ annotations: annotations });
        };
    });
})();
