package main

import (
	"crypto/md5"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
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
	fmt.Println("the r.methond is", r.Method)
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

		r.ParseMultipartForm(32 << 20) //defined maximum size of file
		formToken := template.HTMLEscapeString(r.Form.Get("CSRFToken"))
		cookie, e := r.Cookie("csrftoken")
		if e != nil {
			log.Print(e)
			return
		}
		//fmt.Println("formtoken", formToken, "===", "cooke.value", cookie.Value)
		if formToken == cookie.Value {
			file, handler, e := r.FormFile("uploadfile")
			if e != nil {
				http.Error(w, e.Error(), http.StatusInternalServerError)
				return
			}
			if handler != nil {
				defer file.Close()
				f, e := os.OpenFile("./srcproto/"+formToken, os.O_WRONLY|os.O_CREATE, 0666)
				if e != nil {
					log.Println(e)
					return
				}
				defer f.Close()
				io.Copy(f, file)
				fmt.Println("upload a file done!")

				t := time.Now().Format("2006-01-02 15:04:05")
				n := strings.Split(r.RemoteAddr, ":")[0] + "-" + strings.TrimLeft(strings.Fields(r.UserAgent())[1], "(")
				uname := strings.TrimRight(n, ";")
				fmt.Println("uanme =", uname, t)

			} else {
				log.Fatal("Can't create the data source file")
			}
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
	http.Handle("/files/", http.StripPrefix("/files/", http.FileServer(http.Dir("./files"))))
	log.Print("Running the server on port 8091.")
	log.Fatal(http.ListenAndServe(":8091", nil))
}
