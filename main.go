package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/Lynx/db"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	client     = db.GetDBCli()
	database   *mongo.Database
	collection *mongo.Collection
)

func sayhelloName(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()       //解析參數，預設是不會解析的
	fmt.Println(r.Form) //這些資訊是輸出到伺服器端的列印資訊
	fmt.Println("path", r.URL.Path)
	fmt.Println("scheme", r.URL.Scheme)
	fmt.Println(r.Form["url_long"])
	for k, v := range r.Form {
		fmt.Println("key:", k)
		fmt.Println("val:", strings.Join(v, ""))
	}
	fmt.Fprintf(w, "Hello astaxie!") //這個寫入到 w 的是輸出到客戶端的
}

// type RouteMux struct {
// }

// func (p *RouteMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
// 	if r.URL.Path == "/" {
// 		sayhelloName(w, r)
// 		return
// 	}
// 	if r.URL.Path == "/labels" {
// 		respond.GetLabelsByTaskId(collection, w, r)
// 		return
// 	}
// 	http.NotFound(w, r)
// 	return
// }

func main() {
	database = client.Database("LINE_LABEL")
	collection = database.Collection("Label")
	mux := &RouteMux{}
	http.ListenAndServe(":9090", mux)
}
