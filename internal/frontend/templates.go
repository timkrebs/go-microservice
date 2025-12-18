package frontend

import (
	"fmt"
	"html/template"
	"io"
	"time"

	"github.com/timkrebs/image-processor/internal/models"
)

// Templates holds all HTML templates
type Templates struct {
	templates *template.Template
}

// NewTemplates creates and parses all templates
func NewTemplates() *Templates {
	funcMap := template.FuncMap{
		"formatTime": func(t time.Time) string {
			return t.Format("Jan 02, 2006 15:04:05")
		},
		"formatTimePtr": func(t *time.Time) string {
			if t == nil {
				return "-"
			}
			return t.Format("Jan 02, 2006 15:04:05")
		},
		"formatBytes": func(bytes int64) string {
			const unit = 1024
			if bytes < unit {
				return fmt.Sprintf("%d B", bytes)
			}
			div, exp := int64(unit), 0
			for n := bytes / unit; n >= unit; n /= unit {
				div *= unit
				exp++
			}
			return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
		},
		"formatDuration": func(ms *int64) string {
			if ms == nil {
				return "-"
			}
			d := time.Duration(*ms) * time.Millisecond
			if d < time.Second {
				return fmt.Sprintf("%dms", *ms)
			}
			return fmt.Sprintf("%.2fs", d.Seconds())
		},
		"shortID": func(id string) string {
			if len(id) > 8 {
				return id[:8]
			}
			return id
		},
		"add": func(a, b int) int {
			return a + b
		},
		"subtract": func(a, b int) int {
			return a - b
		},
	}

	tmpl := template.Must(template.New("").Funcs(funcMap).Parse(homeTemplate))
	template.Must(tmpl.Parse(uploadTemplate))
	template.Must(tmpl.Parse(jobsTemplate))
	template.Must(tmpl.Parse(jobDetailTemplate))

	return &Templates{templates: tmpl}
}

// Render renders a template with the given data
func (t *Templates) Render(w io.Writer, name string, data interface{}) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

// PageData holds common page data
type PageData struct {
	Title   string
	Active  string
	Content interface{}
}

// HomeData holds home page data
type HomeData struct {
	Stats *models.QueueStats
}

// JobsData holds jobs list page data
type JobsData struct {
	Jobs       []*models.Job
	Page       int
	TotalPages int
	Total      int
}

// JobDetailData holds job detail page data
type JobDetailData struct {
	Job *models.Job
}

const homeTemplate = `
{{define "home"}}
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}} - Image Processor</title>
    <link rel="stylesheet" href="/static/css/style.css">
    <script src="/static/js/app.js" defer></script>
</head>
<body>
    <header>
        <div class="container">
            <h1>Image Processor</h1>
            <nav>
                <a href="/" {{if eq .Active "home"}}class="active"{{end}}>Home</a>
                <a href="/upload" {{if eq .Active "upload"}}class="active"{{end}}>Upload</a>
                <a href="/jobs" {{if eq .Active "jobs"}}class="active"{{end}}>Jobs</a>
            </nav>
        </div>
    </header>
    <main>
        <div class="container">
            <h2>Dashboard</h2>
            {{with .Content}}
            <div class="stats">
                <div class="stat-card">
                    <div class="stat-value">{{if .Stats}}{{.Stats.StreamLength}}{{else}}0{{end}}</div>
                    <div class="stat-label">Queue Length</div>
                </div>
                <div class="stat-card">
                    <div class="stat-value">{{if .Stats}}{{.Stats.PendingMessages}}{{else}}0{{end}}</div>
                    <div class="stat-label">Pending</div>
                </div>
                <div class="stat-card">
                    <div class="stat-value">{{if .Stats}}{{.Stats.ConsumerCount}}{{else}}0{{end}}</div>
                    <div class="stat-label">Workers</div>
                </div>
            </div>
            {{end}}
            <div style="text-align: center; margin-top: 40px;">
                <a href="/upload" class="btn">Upload New Image</a>
                <a href="/jobs" class="btn btn-outline" style="margin-left: 10px;">View All Jobs</a>
            </div>
        </div>
    </main>
    <footer>
        <div class="container">
            <p>Image Processor Microservices Platform</p>
        </div>
    </footer>
</body>
</html>
{{end}}
`

const uploadTemplate = `
{{define "upload"}}
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}} - Image Processor</title>
    <link rel="stylesheet" href="/static/css/style.css">
    <script src="/static/js/app.js" defer></script>
</head>
<body>
    <header>
        <div class="container">
            <h1>Image Processor</h1>
            <nav>
                <a href="/" {{if eq .Active "home"}}class="active"{{end}}>Home</a>
                <a href="/upload" {{if eq .Active "upload"}}class="active"{{end}}>Upload</a>
                <a href="/jobs" {{if eq .Active "jobs"}}class="active"{{end}}>Jobs</a>
            </nav>
        </div>
    </header>
    <main>
        <div class="container">
            <h2>Upload Image</h2>
            <form id="upload-form" onsubmit="submitJob(event)" enctype="multipart/form-data">
                <div class="upload-section">
                    <label for="image-input">
                        <div class="upload-icon">+</div>
                        <p><strong>Click to upload</strong> or drag and drop</p>
                        <p>PNG, JPG, GIF up to 50MB</p>
                    </label>
                    <input type="file" id="image-input" name="image" accept="image/*" required>
                    <div id="preview-container"></div>
                </div>

                <h2>Processing Operations</h2>

                <div class="form-group">
                    <label for="operation-select">Add Operation</label>
                    <select id="operation-select" onchange="showParams(this.value)">
                        <option value="">Select an operation...</option>
                        <option value="resize">Resize</option>
                        <option value="thumbnail">Thumbnail</option>
                        <option value="blur">Blur</option>
                        <option value="sharpen">Sharpen</option>
                        <option value="grayscale">Grayscale</option>
                        <option value="sepia">Sepia</option>
                        <option value="rotate">Rotate</option>
                        <option value="flip">Flip</option>
                        <option value="brightness">Brightness</option>
                        <option value="contrast">Contrast</option>
                        <option value="saturation">Saturation</option>
                    </select>
                </div>

                <div id="params-resize" class="param-group" style="display:none;">
                    <div class="form-group">
                        <label>Width (px)</label>
                        <input type="number" id="param-width" placeholder="800">
                    </div>
                    <div class="form-group">
                        <label>Height (px)</label>
                        <input type="number" id="param-height" placeholder="600">
                    </div>
                </div>

                <div id="params-thumbnail" class="param-group" style="display:none;">
                    <div class="form-group">
                        <label>Size (px)</label>
                        <input type="number" id="param-size" value="150">
                    </div>
                </div>

                <div id="params-blur" class="param-group" style="display:none;">
                    <div class="form-group">
                        <label>Sigma</label>
                        <input type="number" id="param-sigma" value="3" step="0.5">
                    </div>
                </div>

                <div id="params-sharpen" class="param-group" style="display:none;">
                    <div class="form-group">
                        <label>Sigma</label>
                        <input type="number" id="param-sigma" value="1" step="0.5">
                    </div>
                </div>

                <div id="params-rotate" class="param-group" style="display:none;">
                    <div class="form-group">
                        <label>Angle (degrees)</label>
                        <input type="number" id="param-angle" value="90">
                    </div>
                </div>

                <div id="params-flip" class="param-group" style="display:none;">
                    <div class="form-group">
                        <label><input type="checkbox" id="param-horizontal" checked> Horizontal</label>
                    </div>
                </div>

                <div id="params-brightness" class="param-group" style="display:none;">
                    <div class="form-group">
                        <label>Amount (-100 to 100)</label>
                        <input type="number" id="param-amount" value="20" min="-100" max="100">
                    </div>
                </div>

                <div id="params-contrast" class="param-group" style="display:none;">
                    <div class="form-group">
                        <label>Amount (-100 to 100)</label>
                        <input type="number" id="param-amount" value="20" min="-100" max="100">
                    </div>
                </div>

                <div id="params-saturation" class="param-group" style="display:none;">
                    <div class="form-group">
                        <label>Amount (-100 to 100)</label>
                        <input type="number" id="param-amount" value="20" min="-100" max="100">
                    </div>
                </div>

                <button type="button" class="btn btn-outline btn-sm" onclick="addOperation()">+ Add Operation</button>

                <div id="operations-list" class="operations-list" style="margin-top: 20px;">
                    <p style="color: #666; font-size: 0.9rem;">No operations added yet</p>
                </div>

                <input type="hidden" id="operations-input" name="operations" value="[]">

                <div style="margin-top: 30px;">
                    <button type="submit" id="submit-btn" class="btn" disabled>Process Image</button>
                </div>
            </form>
        </div>
    </main>
    <footer>
        <div class="container">
            <p>Image Processor Microservices Platform</p>
        </div>
    </footer>
</body>
</html>
{{end}}
`

const jobsTemplate = `
{{define "jobs"}}
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}} - Image Processor</title>
    <link rel="stylesheet" href="/static/css/style.css">
    <script src="/static/js/app.js" defer></script>
</head>
<body>
    <header>
        <div class="container">
            <h1>Image Processor</h1>
            <nav>
                <a href="/" {{if eq .Active "home"}}class="active"{{end}}>Home</a>
                <a href="/upload" {{if eq .Active "upload"}}class="active"{{end}}>Upload</a>
                <a href="/jobs" {{if eq .Active "jobs"}}class="active"{{end}}>Jobs</a>
            </nav>
        </div>
    </header>
    <main>
        <div class="container">
            <h2>Processing Jobs</h2>
            {{with .Content}}
            {{if .Jobs}}
            <div class="jobs-grid">
                {{range .Jobs}}
                <div class="job-card">
                    <div class="job-header">
                        <span class="job-id">{{shortID .ID.String}}</span>
                        <span class="job-status {{.Status}}">{{.Status}}</span>
                    </div>
                    <div class="job-name">{{.OriginalName}}</div>
                    <div class="job-meta">
                        <div>Size: {{formatBytes .FileSize}}</div>
                        <div>Created: {{formatTime .CreatedAt}}</div>
                        {{if .ProcessingTime}}
                        <div>Processing: {{formatDuration .ProcessingTime}}</div>
                        {{end}}
                    </div>
                    {{if or (eq .Status "processing") (eq .Status "queued")}}
                    <div class="progress-container">
                        <div class="progress-bar">
                            <div class="progress job-progress-{{.ID}}" style="width: {{.Progress}}%"></div>
                        </div>
                        <div class="progress-text job-progress-text-{{.ID}}">{{.Progress}}%</div>
                    </div>
                    <script>
                        (function() {
                            const eventSource = new EventSource('/api/v1/jobs/{{.ID}}/stream');
                            eventSource.onmessage = function(e) {
                                try {
                                    const job = JSON.parse(e.data);
                                    const progressEl = document.querySelector('.job-progress-{{.ID}}');
                                    const textEl = document.querySelector('.job-progress-text-{{.ID}}');
                                    if (progressEl && job.progress !== undefined) {
                                        progressEl.style.width = job.progress + '%';
                                        if (textEl) textEl.textContent = job.progress + '%';
                                    }
                                    if (job.status === 'completed' || job.status === 'failed' || job.status === 'cancelled') {
                                        eventSource.close();
                                        setTimeout(() => location.reload(), 1000);
                                    }
                                } catch(err) { console.error(err); }
                            };
                            eventSource.onerror = function() { eventSource.close(); };
                        })();
                    </script>
                    {{end}}
                    <div class="job-actions">
                        <a href="/jobs/{{.ID}}" class="btn btn-sm">View Details</a>
                    </div>
                </div>
                {{end}}
            </div>

            {{if gt .TotalPages 1}}
            <div style="margin-top: 30px; text-align: center;">
                {{if gt .Page 1}}
                <a href="/jobs?page={{subtract .Page 1}}" class="btn btn-outline btn-sm">← Previous</a>
                {{end}}
                <span style="margin: 0 20px;">Page {{.Page}} of {{.TotalPages}}</span>
                {{if lt .Page .TotalPages}}
                <a href="/jobs?page={{add .Page 1}}" class="btn btn-outline btn-sm">Next →</a>
                {{end}}
            </div>
            {{end}}

            {{else}}
            <div class="empty-state">
                <div class="icon">📷</div>
                <p>No jobs yet</p>
                <a href="/upload" class="btn" style="margin-top: 20px;">Upload Your First Image</a>
            </div>
            {{end}}
            {{end}}
        </div>
    </main>
    <footer>
        <div class="container">
            <p>Image Processor Microservices Platform</p>
        </div>
    </footer>
</body>
</html>
{{end}}
`

const jobDetailTemplate = `
{{define "job-detail"}}
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}} - Image Processor</title>
    <link rel="stylesheet" href="/static/css/style.css">
    <script src="/static/js/app.js" defer></script>
</head>
<body>
    <header>
        <div class="container">
            <h1>Image Processor</h1>
            <nav>
                <a href="/" {{if eq .Active "home"}}class="active"{{end}}>Home</a>
                <a href="/upload" {{if eq .Active "upload"}}class="active"{{end}}>Upload</a>
                <a href="/jobs" {{if eq .Active "jobs"}}class="active"{{end}}>Jobs</a>
            </nav>
        </div>
    </header>
    <main>
        <div class="container">
            <h2>Job Details</h2>
            {{with .Content}}
            {{with .Job}}
            <div class="job-card" style="max-width: 600px;">
                <div class="job-header">
                    <span class="job-id">{{.ID}}</span>
                    <span id="job-status" class="job-status {{.Status}}">{{.Status}}</span>
                </div>

                <div class="job-name">{{.OriginalName}}</div>

                {{if or (eq .Status "processing") (eq .Status "queued")}}
                <div id="progress-container" class="progress-container">
                    <div class="progress-bar">
                        <div id="job-progress" class="progress" style="width: {{.Progress}}%"></div>
                    </div>
                    <div id="progress-text" class="progress-text">{{.Progress}}%</div>
                </div>
                <script>streamJobStatus('{{.ID}}');</script>
                {{end}}

                {{if eq .Status "completed"}}
                <div class="job-preview">
                    <img src="/api/v1/images/{{.ID}}" alt="Processed image">
                </div>
                {{end}}

                <div class="job-meta" style="margin-top: 20px;">
                    <p><strong>File Size:</strong> {{formatBytes .FileSize}}</p>
                    <p><strong>Content Type:</strong> {{.ContentType}}</p>
                    <p><strong>Created:</strong> {{formatTime .CreatedAt}}</p>
                    <p><strong>Started:</strong> {{formatTimePtr .StartedAt}}</p>
                    <p><strong>Completed:</strong> {{formatTimePtr .CompletedAt}}</p>
                    {{if .ProcessingTime}}
                    <p><strong>Processing Time:</strong> {{formatDuration .ProcessingTime}}</p>
                    {{end}}
                    {{if .WorkerID}}
                    <p><strong>Worker:</strong> {{.WorkerID}}</p>
                    {{end}}
                    {{if .Error}}
                    <p><strong>Error:</strong> <span style="color: #666;">{{.Error}}</span></p>
                    {{end}}
                </div>

                <div style="margin-top: 20px;">
                    <h3 style="font-size: 1rem; margin-bottom: 10px;">Operations</h3>
                    {{range .Operations}}
                    <div class="operation-item">
                        <span>{{.Operation}}</span>
                    </div>
                    {{else}}
                    <p style="color: #666;">No operations</p>
                    {{end}}
                </div>

                <div class="job-actions" style="margin-top: 20px;">
                    {{if eq .Status "completed"}}
                    <a href="/api/v1/images/{{.ID}}" class="btn btn-sm" download>Download Processed</a>
                    <a href="/api/v1/images/{{.ID}}?original=true" class="btn btn-sm btn-outline" download>Download Original</a>
                    {{end}}
                    <a href="/jobs" class="btn btn-sm btn-outline">Back to Jobs</a>
                </div>
            </div>
            {{end}}
            {{end}}
        </div>
    </main>
    <footer>
        <div class="container">
            <p>Image Processor Microservices Platform</p>
        </div>
    </footer>
</body>
</html>
{{end}}
`
