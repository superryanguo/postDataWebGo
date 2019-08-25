package datastore

import (
	"encoding/gob"
	"log"
	"os"
)

//TODO: multi-thread access the same data issue?
//datamodel:user-TestData:Time,Token,Mode,Type,ReturnCode
//Use json?
type MesgData struct {
	Name    string
	Summary string
}

type AccRecd struct {
	RecData map[string][]string
}

func (a *AccRecd) Save(f string) error {
	//filename := "./datastore/data.gb"
	file, err := os.OpenFile(f, os.O_RDWR|os.O_CREATE, 0777)
	if err != nil {
		return err
	}
	defer file.Close()
	err = gob.NewEncoder(file).Encode(a.RecData)
	if err != nil {
		return err
	} else {
		return nil
	}

}
func (a *AccRecd) ReadFile(f string) error {
	file, err := os.Open(f)
	if err != nil {
		return err
	}
	defer file.Close()
	err = gob.NewDecoder(file).Decode(&a.RecData)
	if err != nil {
		return err
	} else {
		return nil
	}
}
func (a *AccRecd) Receive(m MesgData) {
	value, ok := a.RecData[m.Name]
	if ok {
		value = append(value, m.Summary)
		a.RecData[m.Name] = value
		log.Printf("Add data %v\n", m)
	} else {
		s := make([]string, 1)
		s[0] = m.Summary
		a.RecData[m.Name] = s
		log.Printf("Create data %v\n", m)
	}

}

var DataLib AccRecd
var Gbfile string = "./tmp/data.gb"

var DataChan chan MesgData

func init() {
	DataChan = make(chan MesgData)
	DataLib.RecData = make(map[string][]string)
	//DataLib.ReadFile(Gbfile)
}

func main() {
	log.Println("KickOff the DataStore")

	go func() {
		DataChan <- MesgData{Name: "Ryan", Summary: "Hello"}
		DataChan <- MesgData{Name: "Guo", Summary: "World"}
		log.Println("sending data done")
		close(DataChan)
	}()
	log.Println("Launch the message sender done!")
	var more bool = true
	var msg MesgData
	for more {
		log.Println("in the loop...")
		select {
		case msg, more = <-DataChan:
			if more {
				DataLib.Receive(msg)
				log.Println("Receive data")
			} else {
				log.Println("DataChan closed!")
			}
		}
	}
	DataLib.Save(Gbfile)
}
