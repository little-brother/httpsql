Httpsql is a http server to provides a simple way of monitoring SQL databases via a http request by predefined queries.

### How to build app
1. Download and install [Golang](https://golang.org/dl/)
2. Download dependencies
   ```bash
   go get -v -d ./
    ```
3. Build application
   ```bash 
   go build -ldflags="-s -w"
   ```

Also you can download [binary](https://github.com/little-brother/httpsql/releases).

### How to use
Before continue you must create `config.json` in app folder. Below is an example:

<pre>
{
  "port": "9000",
  "databases": {
  
    "demo": {
      "driver": "mysql",
      "dns": "myuser@tcp(192.168.0.101)/mydb",
      "metrics": ["now", "count", "minmax", "getbyid"]
    },
    
    "demo2": {
      "driver": "postgres",
      "dns": "host=192.168.0.101 user=home password=password dbname=mydb2 sslmode=disable",
      "metrics": ["count", "minmax"]
    }
  },

  "metrics": {
  
    "now": {
      "query": "select now()",
      "description": "Params: none. Returns current date and time."
    },

    "count": {
      "query": "select count(1) as count from #table",
      "description": "Params: table. Returns row count of table."
    },
    
    "minmax": {
      "query": "select min(#column) min, max(#column) max from #table",
      "description": "Params: table, column. Returns max and min value."
    },
    
    "getbyid": {
      "query": "select * from #table where id = $id",
      "description": "Params: table, id. Returns row with requested id."
    }
  }
}
</pre>

The following links will be available for this configuration:
* `/` returns all database aliases: `demo` and `demo2`
* `/demo` returns all available metrics for database `demo` and their description
* `/demo/now` returns `mydb` date and time
* `/demo/count?table=orders` returns row count for `mydb.orders`
* `/demo/minmax?table=orders&column=price` returns minimal and maximum `price` in `mydb.orders`
* `/demo/getbyid?table=orders&id=10` returns order detail with id = `10`
* `/demo2/count?table=customers` returns customer count in `mydb2.customers`
* `/demo2/minmax?table=...`
 
In query `#param` defines a url parameter `param` that value will be substituted directly into the query. To avoid sql injections, all characters except `a-Z0-9_.$` will be removed from value and length is limited to 64 characters. `$param` defines a placeholder parameter and can contains any symbols.
<br>

Request result is `json` or `text`(csv). By default data format defines by http `Accept` header. You can lock format by adding `json` or `text` to requested url e.g. `/demo2/count?table=customers&text`.
<br>

| One permanent connection is used for each database. If necessary, the connection will be restored. |
|---|

### Supported databases

|DBMS|Driver|Dns example|
|-----|--------|----------|
|MySQL|[mysql](https://github.com/go-sql-driver/mysql)|myuser@tcp(192.168.0.101)/mydb|
|PosgreSQL|[postgres](https://github.com/lib/pq)|host=192.168.0.101 user=home password=password dbname=mydb sslmode=disable|
|MSSQL|[mssql](https://github.com/denisenkom/go-mssqldb)|sqlserver://username:password@host/instance?param1=value&param2=value<br>sqlserver://username:password@host:port?param1=value&param2=value|
|ADODB|[adodb](https://github.com/mattn/go-adodb)|Provider=Microsoft.Jet.OLEDB.4.0;Data Source=my.mdb;|
|ODBC|[odbc](https://github.com/alexbrainman/odbc)|Driver={Microsoft ODBC for Oracle};Server=ORACLE8i7;Persist Security Info=False;Trusted_Connection=Yes|
|ClickHouse|[clickhouse](https://github.com/kshvakov/clickhouse)|tcp://127.0.0.1:9000?username=&debug=true|
|Firebird|[firebirdsql](https://github.com/nakagami/firebirdsql)|user:password@servername/foo/bar.fdb|
|SQLite3|[sqlite3](https://github.com/mattn/go-sqlite3)|D:/aaa/bbb/mydb.sqlite|

> Notice: most databases require additional configuration for remote connections

You can add other [drivers](https://github.com/golang/go/wiki/SQLDrivers) but some of them requires additional software.