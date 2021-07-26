package main

import (
  "os"
  // "os/signal"
  // "syscall"
  "fmt"
  "log"
  "time"
  "strings"
  "strconv"
  "net/http"
  "database/sql"
  _ "github.com/mattn/go-sqlite3"
  "github.com/prometheus/client_golang/prometheus"
  "github.com/prometheus/client_golang/prometheus/promhttp"
  "github.com/prometheus/client_golang/prometheus/promauto"
  "github.com/gorilla/mux"
)

var storage = getenv("STORAGE", "file")

func checkErr(err error) {
  if err != nil {
    log.Fatal(err.Error())
  }
}

func getenv(key, fallback string) string {
    value := os.Getenv(key)
    if len(value) == 0 {
        return fallback
    }
    return value
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func NewResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{w, http.StatusOK}
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

var totalRequests = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Number of get requests.",
	},
	[]string{"code", "path", "method"},
)

var responseStatus = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "response_status",
		Help: "Status of HTTP response",
	},
	[]string{"status"},
)

var httpDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Name: "http_response_time_seconds",
	Help: "Duration of HTTP requests.",
}, []string{"path"})

func prometheusMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route := mux.CurrentRoute(r)
		path, _ := route.GetPathTemplate()
    method := r.Method

		timer := prometheus.NewTimer(httpDuration.WithLabelValues(path))
		rw := NewResponseWriter(w)
		next.ServeHTTP(rw, r)

		statusCode := rw.statusCode

		responseStatus.WithLabelValues(strconv.Itoa(statusCode)).Inc()

    totalRequests.WithLabelValues(strconv.Itoa(statusCode), path, method).Inc()

		timer.ObserveDuration()
	})
}

func init() {
	prometheus.Register(totalRequests)
	prometheus.Register(responseStatus)
	prometheus.Register(httpDuration)
}

func main() {
  router := mux.NewRouter()
  router.Use(prometheusMiddleware)

  router.HandleFunc("/hello", HelloPage).Methods("GET")
  router.HandleFunc("/", HelloServer).Methods("GET")
  router.HandleFunc("/user", LogAccess).Methods("GET", "POST")
  router.Handle("/metrics", promhttp.Handler()).Methods("GET")

  http.ListenAndServe(":8080", router)
  log.Printf("Listening on port 8080")
}

func Metrics(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "Hello, %s!", r.URL.Path[1:])
}

func HelloServer(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "Hello, %s!", r.URL.Path[1:])
}

func HelloPage(w http.ResponseWriter, r *http.Request) {
  if r.Method == "GET" {
    w.WriteHeader(http.StatusOK)
    log.Printf("Display hello-page")
    fmt.Fprintf(w, "Hello Page")
    return
  }
  log.Printf("Client used wrong method")
  w.WriteHeader(http.StatusMethodNotAllowed)
}

func LogAccess(w http.ResponseWriter, r *http.Request) {
  if storage == "file" {
    if r.Method == "POST" {

      name := r.FormValue("name")
      if name == "" {
        w.WriteHeader(http.StatusExpectationFailed)
        fmt.Fprintf(w, "name not defined")
      }


      file, err := os.OpenFile("db.txt", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
      if err != nil {
          panic(err)
      }
      defer file.Close()

      _, err2 := file.WriteString(fmt.Sprintf("%v,%v\n", name, time.Now().Format(time.RFC3339)))

      if err2 != nil {
          log.Fatal(err2)
      }

      w.WriteHeader(http.StatusOK)
      log.Printf("Add '" + name + "' to journal")
      return
    }
    w.WriteHeader(http.StatusNotImplemented)
  }
  if storage == "sql" {
    user := r.FormValue("name")
    database, err := sql.Open("sqlite3", "./db.sql")
    checkErr(err)
    defer database.Close()
    createTable(database)

    if r.Method == "GET" {
      timestamps := showTimestamps(database, user)

      fmt.Fprintf(w, strings.Join(timestamps, "\n"))

      log.Printf("Show log")
      return
    }

    if r.Method == "POST" {
      insertRow(database, user, time.Now().Format(time.RFC3339))

      w.WriteHeader(http.StatusOK)
      log.Printf("Add '" + user + "' to journal")
      return
    }
    w.WriteHeader(http.StatusNotImplemented)
  }
}

func insertRow(db *sql.DB, user string, timestamp string) {
	log.Println("Inserting record ...")
	insertRowSQL := `INSERT INTO users (user, timestamp) values (?, ?)`

	statement, err := db.Prepare(insertRowSQL)
  checkErr(err)
                                                   // This is good to avoid SQL injections
	_, err = statement.Exec(user, timestamp)
  checkErr(err)
}

func showTimestamps(db *sql.DB, u string) []string {
  rows, err := db.Query("SELECT id, user, timestamp FROM users WHERE user IS ?", u)
  checkErr(err)
  defer rows.Close()

  timestamps := make([]string, 0)

  var id int
  var user string
  var timestamp string
  for rows.Next() {
    rows.Scan(&id, &user, &timestamp)
    timestamps = append(timestamps, timestamp)
    log.Printf("Got row: " + strconv.Itoa(id))
  }
  return timestamps
}

func createTable(db *sql.DB) {
	createTableSQL := `CREATE TABLE IF NOT EXISTS users (
      "id" integer NOT NULL PRIMARY KEY AUTOINCREMENT,		
      "user" TEXT,
      "timestamp" TEXT
      );`

  log.Println("Creating table...")
  statement, err := db.Prepare(createTableSQL)
  checkErr(err)
  statement.Exec()
  log.Println("Table created")
}
