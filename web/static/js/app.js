// Image Processor Frontend JavaScript

document.addEventListener('DOMContentLoaded', function() {
    initUpload();
    initOperations();
});

// File Upload Handling
function initUpload() {
    const uploadSection = document.querySelector('.upload-section');
    const fileInput = document.getElementById('image-input');
    const previewContainer = document.getElementById('preview-container');

    if (!uploadSection) return;

    // Drag and drop
    uploadSection.addEventListener('dragover', function(e) {
        e.preventDefault();
        uploadSection.classList.add('dragover');
    });

    uploadSection.addEventListener('dragleave', function(e) {
        e.preventDefault();
        uploadSection.classList.remove('dragover');
    });

    uploadSection.addEventListener('drop', function(e) {
        e.preventDefault();
        uploadSection.classList.remove('dragover');

        const files = e.dataTransfer.files;
        if (files.length > 0) {
            fileInput.files = files;
            handleFileSelect(files[0]);
        }
    });

    // File input change
    if (fileInput) {
        fileInput.addEventListener('change', function() {
            if (this.files.length > 0) {
                handleFileSelect(this.files[0]);
            }
        });
    }
}

function handleFileSelect(file) {
    const previewContainer = document.getElementById('preview-container');
    const submitBtn = document.getElementById('submit-btn');

    if (!file.type.startsWith('image/')) {
        alert('Please select an image file');
        return;
    }

    // Show preview
    const reader = new FileReader();
    reader.onload = function(e) {
        previewContainer.innerHTML = `
            <div style="margin-top: 20px;">
                <img src="${e.target.result}" style="max-width: 300px; max-height: 200px; border: 1px solid #000;">
                <p style="margin-top: 10px; font-size: 0.9rem;">${file.name} (${formatBytes(file.size)})</p>
            </div>
        `;
    };
    reader.readAsDataURL(file);

    // Enable submit button
    if (submitBtn) {
        submitBtn.disabled = false;
    }
}

function formatBytes(bytes) {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

// Operations Management
let operations = [];

function initOperations() {
    updateOperationsInput();
}

function addOperation() {
    const select = document.getElementById('operation-select');
    const operation = select.value;

    if (!operation) return;

    const params = getOperationParams(operation);
    operations.push({
        operation: operation,
        parameters: params
    });

    renderOperations();
    updateOperationsInput();
}

function getOperationParams(operation) {
    const params = {};

    switch(operation) {
        case 'resize':
            const width = document.getElementById('param-width');
            const height = document.getElementById('param-height');
            if (width && width.value) params.width = parseInt(width.value);
            if (height && height.value) params.height = parseInt(height.value);
            break;
        case 'thumbnail':
            const size = document.getElementById('param-size');
            if (size && size.value) params.size = parseInt(size.value);
            break;
        case 'blur':
        case 'sharpen':
            const sigma = document.getElementById('param-sigma');
            if (sigma && sigma.value) params.sigma = parseFloat(sigma.value);
            break;
        case 'rotate':
            const angle = document.getElementById('param-angle');
            if (angle && angle.value) params.angle = parseFloat(angle.value);
            break;
        case 'flip':
            const horizontal = document.getElementById('param-horizontal');
            params.horizontal = horizontal ? horizontal.checked : true;
            break;
        case 'brightness':
        case 'contrast':
        case 'saturation':
            const amount = document.getElementById('param-amount');
            if (amount && amount.value) params.amount = parseFloat(amount.value);
            break;
    }

    return params;
}

function removeOperation(index) {
    operations.splice(index, 1);
    renderOperations();
    updateOperationsInput();
}

function renderOperations() {
    const container = document.getElementById('operations-list');
    if (!container) return;

    if (operations.length === 0) {
        container.innerHTML = '<p style="color: #666; font-size: 0.9rem;">No operations added yet</p>';
        return;
    }

    container.innerHTML = operations.map((op, index) => `
        <div class="operation-item">
            <span><strong>${op.operation}</strong> ${formatParams(op.parameters)}</span>
            <button type="button" class="remove-btn" onclick="removeOperation(${index})">×</button>
        </div>
    `).join('');
}

function formatParams(params) {
    const entries = Object.entries(params);
    if (entries.length === 0) return '';
    return '(' + entries.map(([k, v]) => `${k}: ${v}`).join(', ') + ')';
}

function updateOperationsInput() {
    const input = document.getElementById('operations-input');
    if (input) {
        input.value = JSON.stringify(operations);
    }
}

function showParams(operation) {
    // Hide all param groups
    document.querySelectorAll('.param-group').forEach(el => el.style.display = 'none');

    // Show relevant param group
    const paramGroup = document.getElementById(`params-${operation}`);
    if (paramGroup) {
        paramGroup.style.display = 'block';
    }
}

// Form submission
function submitJob(event) {
    event.preventDefault();

    const form = event.target;
    const formData = new FormData(form);

    // Add operations
    formData.set('operations', JSON.stringify(operations));

    const submitBtn = document.getElementById('submit-btn');
    submitBtn.disabled = true;
    submitBtn.innerHTML = '<span class="spinner"></span> Processing...';

    fetch('/api/v1/jobs', {
        method: 'POST',
        body: formData
    })
    .then(response => response.json())
    .then(data => {
        if (data.error) {
            alert('Error: ' + data.error);
        } else {
            window.location.href = '/jobs/' + data.id;
        }
    })
    .catch(error => {
        alert('Error: ' + error.message);
    })
    .finally(() => {
        submitBtn.disabled = false;
        submitBtn.innerHTML = 'Process Image';
    });
}

// Streaming job status with Server-Sent Events
function streamJobStatus(jobId) {
    const statusEl = document.getElementById('job-status');
    const progressEl = document.getElementById('job-progress');
    const progressText = document.getElementById('progress-text');
    const progressContainer = document.getElementById('progress-container');

    // Create EventSource for SSE
    const eventSource = new EventSource(`/api/v1/jobs/${jobId}/stream`);

    eventSource.onmessage = function(event) {
        try {
            const job = JSON.parse(event.data);

            // Update status
            if (statusEl) {
                statusEl.textContent = job.status.toUpperCase();
                statusEl.className = 'job-status ' + job.status;
            }

            // Update progress bar
            if (progressEl && job.progress !== undefined) {
                progressEl.style.width = job.progress + '%';
                
                if (progressText) {
                    progressText.textContent = job.progress + '%';
                }
            }

            // Show/hide progress container based on status
            if (progressContainer) {
                if (job.status === 'processing' || job.status === 'queued') {
                    progressContainer.style.display = 'block';
                } else {
                    progressContainer.style.display = 'none';
                }
            }

            // Reload page on completion for final state
            if (job.status === 'completed' || job.status === 'failed' || job.status === 'cancelled') {
                eventSource.close();
                // Wait a moment before reload to show final status
                setTimeout(() => {
                    location.reload();
                }, 1000);
            }
        } catch (error) {
            console.error('Error parsing SSE data:', error);
        }
    };

    eventSource.onerror = function(error) {
        console.error('SSE error:', error);
        eventSource.close();
        // Fallback to polling if SSE fails
        console.log('Falling back to polling...');
        pollJobStatus(jobId);
    };

    // Cleanup on page unload
    window.addEventListener('beforeunload', function() {
        eventSource.close();
    });
}

// Polling for job status (fallback)
function pollJobStatus(jobId) {
    const statusEl = document.getElementById('job-status');
    const progressEl = document.getElementById('job-progress');
    const progressText = document.getElementById('progress-text');

    const poll = setInterval(() => {
        fetch(`/api/v1/jobs/${jobId}`)
            .then(response => response.json())
            .then(job => {
                if (statusEl) {
                    statusEl.textContent = job.status.toUpperCase();
                    statusEl.className = 'job-status ' + job.status;
                }

                if (progressEl && job.progress !== undefined) {
                    progressEl.style.width = job.progress + '%';
                    
                    if (progressText) {
                        progressText.textContent = job.progress + '%';
                    }
                }

                if (job.status === 'completed' || job.status === 'failed' || job.status === 'cancelled') {
                    clearInterval(poll);
                    location.reload();
                }
            })
            .catch(error => {
                console.error('Poll error:', error);
            });
    }, 2000);
}
