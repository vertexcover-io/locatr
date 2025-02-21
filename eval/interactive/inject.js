window.Captured = [];

// Check if elementSelector already exists to prevent duplicate declarations
if (typeof window.elementSelector === 'undefined') {
    window.elementSelector = {
        isEnabled: false,
        mouseOverColor: 'blue',
        clickColor: 'red',
        originalOutline: '',
        selectedElement: null,

        isPartOfInputBox: function(element) {
            const inputBox = document.getElementById('element-input-box');
            return inputBox && (inputBox === element || inputBox.contains(element));
        },

        handleMouseOver: async function(e) {
            if (this.isPartOfInputBox(e.target)) {
                return; // Ignore elements within the input box
            }
            e.preventDefault();
            e.stopPropagation();
            this.originalOutline = e.target.style.outline;
            e.target.style.outline = `2px solid ${this.mouseOverColor}`;
        },

        handleMouseOut: async function(e) {
            if (this.isPartOfInputBox(e.target)) {
                return; // Ignore elements within the input box
            }
            e.preventDefault();
            e.stopPropagation();
            e.target.style.outline = this.originalOutline;
        },

        handleClick: async function(e) {
            if (this.isPartOfInputBox(e.target)) {
                return; // Ignore elements within the input box
            }
            e.preventDefault();
            e.stopPropagation();
            
            this.selectedElement = e.target;
            this.selectedElement.style.outline = `5px solid ${this.clickColor}`;
            await this.disable();
            
            // Enable input and update toggle button
            const descriptionInput = document.getElementById('element-description-input');
            const toggleButton = document.getElementById('element-toggle-button');
            const captureButton = document.getElementById('element-capture-button');
            
            if (descriptionInput && toggleButton && captureButton) {
                descriptionInput.disabled = false;
                this.updateToggleButton();
                this.checkInput();
            }
        },

        updateToggleButton: function() {
            const toggleButton = document.getElementById('element-toggle-button');
            if (toggleButton) {
                if (this.selectedElement) {
                    toggleButton.textContent = 'Unselect Element';
                    toggleButton.style.backgroundColor = '#EAB308'; // Yellow color for unselect
                } else if (this.isEnabled) {
                    toggleButton.textContent = 'Disable Element Selector';
                    toggleButton.style.backgroundColor = '#E53E3E'; // Red for disable
                } else {
                    toggleButton.textContent = 'Enable Element Selector';
                    toggleButton.style.backgroundColor = '#4299E1'; // Blue for enable
                }
            }
        },

        unselectElement: function() {
            if (this.selectedElement) {
                this.selectedElement.style.outline = this.originalOutline;
                this.selectedElement = null;
                
                const descriptionInput = document.getElementById('element-description-input');
                if (descriptionInput) {
                    descriptionInput.disabled = true;
                    descriptionInput.value = '';
                }
                
                this.checkInput();
                this.updateToggleButton();
            }
        },

        enable: async function() {
            if (!this.isEnabled) {
                this.boundMouseOver = this.handleMouseOver.bind(this);
                this.boundMouseOut = this.handleMouseOut.bind(this);
                this.boundClick = this.handleClick.bind(this);

                document.addEventListener('mouseover', this.boundMouseOver, true);
                document.addEventListener('mouseout', this.boundMouseOut, true);
                document.addEventListener('click', this.boundClick, true);

                document.body.style.cursor = 'grab';
                this.isEnabled = true;
                this.updateToggleButton();
            }
        },

        disable: async function() {
            if (this.isEnabled) {
                document.removeEventListener('mouseover', this.boundMouseOver, true);
                document.removeEventListener('mouseout', this.boundMouseOut, true);
                document.removeEventListener('click', this.boundClick, true);

                document.body.style.cursor = '';
                this.isEnabled = false;
                this.updateToggleButton();
            }
        },

        checkInput: function() {
            const descriptionInput = document.getElementById('element-description-input');
            const captureButton = document.getElementById('element-capture-button');
            
            if (descriptionInput && captureButton) {
                const isEnabled = descriptionInput.value.trim() && this.selectedElement;
                captureButton.disabled = !isEnabled;
                captureButton.style.backgroundColor = isEnabled ? '#4299E1' : '#CBD5E0';
                captureButton.style.cursor = isEnabled ? 'pointer' : 'not-allowed';
                
                descriptionInput.style.borderColor = descriptionInput.value.trim() ? '#CBD5E0' : '#E53E3E';
                descriptionInput.style.backgroundColor = descriptionInput.value.trim() ? 'white' : '#FFF5F5';
            }
        },

        createInputBox: function() {
            if (document.getElementById('element-input-box')) {
                return; // Input box already exists
            }

            const inputBox = document.createElement('div');
            inputBox.id = 'element-input-box';
            inputBox.style.position = 'fixed';
            inputBox.style.left = '20px';
            inputBox.style.top = '20px';
            inputBox.style.background = '#f8fafc';
            inputBox.style.border = '1px solid #e2e8f0';
            inputBox.style.borderRadius = '8px';
            inputBox.style.padding = '20px';
            inputBox.style.zIndex = '9999';
            inputBox.style.width = '300px';
            inputBox.style.maxWidth = `${window.innerWidth * 0.25}px`;
            inputBox.style.boxShadow = '0 4px 6px rgba(0, 0, 0, 0.1)';

            // Add draggable header
            const header = document.createElement('div');
            header.style.cursor = 'move';
            header.style.padding = '10px';
            header.style.marginBottom = '15px';
            header.style.borderBottom = '1px solid #e2e8f0';
            header.style.display = 'flex';
            header.style.justifyContent = 'space-between';
            header.style.alignItems = 'center';
            header.innerHTML = '<span style="font-weight: 600; color: #2d3748;">Element Description</span>';

            // Create description input
            const descriptionInput = document.createElement('textarea');
            descriptionInput.id = 'element-description-input';
            descriptionInput.placeholder = 'Enter description (required)';
            descriptionInput.disabled = true; // Initially disabled
            descriptionInput.style.cssText = `
                width: 100%;
                padding: 8px 12px;
                margin: 8px 0;
                border: 2px solid #E53E3E;
                border-radius: 6px;
                font-size: 14px;
                color: #2D3748;
                background: #FFF5F5;
                resize: vertical;
                min-height: 38px;
                max-height: ${window.innerHeight * 0.15}px;
            `;

            // Element selector toggle button
            const toggleButton = document.createElement('button');
            toggleButton.id = 'element-toggle-button';
            toggleButton.textContent = 'Enable Element Selector';
            toggleButton.style.width = '100%';
            toggleButton.style.padding = '8px 16px';
            toggleButton.style.backgroundColor = '#4299E1';
            toggleButton.style.color = 'white';
            toggleButton.style.border = 'none';
            toggleButton.style.borderRadius = '6px';
            toggleButton.style.cursor = 'pointer';
            toggleButton.style.marginTop = '15px';

            // Capture button
            const captureButton = document.createElement('button');
            captureButton.id = 'element-capture-button';
            captureButton.textContent = 'Capture';
            captureButton.style.width = '100%';
            captureButton.style.padding = '8px 16px';
            captureButton.style.backgroundColor = '#CBD5E0';
            captureButton.style.color = 'white';
            captureButton.style.border = 'none';
            captureButton.style.borderRadius = '6px';
            captureButton.style.cursor = 'not-allowed';
            captureButton.style.marginTop = '15px';
            captureButton.disabled = true;

            descriptionInput.addEventListener('input', () => this.checkInput());

            // Dragging functionality
            let isDragging = false;
            let currentX;
            let currentY;
            let initialX;
            let initialY;

            const dragStart = (e) => {
                if (e.target === header) {
                    isDragging = true;
                    initialX = e.clientX - inputBox.offsetLeft;
                    initialY = e.clientY - inputBox.offsetTop;
                }
            };

            const dragEnd = () => {
                isDragging = false;
            };

            const drag = (e) => {
                if (isDragging) {
                    e.preventDefault();
                    currentX = e.clientX - initialX;
                    currentY = e.clientY - initialY;

                    // Constrain to viewport
                    currentX = Math.min(Math.max(0, currentX), window.innerWidth - inputBox.offsetWidth);
                    currentY = Math.min(Math.max(0, currentY), window.innerHeight - inputBox.offsetHeight);

                    inputBox.style.left = `${currentX}px`;
                    inputBox.style.top = `${currentY}px`;
                }
            };

            header.addEventListener('mousedown', dragStart);
            document.addEventListener('mousemove', drag);
            document.addEventListener('mouseup', dragEnd);

            // Toggle button handler
            toggleButton.addEventListener('click', () => {
                if (this.selectedElement) {
                    this.unselectElement();
                    this.enable();
                } else if (this.isEnabled) {
                    this.disable();
                } else {
                    this.enable();
                }
            });

            // Capture handler
            captureButton.addEventListener('click', () => {
                const bbox = this.selectedElement.getBoundingClientRect();
                window.Captured.push({
                    Url: window.location.href,
                    ScrollCoordinates: {x: window.scrollX, y: window.scrollY},
                    ElementDescription: descriptionInput.value,
                    ElementCoordinates: {
                        x: bbox.left + bbox.width / 2, y: bbox.top + bbox.height / 2
                    }
                });
                
                // Reset the form but keep the box
                this.unselectElement();
                this.enable();
            });

            // Build and add the input box
            inputBox.appendChild(toggleButton);
            inputBox.appendChild(header);
            inputBox.appendChild(descriptionInput);
            inputBox.appendChild(captureButton);
            
            document.body.appendChild(inputBox);

            // Handle tab switch
            const handleTabChange = () => {
                if (document.hidden) {
                    this.destroyInputBox();
                }
            };
            document.addEventListener('visibilitychange', handleTabChange);
        },

        destroyInputBox: function() {
            const inputBox = document.getElementById('element-input-box');
            if (inputBox) {
                this.unselectElement();
                inputBox.remove();
                this.disable(); // Ensure element selector is disabled when destroying input box
            }
        }
    };
}