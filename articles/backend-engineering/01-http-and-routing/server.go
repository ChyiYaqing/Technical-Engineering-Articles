package main

import (
	"encoding/json"
	"errors"
	"html/template"
	"log"
	"net/http"
)

const (
	pageAddress = "localhost:8080"
	apiAddress  = "localhost:8081"
)

func main() {
	pageOrigin := "http://" + pageAddress
	apiOrigin := "http://" + apiAddress

	pageServer := &http.Server{
		Addr:    pageAddress,
		Handler: pageHandler(pageOrigin, apiOrigin),
	}
	apiServer := &http.Server{
		Addr:    apiAddress,
		Handler: logRequests(apiHandler(pageOrigin)),
	}

	errorsFromServers := make(chan error, 2)
	go func() {
		log.Printf("实验页面：http://%s", pageAddress)
		errorsFromServers <- pageServer.ListenAndServe()
	}()
	go func() {
		log.Printf("API 服务：http://%s", apiAddress)
		errorsFromServers <- apiServer.ListenAndServe()
	}()

	if err := <-errorsFromServers; !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func pageHandler(pageOrigin, apiOrigin string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := pageTemplate.Execute(w, struct {
			PageOrigin string
			APIOrigin  string
		}{pageOrigin, apiOrigin}); err != nil {
			log.Printf("render page: %v", err)
		}
	})
	return mux
}

func apiHandler(allowedOrigin string) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/no-cors", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, map[string]string{
			"message": "服务器返回了数据，但响应没有 CORS 许可头",
		})
	})

	mux.HandleFunc("/api/cors", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		allowExperimentOrigin(w, r, allowedOrigin)
		writeJSON(w, map[string]string{
			"message": "浏览器允许页面 JavaScript 读取这份响应",
		})
	})

	mux.HandleFunc("/api/preflight", func(w http.ResponseWriter, r *http.Request) {
		allowExperimentOrigin(w, r, allowedOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "PUT, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-CORS-Lesson")
		w.Header().Set("Access-Control-Max-Age", "10")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPut {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, map[string]string{
			"message": "预检通过，浏览器随后发送了 PUT 请求",
		})
	})

	return mux
}

func allowExperimentOrigin(w http.ResponseWriter, r *http.Request, allowedOrigin string) {
	w.Header().Add("Vary", "Origin")
	if r.Header.Get("Origin") == allowedOrigin {
		w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
	}
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("write response: %v", err)
	}
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("API 收到 %-7s %-20s Origin=%q", r.Method, r.URL.Path, r.Header.Get("Origin"))
		next.ServeHTTP(w, r)
	})
}

var pageTemplate = template.Must(template.New("page").Parse(`<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>CORS 浏览器实验</title>
  <style>
    body { max-width: 760px; margin: 48px auto; padding: 0 20px; font: 16px/1.6 system-ui, sans-serif; color: #202124; }
    button { margin: 0 8px 12px 0; padding: 9px 14px; cursor: pointer; }
    pre { min-height: 72px; padding: 14px; overflow: auto; background: #f4f5f7; border-radius: 8px; }
    code { color: #9c2b58; }
  </style>
</head>
<body>
  <h1>CORS 浏览器实验</h1>
  <p>本页面来自 <code>{{.PageOrigin}}</code>，API 来自 <code>{{.APIOrigin}}</code>。端口不同，因此它们是两个源。</p>
  <p>打开开发者工具的 Network 和 Console 面板，再依次点击按钮。</p>
  <button data-test="no-cors">1. 无 CORS 许可头</button>
  <button data-test="cors">2. 允许当前源</button>
  <button data-test="preflight">3. 触发预检请求</button>
  <pre id="result">等待实验……</pre>

  <script>
    const result = document.querySelector('#result');
    const apiOrigin = {{.APIOrigin}};

    async function run(label, url, options) {
      result.textContent = label + '\n请求中……';
      try {
        const response = await fetch(url, options);
        const data = await response.json();
        result.textContent = label + '\n成功：' + JSON.stringify(data, null, 2);
      } catch (error) {
        result.textContent = label + '\n失败：' + error.message + '\n请同时查看 Console、Network 和 Go 服务日志。';
      }
    }

    document.querySelector('[data-test="no-cors"]').addEventListener('click', () => {
      run('实验 1：请求会到达 API，但 JavaScript 不能读取响应', apiOrigin + '/api/no-cors');
    });

    document.querySelector('[data-test="cors"]').addEventListener('click', () => {
      run('实验 2：响应许可当前源读取', apiOrigin + '/api/cors');
    });

    document.querySelector('[data-test="preflight"]').addEventListener('click', () => {
      run('实验 3：先 OPTIONS，获批后再 PUT', apiOrigin + '/api/preflight', {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          'X-CORS-Lesson': 'preflight'
        },
        body: JSON.stringify({ lesson: 'cors' })
      });
    });
  </script>
</body>
</html>`))
