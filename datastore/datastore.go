package datastore

import (
	"encoding/gob"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
)

//TODO:
//1.multi-thread access the same data issue?
//2.datamodel:user-TestData:Time,Token,Mode,Type,ReturnCode
//3.Use json?
//4.ring-buffer size file to keep the data
type MesgData struct {
	Name    string
	Summary string
}

type AccRecd struct {
	RecData map[string][]string
}

func (a *AccRecd) Save(f string) error {

	//_, e := os.Stat(f)
	//if e == nil {
	////if !os.IsNotExist(e) {
	////if file exist, then remove it first
	//log.Debug("Datafile already exist...")
	//shell := "rm -fr " + f
	//log.Debug("run cmd", shell)
	//cmd := exec.Command("sh", "-c", shell)
	//output, err := cmd.CombinedOutput()
	//if err != nil {
	//log.Warn("File can't remove to save new file!")
	//return err
	//}
	//log.Debugf("Rm done, %s\n", output)
	////}
	//}

	log.Debug("Saving to file...")
	file, err := os.OpenFile(f, os.O_RDWR|os.O_CREATE, 0777)
	defer file.Close()
	if err != nil {
		log.Debug("Open file error when saving to file...")
		return err
	}
	err = gob.NewEncoder(file).Encode(a.RecData)
	if err != nil {
		return err
	} else {
		return nil
	}

}
func (a *AccRecd) ShowData() {
	for k, v := range a.RecData {
		for i, s := range v {
			log.Infof("|%s|%d|%s|\n", k, i, s)
		}

	}
}
func SendData(name string, sum string) {
	msg := MesgData{Name: name, Summary: sum}
	if DataChan != nil {
		DataChan <- msg
		log.Debug("sending data to datastore...\n", msg)
	}

}
func (a *AccRecd) ReadFile(f string) error {
	_, e := os.Stat(f)
	if e != nil {
		if os.IsNotExist(e) {
			log.Debugf("%s not exist, create one\n", f)
			fl, err := os.OpenFile(f, os.O_WRONLY|os.O_CREATE, 0666)
			defer fl.Close()
			if err != nil {
				return err
			}
			return nil
		} else { //shouldn't go here
			log.Warn(e)
		}
	} else {
		file, err := os.Open(f)
		defer file.Close()
		if err != nil {
			return err
		}
		err = gob.NewDecoder(file).Decode(&a.RecData)
		if err != nil {
			return err
		} else {
			return nil
		}
	}
	return nil
}

func (a *AccRecd) Receive(m MesgData) {
	value, ok := a.RecData[m.Name]
	if ok {
		value = append(value, m.Summary)
		a.RecData[m.Name] = value
		log.Debugf("Add data %v\n", m)
	} else {
		s := make([]string, 1)
		s[0] = m.Summary
		a.RecData[m.Name] = s
		log.Debugf("Create data %v\n", m)
	}

}

var DataLib AccRecd
var DataFile string = "./datastore/data.gb"
var DataChan chan MesgData

func init() {
	//log.SetOutput(os.Stdout)
	//log.SetLevel(log.InfoLevel)
	//log.SetLevel(log.DebugLevel)
	DataChan = make(chan MesgData)
	DataLib.RecData = make(map[string][]string)
}

func Run() {
	log.Info("KickOff the DataStore")
	err := DataLib.ReadFile(DataFile)
	if err != nil {
		log.Warn(err)
		return //slient go back, don't fatal
	}
	log.Info("DataStore finish readfile...")

	//go func() {
	//DataChan <- MesgData{Name: "Ryan", Summary: "Hello"}
	//DataChan <- MesgData{Name: "Guo", Summary: "World"}
	//log.Debug("sending data done")
	//close(DataChan)
	//}()
	//log.Debug("Launch the message sender done!")

	var more bool = true
	var msg MesgData
	for more {
		select {
		case msg, more = <-DataChan:
			if more {
				DataLib.Receive(msg)
				log.Debugf("Receive one message %v\n", msg)
			} else {
				log.Debug("DataChan closed!")
			}
		case <-time.After(time.Second * 20):
			log.Debug("Time Out, save file...")
			err := DataLib.Save(DataFile)
			if err != nil {
				log.Warn(err)
			}
		}
	}
	err = DataLib.Save(DataFile)
	if err != nil {
		log.Warn(err)
		return
	}
}
