package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/gorilla/sessions"
)

type PostContext struct {
	Context string
}

var Store = sessions.NewCookieStore([]byte("hpb"))

func showBoard(pattern string, w http.ResponseWriter) error {
	t, e := template.ParseFiles("./templates/board.html")
	if e != nil {
		http.Error(w, e.Error(), http.StatusInternalServerError)
		return e
	}
	e = t.Execute(w, PostContext{Context: "Pls input the data"})
	if e != nil {
		http.Error(w, e.Error(), http.StatusInternalServerError)
		return e
	}
	return e
}
func rootHandler(w http.ResponseWriter, r *http.Request) {
	var e error
	if r.Method == "GET" {
		session, _ := Store.Get(r, "session")

		if session.Values["act"] != "true" {
			session.Values["act"] = "true"
			e = session.Save(r, w)
			if e != nil {
				log.Fatal(e)
			}
			fmt.Println("write the session data")
		}

		e = showBoard("*", w)
		if e != nil {
			log.Print(e)
		}
	} else if r.Method == "POST" {
		session, e := Store.Get(r, "session")
		if e != nil {
			log.Fatal(e)
		}
		if session.Values["act"] == "true" {
			e = r.ParseForm()
			if e != nil {
				http.Error(w, e.Error(), http.StatusInternalServerError)
				return
			}

			//t := time.Now().Format("2006-01-02 15:04:05")
			//n := strings.Split(r.RemoteAddr, ":")[0] + "-" + strings.TrimLeft(strings.Fields(r.UserAgent())[1], "(")
			//uname := strings.TrimRight(n, ";")
			// fmt.Println("uanme =", uname)
			//p := PostData{UserName: uname, Content: r.Form["body"][0], Created: t}

			e = showBoard("*", w)
			if e != nil {
				log.Print(e)
			}
			// session.Values["username"] = username

		} else {
			fmt.Println("the session is not active go to root")
			http.Redirect(w, r, "/", 302)
		}
	} else {
		http.Error(w, "Unknown HTTP Action", http.StatusInternalServerError)
		return

	}
}
func main() {
	fmt.Println("vim-go")
	//http.HandleFunc("/addpost/", AddPostHandler)
	http.HandleFunc("/", rootHandler)
	//http.Handle("/templates/", http.StripPrefix("/templates/", http.FileServer(http.Dir("./templates"))))
	log.Print("Running the server on port 8091.")
	log.Fatal(http.ListenAndServe(":8091", nil))
}
