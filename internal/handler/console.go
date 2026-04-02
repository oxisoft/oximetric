package handler

import (
	"html/template"
	"io/fs"
	"net/http"
	"strings"

	"github.com/oxisoft/oximetric/internal/console"
)

type pageData struct {
	Title        string
	Content      template.HTML
	Scripts      template.HTML
	HideControls bool
}

type ConsoleHandler struct {
	loginTmpl  *template.Template
	layoutTmpl *template.Template
	helpTmpl   *template.Template
	pages      map[string]pageData
	staticFS   http.Handler
	robotsTxt  []byte
	domain     string
}

type helpTemplateData struct {
	Title   string
	Content template.HTML
	Scripts template.HTML
}

func NewConsoleHandler(domain string) (*ConsoleHandler, error) {
	staticFS, err := fs.Sub(console.StaticFS, "static")
	if err != nil {
		return nil, err
	}

	loginTmpl, err := template.New("login").Parse(mustRead(staticFS, "templates/login.html"))
	if err != nil {
		return nil, err
	}

	layoutStr := mustRead(staticFS, "templates/layout.html")
	layoutTmpl, err := template.New("layout").Parse(layoutStr)
	if err != nil {
		return nil, err
	}

	helpRaw := mustRead(staticFS, "templates/help.html")
	helpContentTmpl, err := template.New("help-content").Parse(helpRaw)
	if err != nil {
		return nil, err
	}

	robotsTxt, _ := fs.ReadFile(staticFS, "robots.txt")

	if domain == "" {
		domain = "http://localhost:6940"
	}

	pages := map[string]pageData{
		"/dashboard":       {Title: "Dashboard", Content: template.HTML(mustRead(staticFS, "templates/dashboard.html")), Scripts: scriptTag("/static/js/dashboard.js")},
		"/events":          {Title: "Events", Content: template.HTML(mustRead(staticFS, "templates/events.html")), Scripts: scriptTag("/static/js/events.js")},
		"/devices":         {Title: "Devices", Content: template.HTML(mustRead(staticFS, "templates/devices.html")), Scripts: scriptTag("/static/js/devices.js")},
		"/geo":             {Title: "Geography", Content: template.HTML(mustRead(staticFS, "templates/geo.html")), Scripts: scriptTag("/static/js/geo.js")},
		"/users-analytics": {Title: "Users", Content: template.HTML(mustRead(staticFS, "templates/users-analytics.html")), Scripts: scriptTag("/static/js/users-analytics.js")},
		"/projects":        {Title: "Projects", Content: template.HTML(mustRead(staticFS, "templates/projects.html")), Scripts: scriptTag("/static/js/projects.js"), HideControls: true},
		"/console-users":   {Title: "Console Users", Content: template.HTML(mustRead(staticFS, "templates/console-users.html")), Scripts: scriptTag("/static/js/console-users.js"), HideControls: true},
		"/account":         {Title: "Account", Content: template.HTML(mustRead(staticFS, "templates/account.html")), Scripts: scriptTag("/static/js/account.js"), HideControls: true},
		"/about":           {Title: "About", Content: template.HTML(mustRead(staticFS, "templates/about.html")), Scripts: scriptTag("/static/js/about.js"), HideControls: true},
	}

	// Pre-render help template with domain
	var helpBuf strings.Builder
	helpContentTmpl.Execute(&helpBuf, map[string]string{"Domain": domain})

	pages["/help"] = pageData{
		Title:        "Help",
		Content:      template.HTML(helpBuf.String()),
		Scripts:      scriptTag("/static/js/help.js"),
		HideControls: true,
	}

	return &ConsoleHandler{
		loginTmpl:  loginTmpl,
		layoutTmpl: layoutTmpl,
		pages:      pages,
		staticFS:   http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))),
		robotsTxt:  robotsTxt,
		domain:     domain,
	}, nil
}

func noIndex(w http.ResponseWriter) {
	w.Header().Set("X-Robots-Tag", "noindex, nofollow")
}

func (h *ConsoleHandler) Login(w http.ResponseWriter, r *http.Request) {
	noIndex(w)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	h.loginTmpl.Execute(w, nil)
}

func (h *ConsoleHandler) Page(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	page, ok := h.pages[path]
	if !ok {
		http.Redirect(w, r, "/dashboard", http.StatusFound)
		return
	}
	noIndex(w)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	h.layoutTmpl.Execute(w, page)
}

func (h *ConsoleHandler) Static(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(r.URL.Path, ".css") {
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	} else if strings.HasSuffix(r.URL.Path, ".js") {
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	}
	h.staticFS.ServeHTTP(w, r)
}

func (h *ConsoleHandler) RobotsTxt(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write(h.robotsTxt)
}

func mustRead(fsys fs.FS, name string) string {
	data, err := fs.ReadFile(fsys, name)
	if err != nil {
		panic("console: missing " + name + ": " + err.Error())
	}
	return string(data)
}

func scriptTag(path string) template.HTML {
	return template.HTML(`<script src="` + template.HTMLEscapeString(path) + `"></script>`)
}
