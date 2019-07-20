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
	Token      string
	Binstr     string
	Decode     string
	Returncode string
}

func tokenCreate() string {
	ct := time.Now().Unix()
	h := md5.New()
	io.WriteString(h, strconv.FormatInt(ct, 10))
	token := fmt.Sprintf("%x", h.Sum(nil))
	//fmt.Println("token created :", token)
	return token
}
func PostDataHandler(w http.ResponseWriter, r *http.Request) {
	var e error
	ti := time.Now().Format("2006-01-02 15:04:05")
	if r.Method == "GET" {
		var context DataContext
		t, e := template.ParseFiles("./templates/datapost.html")
		if e != nil {
			http.Error(w, e.Error(), http.StatusInternalServerError)
			return
		}
		context.Token = tokenCreate()
		fmt.Println(ti, "the r.methond ", r.Method, "create token", context.Token)
		expiration := time.Now().Add(365 * 24 * time.Hour)
		cookie := http.Cookie{Name: "csrftoken", Value: context.Token, Expires: expiration}
		http.SetCookie(w, &cookie)
		e = t.Execute(w, context)
		if e != nil {
			http.Error(w, e.Error(), http.StatusInternalServerError)
			return
		}
	} else if r.Method == "POST" {
		var context DataContext
		e = r.ParseForm()
		if e != nil {
			http.Error(w, e.Error(), http.StatusInternalServerError)
			return
		}

		r.ParseMultipartForm(32 << 20) //defined maximum size of file
		context.Returncode = "Parse done"
		formToken := template.HTMLEscapeString(r.Form.Get("CSRFToken"))
		context.Binstr = template.HTMLEscapeString(r.Form.Get("bodyin"))
		context.Token = formToken
		cookie, e := r.Cookie("csrftoken")
		if e != nil {
			log.Print(e)
			return
		}
		n := strings.Split(r.RemoteAddr, ":")[0] + "-" + strings.TrimLeft(strings.Fields(r.UserAgent())[1], "(")
		uname := strings.TrimRight(n, ";")
		fmt.Printf("%s %s %s  with cookie token %s and form token %s\n",
			ti, uname, r.Method, cookie.Value, context.Token)
		fmt.Println("indata :\n", context.Binstr)
		if formToken == cookie.Value {
			context.Returncode = "Get EqualToken done"
			file, handler, e := r.FormFile("uploadfile")
			if e != nil {
				http.Error(w, e.Error(), http.StatusInternalServerError)
				return
			}
			if handler != nil {
				defer file.Close()

				e = os.Mkdir("./runcmd/"+formToken, os.ModePerm)
				if e != nil {
					log.Println(e)
					context.Returncode = "Can't create the dir!"
					return
				}
				context.Returncode = "create the dir done"
				f, e := os.OpenFile("./runcmd/"+formToken+"/my.proto", os.O_WRONLY|os.O_CREATE, 0666)
				if e != nil {
					log.Println(e)
					context.Returncode = "Can't create the file!"
					return
				}
				defer f.Close()
				io.Copy(f, file)
				context.Returncode = "upload file done"

				context.Decode = "upload file done"
				context.Returncode = "Success!"

			} else {
				context.Returncode = "Can't read the src file!"
				log.Fatal("Can't create the data source file")
			}
		} else {
			log.Print("form token mismatch")
			context.Returncode = "form token mismatch"
		}
		b, e := template.ParseFiles("./templates/datapost.html")
		if e != nil {
			http.Error(w, e.Error(), http.StatusInternalServerError)
			return
		}
		e = b.Execute(w, context)
		if e != nil {
			http.Error(w, e.Error(), http.StatusInternalServerError)
			return
		}
		//http.Redirect(w, r, "/", 302)
	} else {
		log.Print("Unknown request")
		http.Redirect(w, r, "/", 302)
	}

}

func main() {
	http.HandleFunc("/", PostDataHandler)
	http.Handle("/templates/", http.StripPrefix("/templates/", http.FileServer(http.Dir("./templates"))))
	http.Handle("/runcmd/", http.StripPrefix("/runcmd/", http.FileServer(http.Dir("./runcmd"))))
	log.Print("Running the server on port 8091.")
	log.Fatal(http.ListenAndServe(":8091", nil))
}
