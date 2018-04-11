package main
import (
	"fmt"
	"regexp"
	"strings"
	"net/http"
	"encoding/json"
	"io/ioutil"
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/denisenkom/go-mssqldb"
	_ "github.com/mattn/go-adodb"
	_ "github.com/alexbrainman/odbc"
	_ "github.com/kshvakov/clickhouse"
	_ "github.com/nakagami/firebirdsql"
	_ "github.com/mattn/go-sqlite3"

	// _ "github.com/mattn/go-oci8"
)

var config Config
var connections map[string]*sql.DB
var reSafe = regexp.MustCompile("[^a-zA-Z0-9_\\.$]+")

type Config struct {
	Port string
	Databases map[string]Database
	Metrics map[string]Metric
}

type Database struct {
	Driver string
	Dns string
	Metrics []string
}

type Metric struct {
	Query string
	Description string
}

func sendJson(w http.ResponseWriter, data interface{}) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)	
		return 
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
	return
}

func sendText(w http.ResponseWriter, data []map[string]interface {}, columns []string) {
	if len(data) == 0 {
		fmt.Fprintf(w, "")
		return
	}
	
	text := ""
	for i, row := range (data) {
		values := make([]string, 0);
		for _, col := range(columns) {
			if row[col] != nil {
				values = append(values, fmt.Sprintf("%v", row[col]))
			} else {
				values = append(values, "null")
			}
		}

		text = text + strings.Join(values, ";")
		if i != len(data) - 1 {
			text = text + "\n"
		}
	}
	
	fmt.Fprintf(w, "%v", text)
	return
}

func httpHandler(w http.ResponseWriter, r *http.Request) {
	url := strings.Split(r.URL.Path, "/")
	if (len(url) < 2) {
		http.Error(w, "UNKNOWN", http.StatusInternalServerError)
		return
	}

	alias := url[1]
	metric := ""
	if len(url) > 2 {
		metric = url[2]
	}

	if alias == "" {
		keys := []string{}
		for k := range config.Databases {
			keys = append(keys, k)
		}

		sendJson(w, keys)
		return
	}

	prop, ok := config.Databases[alias]; 
	if !ok {
		http.Error(w, "NOT_FOUND", http.StatusNotFound)
		return
	}

	if metric == "" {
		mdesc := make(map[string]string)
		for _, name := range (prop.Metrics) {
			m, ok := config.Metrics[name];
			if ok {
				mdesc[name] = m.Description			
			}
		}
		sendJson(w, mdesc)
		return
	}

	hasMetric := false
	for _, m := range(config.Databases[alias].Metrics) {
		hasMetric = hasMetric || m == metric
	}

	if !hasMetric {
		http.Error(w, "NOT_FOUND", http.StatusNotFound)
		return
	}
	
	mprop, ok := config.Metrics[metric]; 
	if  !ok {
		http.Error(w, "NOT_FOUND", http.StatusNotFound)
		return
	}

	var err error
	db, ok := connections[alias]
	if !ok || db == nil  {
		db, err = sql.Open(prop.Driver, prop.Dns)
		if err != nil {
			http.Error(w, "ECONNREFUSED", http.StatusInternalServerError)
			fmt.Println(alias +"/" + metric + "\n", err.Error())
			return		
		}
		connections[alias] = db
	} else {
		err = db.Ping()
		if err != nil {
			http.Error(w, "ECONNREFUSED", http.StatusInternalServerError)
			fmt.Println(alias +"/" + metric + "\n", err.Error())
			defer db.Close()
			return
		}
	}

	query := mprop.Query
	var params []interface{}
	for param, value := range (r.URL.Query()) {
		if len(value) > 0 && len(value[0]) < 64 {
			val := reSafe.ReplaceAllString(value[0], "")
			query = strings.Replace(query, "#" + param, val, -1)
		}

		if len(value) > 0 && strings.Contains(query, "$" + param) {
			params = append(params, value[0]);
			query = strings.Replace(query, "$" + param, "?", 1)
		}
	}

	rows, err := db.Query(query, params...)
	if err != nil {
		http.Error(w, "SQL_ERROR", http.StatusInternalServerError)
		fmt.Println(alias +"/" + metric + "\n", "Query: ", query, "\n", "Params: ", params, "\n", err.Error())
		return
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	count := len(columns)
	tableData := make([]map[string]interface{}, 0)
	values := make([]interface{}, count)
	valuePtrs := make([]interface{}, count)
	for rows.Next() {
		for i := 0; i < count; i++ {
			valuePtrs[i] = &values[i]
		}
		rows.Scan(valuePtrs...)
		entry := make(map[string]interface{})
		for i, col := range columns {
			var v interface{}
			val := values[i]
			b, ok := val.([]byte)
			if ok {
				v = string(b)
			} else {
				v = val
			}
			entry[col] = v
		}
		tableData = append(tableData, entry)
	}

	_, jsonok := r.URL.Query()["json"]
	_, textok := r.URL.Query()["text"]
	if textok || !jsonok && strings.HasPrefix(r.Header.Get("Accept"), "text") {
		sendText(w, tableData, columns)
		return
	}
	
	sendJson(w, tableData)
	return
}

func main() {
	file, err := ioutil.ReadFile("./config.json")
	if err != nil {
		fmt.Printf("Couldn't  read config.json\n", err.Error())
		return
	}

	err = json.Unmarshal(file, &config)
	if err != nil {
		fmt.Println("Couldn't  parse config.json\n", err.Error())
		return
	}

	connections = make(map[string]*sql.DB, 0)

	if config.Port == "" {
		config.Port = "9000"
	}

	http.HandleFunc("/", httpHandler)
	fmt.Println("Httpsql running on " + config.Port + " port\n")
	err = http.ListenAndServe(":" + config.Port, nil)
	if err != nil {
		fmt.Println("Couldn't start http server\n", err.Error())
		return
	}
}