package main

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

//ArticleHandler 文章目录下的文章
func ArticleHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Category: %v id:%v\n", vars["category"], vars["id"])

	fmt.Println("Category:", vars["category"], "id:", vars["id"])
}

//ArticlesCategoryHandler 文章目录
func ArticlesCategoryHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Category: %v\n", vars["category"])
	fmt.Println("Category:", vars["category"])
}

func main() {
	r := mux.NewRouter()
	//r.HandleFunc("/products/{key}", ProductHandler)
	r.HandleFunc("/articles/{category}", ArticlesCategoryHandler)
	r.HandleFunc("/articles/{category}/{id:[0-9]+}", ArticleHandler)

	http.ListenAndServe("0.0.0.0:7777", r)
}
