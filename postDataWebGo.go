package main

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	dstore "github.com/superryanguo/postDataWebGo/datastore"

	log "github.com/sirupsen/logrus"
)

//TODO:
//1. use the trace log level to avoid the too many trace[done]
//2. a full cover test case in _test file
//3. auto loadbuild[done]
//4. server show how many visitor
//5. escapebytes to jump the header to real gpb bytes[done]
//6. parse [1] = 65, type data in[done]
//7. server port can be not hard code one[done]
//8. light-weight datastore about the vistor operation[done]
//9. progress bar[not done]
//10. multi-access test
//11. myobject. issue

type DataContext struct {
	Token      string
	Binstr     []byte
	Encode     string
	Decode     string
	Returncode string
}

var ProtoFile string = "my.proto"
var EscapeBytesMax int = 25

func init() {
	//log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	//log.SetLevel(log.InfoLevel)
	log.SetLevel(log.DebugLevel)
}
func tokenCreate() string {
	ct := time.Now().Unix()
	h := md5.New()
	io.WriteString(h, strconv.FormatInt(ct, 10))
	token := fmt.Sprintf("%x", h.Sum(nil))
	return token
}
func PostDataHandler(w http.ResponseWriter, r *http.Request) {
	var e error
	var summary string
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
			log.Debug(ti, "the r.method ", r.Method, "create token", context.Token)
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
		mode := template.HTMLEscapeString(r.Form.Get("Mode"))
		mesgType := template.HTMLEscapeString(r.Form.Get("MessageType"))
		context.Token = formToken
		n := strings.Split(r.RemoteAddr, ":")[0] + "-" + strings.TrimLeft(strings.Fields(r.UserAgent())[1], "(")
		uname := strings.TrimRight(n, ";")
		bodyin := template.HTMLEscapeString(r.Form.Get("bodyin"))
		cookie, e := r.Cookie("csrftoken")
		if e != nil {
			log.Warn(e)
			context.Returncode = "cookie read error" + e.Error()
			goto SHOW
		}
		context.Binstr, e = CheckAndFilterDataInput(bodyin)
		if e != nil || context.Binstr == nil {
			log.Warn(e)
			context.Returncode = e.Error() + "or nil data"
			goto SHOW
		}
		log.Infof("%s %s %s  with cookie token %s and form token %s, Mode:%s,Type:%s\n",
			ti, uname, r.Method, cookie.Value, context.Token, mode, mesgType)
		summary = ti + "|" + r.Method + "|" + mode + "|" + mesgType
		log.Info("indata :\n", bodyin)
		context.Encode = hex.EncodeToString(context.Binstr)
		if formToken == cookie.Value {
			context.Returncode = "Get EqualToken done"
			file, header, e := r.FormFile("uploadfile")
			if e != nil {
				log.Warn(e)
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
							log.Warn(e)
							context.Returncode = "Can't create the dir!"
							//return //TODO: should we return or go to the html show place?
							goto SHOW
						}
					}
				}
				context.Returncode = "create the dir done"
				upload := "./runcmd/" + formToken + "/" + ProtoFile
				_, e := os.Stat(upload)
				if e == nil {
					log.Debug("upload file already exist, rm it first...")
					shell := "rm -fr " + upload
					log.Debug("run cmd", shell)
					cmd := exec.Command("sh", "-c", shell)
					_, e := cmd.CombinedOutput()
					if e != nil {
						log.Warn(e)
						context.Returncode = "Can't remove the file already exist!"
						goto SHOW
					}
				}

				f, e := os.OpenFile(upload, os.O_WRONLY|os.O_CREATE, 0666)
				if e != nil {
					log.Warn(e)
					context.Returncode = "Can't create the file!"
					goto SHOW
				}
				defer f.Close()
				io.Copy(f, file)
				context.Returncode = "upload file done"

				//run cmd for what you want
				if mode == "Normal" {
					if mesgType != "" {
						output, e := ParseGpbNormalMode(context.Binstr, mesgType, upload)
						if e != nil {
							log.Warn(e)
							context.Decode = e.Error()
							context.Returncode = fmt.Sprintf("ParseNormalMode Error:%s", e.Error())
						} else {
							context.Decode = fmt.Sprintf("%s", output)
							context.Returncode = "Successfully Parse Normal mode done!"
							summary += "Succ"
						}
					} else {
						context.Returncode = "Error! NormalMode Must fill the messagetype"
					}
				} else if mode == "HardCore" {
					output, e := HardcoreDecode(upload, context.Binstr)
					if e != nil {
						log.Warn(e)
						context.Decode = e.Error()
						context.Returncode = fmt.Sprintf("HardCoreMode Error:%s", e.Error())
					} else {
						context.Decode = fmt.Sprintf("%s", output)
						context.Returncode = "Successfully Parse HardCore mode done!"
						summary += "Succ"

					}
				} else {
					log.Warn("Unknow parse mode")
					context.Returncode = "Unknown parse mode!"
				}

			} else {
				context.Returncode = "Can't read the src file!"
				log.Warn("Can't create the data source file, maybe nil or empty upload filename")
			}
		} else {
			log.Warn("form token mismatch")
			context.Returncode = "form token mismatch"
		}
	SHOW:
		dstore.SendData(uname, summary+"|"+context.Returncode)
		b, e := template.ParseFiles("./templates/datapost.html")
		if e != nil {
			log.Warn(e)
			http.Error(w, e.Error(), http.StatusInternalServerError)
			return
		}
		log.Infof("Encode:\n%s", context.Encode)
		log.Infof("Decode:\n%s", context.Decode)
		log.Infof("Returncode:\n%s", context.Returncode)
		e = b.Execute(w, context)
		if e != nil {
			log.Warn(e)
			http.Error(w, e.Error(), http.StatusInternalServerError)
			return
		}
		//http.Redirect(w, r, "/", 302)
	} else {
		log.Warn("Unknown request")
		http.Redirect(w, r, "/", 302)
	}

}

func ParseGpbNormalMode(data []byte, message string, proto string) ([]byte, error) {
	pkg, e := filterPkg(proto)
	if e != nil {
		return nil, e
	}
	messageType := pkg + "." + pureCmdStringPlus(message)
	log.Debug("ParseGpbNormalMode Message type:", messageType)
	cmdstr := fmt.Sprintf("echo %x | xxd -r -p | protoc --decode %s %s", data, messageType, proto)
	log.Debugf("ParseGpbNormalMode cmdstr:%s\n", cmdstr)
	output, e := runshell(cmdstr)
	if e != nil {
		return nil, e
	}
	log.Debugf("ParseGpbNormalMode output is:\n%s\n", output)
	return output, nil

}
func pureCmdString(str string) string {
	return strings.Trim(strings.Trim(strings.Trim(strings.Trim(str, "\n"), "\r"), " "), ";")
}

func pureCmdStringPlus(str string) string {
	return strings.Replace(strings.Replace(strings.Replace(strings.Replace(str, "\n", "", -1), "\r", "", -1), " ", "", -1), ";", "", -1)
}
func filterPkg(proto string) (string, error) {
	cmdstr := fmt.Sprintf("awk '$1 == \"package\" {print $2}' %s", proto)
	output, e := runshell(cmdstr)
	if e != nil {
		return "", e
	}
	return pureCmdStringPlus(fmt.Sprintf("%s", output)), nil

}
func filterMessageTypes(proto string) ([]string, error) {
	cmdstr := fmt.Sprintf("awk '$1 == \"message\" {print $2}' %s", proto)
	output, e := runshell(cmdstr)
	if e != nil {
		return nil, e
	}
	messages := strings.Split(fmt.Sprintf("%s", output), "\n")
	log.Debugf("before filter return %s\n", messages)
	for i := 0; i < len(messages); {
		if messages[i] == "\n" {
			messages = append(messages[:i], messages[i+1:]...)
		} else {
			i++
		}
	}
	log.Debugf("After filter return %s\n", messages)
	return messages, nil
}

func HardcoreDecode(proto string, data []byte) ([]byte, error) {
	var pkgMesg, cmdstr string
	pkg, e := filterPkg(proto)
	if e != nil {
		return nil, e
	}
	types, e := filterMessageTypes(proto)
	if e != nil {
		return nil, e
	}
	for i := 0; i < EscapeBytesMax; i++ {
		log.Debugf("HardcoreDecode Index=%d, data=%x\n", i, data[i:])
		for k, message := range types {
			pkgMesg = pkg + "." + message
			log.Debugf("decode the %v type %s\n", k, pkgMesg)

			cmdstr = fmt.Sprintf("echo %x | xxd -r -p | protoc --decode %s %s", data[i:], pkgMesg, proto)
			log.Debug("cmd =", cmdstr)
			cmd := exec.Command("sh", "-c", cmdstr)
			output, err := cmd.CombinedOutput()
			if err != nil || JudgeHardcoreDecodeResult(output) {
				log.Debug("DecodeFail on messageType ", pkgMesg, " continue...")
				continue
			} else {
				return []byte("HardcoreDecode Type->" + pkgMesg + ":\n" + string(output)), nil
			}
		}
	}
	//finally give a raw decode
	cmdstr = fmt.Sprintf("echo %x | xxd -r -p | protoc --decode_raw", data)
	return runshell(cmdstr)
}

func JudgeHardcoreDecodeResult(result []byte) bool {
	data := string(result)
	datas := strings.Split(data, "\n")
	re := regexp.MustCompile("^[0-9]*:{1} ")
	for k, v := range datas {
		log.Debug(k, "line:", v)
		if re.MatchString(v) {
			log.Debug("JudgeHardcoreDecodeResult return true... on line", k, ":", v)
			return true
		}
	}
	return false
}
func pureHtmlDataIn(in string) string {
	return strings.Replace(strings.Replace(
		strings.Replace(strings.Replace(in, "\n", "", -1), "\r",
			"", -1), "0x", "", -1), " ", "", -1)
}

func runshell(shell string) ([]byte, error) {
	cmd := exec.Command("sh", "-c", shell)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}
	return output, nil
}
func CheckAndFilterDataInput(data string) ([]byte, error) {
	if strings.Contains(data, "[") && strings.Contains(data, "]") {
		log.Info("[1] = 65 type data")
		return ConvertDecToHexDataString(data)
	} else {
		log.Info("hex 08aebf type data")
		log.Debug("PureDataIn :", pureHtmlDataIn(data))

		return hex.DecodeString(pureHtmlDataIn(data))
	}
	return nil, errors.New("Errors in check the data")
}
func ConvertDecToHexDataString(data string) ([]byte, error) {
	re := regexp.MustCompile("\\[{1}[0-9]*]{1} ={1} {1}")
	str := re.ReplaceAllString(data, "")
	log.Debug("Filter data=", str)
	str = strings.TrimSpace(strings.Replace(strings.Replace(strings.Replace(str, "\n", "", -1), "\r", "", -1), ",", "", -1))
	s := strings.Split(str, " ")
	log.Debug("Filter s=", s)
	b := make([]byte, len(s))
	for i, v := range s {
		//fmt.Printf("%d=%s\n", i, v)
		t, err := strconv.Atoi(v)
		if err != nil {
			return nil, err
		}
		b[i] = byte(t)
	}
	log.Debugf("Fmt b=%x\n", b)
	return b, nil
}
func FilterDecDataString(data string) string {
	re := regexp.MustCompile("\\[{1}[0-9]*]{1}={1}")
	str := re.ReplaceAllString(pureHtmlDataIn(data), "")
	log.Debug("Filter data=", str)
	return strings.Replace(strings.Replace(str, ",", "", -1), " ", "", -1)
}
func main() {
	port := flag.String("port", "8091", "Server Port")
	flag.Parse()

	go dstore.Run()
	http.HandleFunc("/", PostDataHandler)
	http.Handle("/templates/", http.StripPrefix("/templates/", http.FileServer(http.Dir("./templates"))))
	http.Handle("/runcmd/", http.StripPrefix("/runcmd/", http.FileServer(http.Dir("./runcmd"))))
	log.Infof("Running the server on port %q.", *port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", *port), nil))
}
