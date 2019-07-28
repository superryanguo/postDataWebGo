package main

import (
	"fmt"
	"gpbdecoder/postDataWebGo/test"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golang/protobuf/proto"
)

func TestPostDataHandler(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(PostDataHandler))
	defer ts.Close()
	req, err := http.NewRequest("GET", ts.URL, nil)
	if err != nil {
		t.Errorf("Error occured while constructing request: %s", err)
	}

	w := httptest.NewRecorder()
	PostDataHandler(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Actual status: (%d); Expected status:(%d)", w.Code, http.StatusOK)
	}
}
func TestMarshalRight(t *testing.T) {

	u := myobject.User{
		Id:    proto.Int64(1),
		Name:  proto.String("Ryan"),
		Email: proto.String("999dingguagua@hotml.com"),
	}
	data, err := proto.Marshal(&u)
	if err != nil {
		t.Fatal("marshaling error: ", err)
	}

	ur := &myobject.User{}
	err = proto.Unmarshal(data, ur)
	if err != nil {
		t.Fatal("unmarshaling error: ", err)
	}
	//a better way to read the data member?
	if *u.Id != *ur.Id || *u.Name != *ur.Name || *u.Email != *ur.Email {
		t.Error("Unmatch data found")
	}
}
func TestParseGpbNormalMode(t *testing.T) {
	u := myobject.User{
		Id:    proto.Int64(1),
		Name:  proto.String("Ryan"),
		Email: proto.String("999dingguagua@hotml.com"),
	}
	data, err := proto.Marshal(&u)
	if err != nil {
		t.Fatal("marshaling error: ", err)
	}
	p, err := ParseGpbNormalMode(data, "User", "./test/myobject.proto")
	if err != nil {
		t.Error(err.Error())
	} else {
		out := fmt.Sprintf("%s", p)
		if !strings.Contains(out, "Ryan") || !strings.Contains(out, "999dingguagua@hotml.com") {
			t.Error("Data parse fail")
		}
	}
}
func TestHardcoreDecode(t *testing.T) {
	u := myobject.User{
		Id:    proto.Int64(1),
		Name:  proto.String("Ryan"),
		Email: proto.String("999dingguagua@hotml.com"),
	}
	data, err := proto.Marshal(&u)
	if err != nil {
		t.Fatal("marshaling error: ", err)
	}
	p, err := HardcoreDecode("./test/myobject.proto", data)
	if err != nil {
		t.Error(err.Error())
	} else {
		out := fmt.Sprintf("%s", p)
		if !strings.Contains(out, "Ryan") || !strings.Contains(out, "999dingguagua@hotml.com") {
			t.Error("Data parse fail")
		}
	}
}
