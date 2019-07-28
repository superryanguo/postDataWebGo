package main

import (
	"crypto/md5"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type DataContext struct {
	Token      string
	Binstr     []byte
	Decode     string
	Returncode string
}

var ProtoFile string = "my.proto"

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
		if r.RequestURI != "/favicon.ico" {
			var context DataContext
			t, e := template.ParseFiles("./templates/datapost.html")
			if e != nil {
				http.Error(w, e.Error(), http.StatusInternalServerError)
				return
			}
			context.Token = tokenCreate()
			fmt.Println(ti, "the r.method ", r.Method, "create token", context.Token)
			expiration := time.Now().Add(365 * 24 * time.Hour)
			cookie := http.Cookie{Name: "csrftoken", Value: context.Token, Expires: expiration}
			http.SetCookie(w, &cookie)
			e = t.Execute(w, context)
			if e != nil {
				http.Error(w, e.Error(), http.StatusInternalServerError)
				return
			}
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
		bodyin := template.HTMLEscapeString(r.Form.Get("bodyin"))
		context.Binstr = []byte(strings.Replace(strings.Replace(bodyin, "\n", "", -1), "\r", "", -1))
		mode := template.HTMLEscapeString(r.Form.Get("Mode"))
		mesgType := template.HTMLEscapeString(r.Form.Get("MessageType"))
		context.Token = formToken
		n := strings.Split(r.RemoteAddr, ":")[0] + "-" + strings.TrimLeft(strings.Fields(r.UserAgent())[1], "(")
		uname := strings.TrimRight(n, ";")
		cookie, e := r.Cookie("csrftoken")
		if e != nil {
			log.Print(e)
			context.Returncode = "cookie read error"
			goto SHOW
		}
		fmt.Printf("%s %s %s  with cookie token %s and form token %s, Mode:%s,Type:%s\n",
			ti, uname, r.Method, cookie.Value, context.Token, mode, mesgType)
		fmt.Println("indata :\n", bodyin)
		if formToken == cookie.Value {
			context.Returncode = "Get EqualToken done"
			file, header, e := r.FormFile("uploadfile")
			if e != nil {
				http.Error(w, e.Error(), http.StatusInternalServerError)
				return
			}
			if header != nil && header.Filename != "" {
				defer file.Close()
				dir := "./runcmd/" + formToken
				_, err := os.Stat(dir)
				if err != nil {
					if os.IsNotExist(err) {
						e = os.Mkdir(dir, os.ModePerm)
						if e != nil {
							log.Println(e)
							context.Returncode = "Can't create the dir!"
							//return //TODO: should we return or go to the html show place?
							goto SHOW
						}
					}
				}
				context.Returncode = "create the dir done"
				upload := "./runcmd/" + formToken + ProtoFile
				f, e := os.OpenFile(upload, os.O_WRONLY|os.O_CREATE, 0666)
				if e != nil {
					log.Println(e)
					context.Returncode = "Can't create the file!"
					goto SHOW
				}
				defer f.Close()
				io.Copy(f, file)
				context.Returncode = "upload file done"

				//run cmd for what you want
				if mode == "Normal" {
					if mesgType != "" {
						output := ParseGpbNormalMode(context.Binstr, mesgType, upload)
						context.Decode = fmt.Sprintf("%s", output)
						context.Returncode = "Successfully Parse Normal mode done!"
					} else {

						context.Returncode = "Error! NormalMode Must fill the messagetype"
					}
				} else if mode == "HardCore" {
					output := HardcoreDecode(upload, context.Binstr)
					context.Decode = fmt.Sprintf("%s", output)
					context.Returncode = "Successfully Parse HardCore mode done!"
				} else {

					context.Returncode = "Unknown parse mode!"
				}

			} else {
				context.Returncode = "Can't read the src file!"
				log.Println("Can't create the data source file, maybe nil or empty upload filename")
			}
		} else {
			log.Print("form token mismatch")
			context.Returncode = "form token mismatch"
		}
	SHOW:
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

func ParseGpbNormalMode(data []byte, message string, proto string) []byte {
	pkg := filterPkg(proto)
	messageType := pkg + "." + pureCmdStringPlus(message)
	fmt.Println("ParseGpbNormalMode Message type:", messageType)
	cmdstr := fmt.Sprintf("echo %x | xxd -r -p | protoc --decode %s %s", data, messageType, proto)
	output := runshell(cmdstr)
	fmt.Printf("ParseGpbNormalMode output is:\n%s\n", output)
	return output

}
func pureCmdString(str string) string {
	return strings.Trim(strings.Trim(strings.Trim(strings.Trim(str, "\n"), "\r"), " "), ";")
}

func pureCmdStringPlus(str string) string {
	return strings.Replace(strings.Replace(strings.Replace(strings.Replace(str, "\n", "", -1), "\r", "", -1), " ", "", -1), ";", "", -1)
}
func filterPkg(proto string) string {
	cmdstr := fmt.Sprintf("awk '$1 == \"package\" {print $2}' %s", proto)
	output := runshell(cmdstr)
	return pureCmdStringPlus(fmt.Sprintf("%s", output))

}
func filterMessageTypes(proto string) []string {
	cmdstr := fmt.Sprintf("awk '$1 == \"message\" {print $2}' %s", proto)
	output := runshell(cmdstr)
	messages := strings.Split(fmt.Sprintf("%s", output), "\n")
	fmt.Printf("before filter return %s\n", messages)
	for i := 0; i < len(messages); {
		if messages[i] == "\n" {
			messages = append(messages[:i], messages[i+1:]...)
		} else {
			i++
		}
	}
	fmt.Printf("After filter return %s\n", messages)
	return messages
}

func HardcoreDecode(proto string, data []byte) []byte {
	var pkgMesg, cmdstr string
	pkg := filterPkg(proto)
	types := filterMessageTypes(proto)
	for k, message := range types {
		pkgMesg = pkg + "." + message
		fmt.Printf("decode the %v type %s", k, pkgMesg)

		cmdstr = fmt.Sprintf("echo %x | xxd -r -p | protoc --decode %s %s", data, pkgMesg, proto)
		fmt.Println("cmd =", cmdstr)
		cmd := exec.Command("sh", "-c", cmdstr)
		output, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Println("DecodeFail on messageType", pkgMesg, "continue...")
			continue
		} else {
			return output
		}
	}

	//finally give a raw decode
	cmdstr = fmt.Sprintf("echo %x | xxd -r -p | protoc --decode_raw", data)
	return runshell(cmdstr)
}
func runshell(shell string) []byte {
	cmd := exec.Command("sh", "-c", shell)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatal(err)
	}
	return output
}
func main() {
	http.HandleFunc("/", PostDataHandler)
	http.Handle("/templates/", http.StripPrefix("/templates/", http.FileServer(http.Dir("./templates"))))
	http.Handle("/runcmd/", http.StripPrefix("/runcmd/", http.FileServer(http.Dir("./runcmd"))))
	log.Print("Running the server on port 8091.")
	log.Fatal(http.ListenAndServe(":8091", nil))
}
