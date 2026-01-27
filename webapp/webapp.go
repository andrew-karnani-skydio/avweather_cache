package webapp

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/andrew/avweather_cache/cache"
	"github.com/andrew/avweather_cache/models"
)

// Handler handles web UI requests
type Handler struct {
	cache *cache.Cache
}

// New creates a new webapp handler
func New(c *cache.Cache) *Handler {
	return &Handler{cache: c}
}

// SearchHandler returns JSON data for AJAX search requests
func (h *Handler) SearchHandler(w http.ResponseWriter, r *http.Request) {
	metars := h.cache.GetAll()

	// Get search query
	searchQuery := r.URL.Query().Get("search")
	var filteredMetars []models.METAR

	// Filter by search if provided
	if searchQuery != "" {
		searchQuery = strings.ToUpper(strings.TrimSpace(searchQuery))
		for _, m := range metars {
			if strings.Contains(strings.ToUpper(m.StationID), searchQuery) {
				filteredMetars = append(filteredMetars, m)
			}
		}
	} else {
		filteredMetars = metars
	}

	// Sort by observation time (newest first)
	sort.Slice(filteredMetars, func(i, j int) bool {
		return filteredMetars[i].ObservationTime.After(filteredMetars[j].ObservationTime)
	})

	// Pagination
	pageSize := 100
	page := 1
	if pageParam := r.URL.Query().Get("page"); pageParam != "" {
		if p, err := strconv.Atoi(pageParam); err == nil && p > 0 {
			page = p
		}
	}

	totalPages := (len(filteredMetars) + pageSize - 1) / pageSize
	if totalPages == 0 {
		totalPages = 1
	}
	startIdx := (page - 1) * pageSize
	endIdx := startIdx + pageSize

	if startIdx >= len(filteredMetars) {
		startIdx = 0
		page = 1
	}
	if endIdx > len(filteredMetars) {
		endIdx = len(filteredMetars)
	}

	var displayMetars []models.METAR
	if len(filteredMetars) > 0 {
		displayMetars = filteredMetars[startIdx:endIdx]
	}

	response := struct {
		Metars      []models.METAR `json:"metars"`
		MetarCount  int            `json:"metar_count"`
		TotalCount  int            `json:"total_count"`
		Page        int            `json:"page"`
		TotalPages  int            `json:"total_pages"`
		StartIdx    int            `json:"start_idx"`
		EndIdx      int            `json:"end_idx"`
		SearchQuery string         `json:"search_query"`
	}{
		Metars:      displayMetars,
		MetarCount:  len(filteredMetars),
		TotalCount:  len(metars),
		Page:        page,
		TotalPages:  totalPages,
		StartIdx:    startIdx + 1,
		EndIdx:      endIdx,
		SearchQuery: searchQuery,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// IndexHandler serves the main dashboard
func (h *Handler) IndexHandler(w http.ResponseWriter, r *http.Request) {
	status := h.cache.Status()
	metars := h.cache.GetAll()

	// Get search query
	searchQuery := r.URL.Query().Get("search")
	var filteredMetars []models.METAR

	// Filter by search if provided
	if searchQuery != "" {
		searchQuery = strings.ToUpper(strings.TrimSpace(searchQuery))
		for _, m := range metars {
			if strings.Contains(strings.ToUpper(m.StationID), searchQuery) {
				filteredMetars = append(filteredMetars, m)
			}
		}
	} else {
		filteredMetars = metars
	}

	// Sort by observation time (newest first)
	sort.Slice(filteredMetars, func(i, j int) bool {
		return filteredMetars[i].ObservationTime.After(filteredMetars[j].ObservationTime)
	})

	// Pagination
	pageSize := 100
	page := 1
	if pageParam := r.URL.Query().Get("page"); pageParam != "" {
		if p, err := strconv.Atoi(pageParam); err == nil && p > 0 {
			page = p
		}
	}

	totalPages := (len(filteredMetars) + pageSize - 1) / pageSize
	if totalPages == 0 {
		totalPages = 1
	}
	startIdx := (page - 1) * pageSize
	endIdx := startIdx + pageSize

	if startIdx >= len(filteredMetars) {
		startIdx = 0
		page = 1
	}
	if endIdx > len(filteredMetars) {
		endIdx = len(filteredMetars)
	}

	var displayMetars []models.METAR
	if len(filteredMetars) > 0 {
		displayMetars = filteredMetars[startIdx:endIdx]
	}

	data := struct {
		Status      cache.Status
		Metars      interface{}
		MetarCount  int
		TotalCount  int
		Page        int
		TotalPages  int
		PageSize    int
		StartIdx    int
		EndIdx      int
		SearchQuery string
	}{
		Status:      status,
		Metars:      displayMetars,
		MetarCount:  len(filteredMetars),
		TotalCount:  len(metars),
		Page:        page,
		TotalPages:  totalPages,
		PageSize:    pageSize,
		StartIdx:    startIdx + 1,
		EndIdx:      endIdx,
		SearchQuery: searchQuery,
	}

	tmpl := template.Must(template.New("index").Funcs(template.FuncMap{
		"formatTime": func(t time.Time) string {
			if t.IsZero() {
				return "Never"
			}
			return t.Format("2006-01-02 15:04:05 MST")
		},
		"formatAge": func(t time.Time) string {
			if t.IsZero() {
				return "N/A"
			}
			age := time.Since(t)
			if age < time.Minute {
				return fmt.Sprintf("%.0fs ago", age.Seconds())
			} else if age < time.Hour {
				return fmt.Sprintf("%.0fm ago", age.Minutes())
			} else if age < 24*time.Hour {
				return fmt.Sprintf("%.1fh ago", age.Hours())
			}
			return fmt.Sprintf("%.1fd ago", age.Hours()/24)
		},
		"formatTemp": func(temp *float64) string {
			if temp == nil {
				return "N/A"
			}
			return fmt.Sprintf("%.1f°C", *temp)
		},
		"formatWind": func(dir string, speed *int, gust *int) string {
			if dir == "" || speed == nil {
				return "N/A"
			}
			result := fmt.Sprintf("%s° @ %dkt", dir, *speed)
			if gust != nil && *gust > 0 {
				result += fmt.Sprintf(" G%dkt", *gust)
			}
			return result
		},
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
		},
	}).Parse(htmlTemplate))

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.Execute(w, data)
}

const htmlTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Aviation Weather Cache</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
            margin: 0;
            padding: 20px;
            background: #f5f5f5;
        }
        .container {
            max-width: 1400px;
            margin: 0 auto;
            background: white;
            padding: 20px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        h1 {
            color: #333;
            border-bottom: 3px solid #007bff;
            padding-bottom: 10px;
        }
        h2 {
            color: #555;
            margin-top: 30px;
        }
        .status {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 15px;
            margin-bottom: 30px;
        }
        .status-card {
            background: #f8f9fa;
            padding: 15px;
            border-radius: 6px;
            border-left: 4px solid #007bff;
        }
        .status-card.error {
            border-left-color: #dc3545;
            background: #fff5f5;
        }
        .status-card.success {
            border-left-color: #28a745;
        }
        .status-label {
            font-size: 0.85em;
            color: #666;
            text-transform: uppercase;
            letter-spacing: 0.5px;
        }
        .status-value {
            font-size: 1.3em;
            font-weight: bold;
            color: #333;
            margin-top: 5px;
        }
        table {
            width: 100%;
            border-collapse: collapse;
            margin-top: 20px;
        }
        th {
            background: #007bff;
            color: white;
            padding: 12px;
            text-align: left;
            font-weight: 600;
            position: sticky;
            top: 0;
        }
        td {
            padding: 10px 12px;
            border-bottom: 1px solid #dee2e6;
        }
        tr:hover {
            background: #f8f9fa;
        }
        .flight-cat {
            display: inline-block;
            padding: 3px 8px;
            border-radius: 3px;
            font-size: 0.85em;
            font-weight: bold;
        }
        .flight-cat.VFR { background: #28a745; color: white; }
        .flight-cat.MVFR { background: #007bff; color: white; }
        .flight-cat.IFR { background: #dc3545; color: white; }
        .flight-cat.LIFR { background: #d946ef; color: white; }
        .error-msg {
            color: #dc3545;
            font-weight: bold;
        }
        .info {
            background: #e7f3ff;
            padding: 10px;
            border-radius: 4px;
            margin-bottom: 20px;
            border-left: 4px solid #007bff;
        }
        code {
            background: #f4f4f4;
            padding: 2px 6px;
            border-radius: 3px;
            font-family: 'Courier New', monospace;
        }
        .search-box {
            margin: 20px 0;
            padding: 15px;
            background: #f8f9fa;
            border-radius: 6px;
            border-left: 4px solid #007bff;
        }
        .search-box form {
            margin: 0;
        }
        .search-box input[type="text"] {
            width: 100%;
            padding: 10px;
            font-size: 16px;
            border: 2px solid #dee2e6;
            border-radius: 4px;
            box-sizing: border-box;
        }
        .search-box input[type="text"]:focus {
            outline: none;
            border-color: #007bff;
        }
        .pagination {
            display: flex;
            justify-content: center;
            align-items: center;
            gap: 10px;
            margin: 20px 0;
            padding: 15px;
            background: #f8f9fa;
            border-radius: 6px;
        }
        .pagination a, .pagination span {
            padding: 8px 12px;
            background: white;
            border: 1px solid #dee2e6;
            border-radius: 4px;
            text-decoration: none;
            color: #007bff;
            font-weight: 500;
        }
        .pagination a:hover {
            background: #007bff;
            color: white;
        }
        .pagination .current {
            background: #007bff;
            color: white;
            border-color: #007bff;
        }
        .pagination .disabled {
            color: #6c757d;
            pointer-events: none;
            opacity: 0.5;
        }
        .results-info {
            text-align: center;
            color: #666;
            margin: 10px 0;
            font-size: 0.9em;
        }
        .hidden {
            display: none;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>Aviation Weather Cache</h1>

        <div class="info">
            <strong>API Endpoint:</strong> <code>/api/metar</code> |
            <strong>Metrics:</strong> <code>/metrics</code> |
            <strong>Auto-refresh:</strong> Every 30 seconds (no page reload)
        </div>

        <h2>System Status</h2>
        <div class="status">
            <div class="status-card">
                <div class="status-label">Total Stations</div>
                <div class="status-value">{{.Status.TotalStations}}</div>
            </div>
            <div class="status-card {{if .Status.LastPullError}}error{{else}}success{{end}}">
                <div class="status-label">Last Successful Pull</div>
                <div class="status-value">{{formatAge .Status.LastSuccessfulPull}}</div>
            </div>
            <div class="status-card">
                <div class="status-label">Last Pull Attempt</div>
                <div class="status-value">{{formatAge .Status.LastPullAttempt}}</div>
            </div>
            {{if .Status.LastPullError}}
            <div class="status-card error">
                <div class="status-label">Last Error</div>
                <div class="status-value error-msg">{{.Status.LastPullError.Error}}</div>
            </div>
            {{end}}
        </div>

        <h2>Recent METARs</h2>

        <div class="search-box">
            <form method="GET" action="/" onsubmit="return false;">
                <input type="text" name="search" id="searchInput" value="{{.SearchQuery}}" placeholder="Search by station ID (e.g., KJFK, KLAX)..." autocomplete="off">
            </form>
            {{if .SearchQuery}}
            <div style="margin-top: 10px; color: #666;">
                Showing {{.MetarCount}} of {{.TotalCount}} stations matching "{{.SearchQuery}}"
                <a href="#" onclick="clearSearch(); return false;" style="margin-left: 10px; color: #007bff; text-decoration: none;">Clear Search</a>
            </div>
            {{end}}
        </div>

        <div class="results-info">
            {{if .SearchQuery}}
                Showing results {{.StartIdx}}-{{.EndIdx}} of {{.MetarCount}} (Page {{.Page}} of {{.TotalPages}})
            {{else}}
                Showing {{.StartIdx}}-{{.EndIdx}} of {{.MetarCount}} stations (Page {{.Page}} of {{.TotalPages}})
            {{end}}
        </div>

        <div class="pagination">
            {{if gt .Page 1}}
                <a href="#" onclick="loadPage(1); return false;">« First</a>
                <a href="#" onclick="loadPage({{sub .Page 1}}); return false;">‹ Prev</a>
            {{else}}
                <span class="disabled">« First</span>
                <span class="disabled">‹ Prev</span>
            {{end}}

            <span class="current">Page {{.Page}} of {{.TotalPages}}</span>

            {{if lt .Page .TotalPages}}
                <a href="#" onclick="loadPage({{add .Page 1}}); return false;">Next ›</a>
                <a href="#" onclick="loadPage({{.TotalPages}}); return false;">Last »</a>
            {{else}}
                <span class="disabled">Next ›</span>
                <span class="disabled">Last »</span>
            {{end}}
        </div>

        <table id="metarTable">
            <thead>
                <tr>
                    <th>Station</th>
                    <th>Observation Time</th>
                    <th>Age</th>
                    <th>Flight Cat</th>
                    <th>Temp</th>
                    <th>Wind</th>
                    <th>Visibility</th>
                    <th>Raw Text</th>
                </tr>
            </thead>
            <tbody>
                {{range .Metars}}
                <tr>
                    <td><strong>{{.StationID}}</strong></td>
                    <td>{{formatTime .ObservationTime}}</td>
                    <td>{{formatAge .ObservationTime}}</td>
                    <td><span class="flight-cat {{.FlightCategory}}">{{.FlightCategory}}</span></td>
                    <td>{{formatTemp .TempC}}</td>
                    <td>{{formatWind .WindDirDegrees .WindSpeedKt .WindGustKt}}</td>
                    <td>{{.VisibilityMi}}</td>
                    <td style="font-size: 0.85em;">{{.RawText}}</td>
                </tr>
                {{end}}
            </tbody>
        </table>

        <div class="pagination">
            {{if gt .Page 1}}
                <a href="#" onclick="loadPage(1); return false;">« First</a>
                <a href="#" onclick="loadPage({{sub .Page 1}}); return false;">‹ Prev</a>
            {{else}}
                <span class="disabled">« First</span>
                <span class="disabled">‹ Prev</span>
            {{end}}

            <span class="current">Page {{.Page}} of {{.TotalPages}}</span>

            {{if lt .Page .TotalPages}}
                <a href="#" onclick="loadPage({{add .Page 1}}); return false;">Next ›</a>
                <a href="#" onclick="loadPage({{.TotalPages}}); return false;">Last »</a>
            {{else}}
                <span class="disabled">Next ›</span>
                <span class="disabled">Last »</span>
            {{end}}
        </div>
    </div>

    <script>
        let searchTimeout;
        const searchInput = document.getElementById('searchInput');
        let currentPage = 1;
        let currentSearch = '';

        function formatTime(timeStr) {
            if (!timeStr) return 'Never';
            const date = new Date(timeStr);
            return date.toLocaleString('en-US', {
                year: 'numeric', month: '2-digit', day: '2-digit',
                hour: '2-digit', minute: '2-digit', second: '2-digit',
                timeZoneName: 'short'
            });
        }

        function formatAge(timeStr) {
            if (!timeStr) return 'N/A';
            const now = new Date();
            const time = new Date(timeStr);
            const seconds = (now - time) / 1000;

            if (seconds < 60) return Math.floor(seconds) + 's ago';
            if (seconds < 3600) return Math.floor(seconds / 60) + 'm ago';
            if (seconds < 86400) return (seconds / 3600).toFixed(1) + 'h ago';
            return (seconds / 86400).toFixed(1) + 'd ago';
        }

        function formatTemp(temp) {
            return temp != null ? temp.toFixed(1) + '°C' : 'N/A';
        }

        function formatWind(dir, speed, gust) {
            if (!dir || speed == null) return 'N/A';
            let result = dir + '° @ ' + speed + 'kt';
            if (gust != null && gust > 0) result += ' G' + gust + 'kt';
            return result;
        }

        function updateTable(data) {
            const tbody = document.querySelector('#metarTable tbody');
            tbody.innerHTML = '';

            data.metars.forEach(metar => {
                const row = document.createElement('tr');
                row.innerHTML =
                    '<td><strong>' + (metar.station_id || '') + '</strong></td>' +
                    '<td>' + formatTime(metar.observation_time) + '</td>' +
                    '<td>' + formatAge(metar.observation_time) + '</td>' +
                    '<td><span class="flight-cat ' + (metar.flight_category || '') + '">' + (metar.flight_category || '') + '</span></td>' +
                    '<td>' + formatTemp(metar.temp_c) + '</td>' +
                    '<td>' + formatWind(metar.wind_dir_degrees, metar.wind_speed_kt, metar.wind_gust_kt) + '</td>' +
                    '<td>' + (metar.visibility_statute_mi || '') + '</td>' +
                    '<td style="font-size: 0.85em;">' + (metar.raw_text || '') + '</td>';
                tbody.appendChild(row);
            });

            updatePagination(data);
            updateResultsInfo(data);
        }

        function updatePagination(data) {
            const topPagination = document.querySelector('.pagination');
            const bottomPagination = document.querySelectorAll('.pagination')[1];

            const paginationHTML = generatePaginationHTML(data);
            topPagination.innerHTML = paginationHTML;
            bottomPagination.innerHTML = paginationHTML;
        }

        function generatePaginationHTML(data) {
            const searchParam = data.search_query ? '&search=' + encodeURIComponent(data.search_query) : '';
            let html = '';

            if (data.page > 1) {
                html += '<a href="#" onclick="loadPage(1); return false;">« First</a>';
                html += '<a href="#" onclick="loadPage(' + (data.page - 1) + '); return false;">‹ Prev</a>';
            } else {
                html += '<span class="disabled">« First</span>';
                html += '<span class="disabled">‹ Prev</span>';
            }

            html += '<span class="current">Page ' + data.page + ' of ' + data.total_pages + '</span>';

            if (data.page < data.total_pages) {
                html += '<a href="#" onclick="loadPage(' + (data.page + 1) + '); return false;">Next ›</a>';
                html += '<a href="#" onclick="loadPage(' + data.total_pages + '); return false;">Last »</a>';
            } else {
                html += '<span class="disabled">Next ›</span>';
                html += '<span class="disabled">Last »</span>';
            }

            return html;
        }

        function updateResultsInfo(data) {
            const resultsInfo = document.querySelector('.results-info');
            if (data.search_query) {
                resultsInfo.textContent = 'Showing results ' + data.start_idx + '-' + data.end_idx + ' of ' + data.metar_count + ' (Page ' + data.page + ' of ' + data.total_pages + ')';
            } else {
                resultsInfo.textContent = 'Showing ' + data.start_idx + '-' + data.end_idx + ' of ' + data.metar_count + ' stations (Page ' + data.page + ' of ' + data.total_pages + ')';
            }

            // Update search info
            const searchInfo = document.querySelector('.search-box div');
            if (data.search_query) {
                const infoHTML = 'Showing ' + data.metar_count + ' of ' + data.total_count + ' stations matching "' + data.search_query + '" <a href="#" onclick="clearSearch(); return false;" style="margin-left: 10px; color: #007bff; text-decoration: none;">Clear Search</a>';
                if (!searchInfo) {
                    const div = document.createElement('div');
                    div.style.marginTop = '10px';
                    div.style.color = '#666';
                    div.innerHTML = infoHTML;
                    document.querySelector('.search-box').appendChild(div);
                } else {
                    searchInfo.innerHTML = infoHTML;
                }
            } else if (searchInfo) {
                searchInfo.remove();
            }
        }

        function loadPage(page) {
            currentPage = page;
            performSearch(currentSearch, page);
        }

        function clearSearch() {
            searchInput.value = '';
            currentSearch = '';
            currentPage = 1;
            performSearch('', 1);
            searchInput.focus();
        }

        function performSearch(query, page) {
            page = page || 1;
            const url = '/search?search=' + encodeURIComponent(query) + '&page=' + page;

            // Update URL without reload
            const newUrl = query ? '/?search=' + encodeURIComponent(query) + '&page=' + page : '/?page=' + page;
            window.history.pushState({search: query, page: page}, '', newUrl);

            fetch(url)
                .then(response => response.json())
                .then(data => {
                    updateTable(data);
                })
                .catch(error => {
                    console.error('Search error:', error);
                });
        }

        searchInput.addEventListener('input', function(e) {
            clearTimeout(searchTimeout);
            const query = this.value.trim();

            // Auto-search after 2+ characters with 500ms debounce
            if (query.length >= 2) {
                searchTimeout = setTimeout(() => {
                    currentSearch = query;
                    currentPage = 1;
                    performSearch(query, 1);
                }, 500);
            } else if (query.length === 0) {
                // Clear search immediately when input is empty
                clearSearch();
            }
        });

        // Also submit on Enter key
        searchInput.addEventListener('keypress', function(e) {
            if (e.key === 'Enter') {
                e.preventDefault();
                clearTimeout(searchTimeout);
                currentSearch = this.value.trim();
                currentPage = 1;
                if (currentSearch.length > 0) {
                    performSearch(currentSearch, 1);
                }
            }
        });

        // Handle browser back/forward buttons
        window.addEventListener('popstate', function(e) {
            if (e.state) {
                currentSearch = e.state.search || '';
                currentPage = e.state.page || 1;
                searchInput.value = currentSearch;
                performSearch(currentSearch, currentPage);
            }
        });

        // Auto-focus search box on page load
        searchInput.focus();

        // Auto-refresh data every 30 seconds without page reload
        setInterval(function() {
            performSearch(currentSearch, currentPage);
        }, 30000);
    </script>
</body>
</html>
`
