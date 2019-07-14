package main

import (
	"crypto/md5"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type DataContext struct {
	Context string
}

func tokenCreate() string {
	ct := time.Now().Unix()
	h := md5.New()
	io.WriteString(h, strconv.FormatInt(ct, 10))
	token := fmt.Sprintf("%x", h.Sum(nil))
	fmt.Println("token created :", token)
	return token
}
func PostDataHandler(w http.ResponseWriter, r *http.Request) {
	var e error
	if r.Method == "GET" {
		t, e := template.ParseFiles("./templates/datapost.html")
		if e != nil {
			http.Error(w, e.Error(), http.StatusInternalServerError)
			return
		}
		token := tokenCreate()
		expiration := time.Now().Add(365 * 24 * time.Hour)
		cookie := http.Cookie{Name: "csrftoken", Value: token, Expires: expiration}
		http.SetCookie(w, &cookie)
		e = t.Execute(w, token)
		if e != nil {
			http.Error(w, e.Error(), http.StatusInternalServerError)
			return
		}
	} else if r.Method == "POST" {
		e = r.ParseForm()
		if e != nil {
			http.Error(w, e.Error(), http.StatusInternalServerError)
			return
		}
		//
		formToken := template.HTMLEscapeString(r.Form.Get("CSRFToken"))
		cookie, e := r.Cookie("csrftoken")
		if e != nil {
			log.Print(e)
			return
		}
		if formToken == cookie.Value {

			t := time.Now().Format("2006-01-02 15:04:05")
			n := strings.Split(r.RemoteAddr, ":")[0] + "-" + strings.TrimLeft(strings.Fields(r.UserAgent())[1], "(")
			uname := strings.TrimRight(n, ";")
			fmt.Println("uanme =", uname, t)
		} else {
			log.Print("form token mismatch")
		}
		http.Redirect(w, r, "/", 302)
	} else {
		log.Print("Unknown request")
		http.Redirect(w, r, "/", 302)
	}

}

func main() {
	http.HandleFunc("/", PostDataHandler)
	http.Handle("/templates/", http.StripPrefix("/templates/", http.FileServer(http.Dir("./templates"))))
	log.Print("Running the server on port 8091.")
	log.Fatal(http.ListenAndServe(":8091", nil))
}
