package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
)

type config_t struct {
	Server string
	Login  string
	Pass   string
}

func configExample() {
	config := config_t{
		Server: "8.8.8.8",
		Login:  "user",
		Pass:   "password",
	}

	data, err := json.MarshalIndent(config, "", "	")
	if err != nil {
		panic(err)
	}

	fmt.Println("config.json example:")
	fmt.Println(string(data))
}

func readDataFromFile(filename string) (string, error) {
	dat, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}

	str := string(dat)
	str = str[0 : len(str)-1]

	return str, nil
}

func readFromHomeDirectory() (*config_t, error) {
	config := config_t{}

	homeDir := os.Getenv("HOME")
	if len(homeDir) == 0 {
		return nil, errors.New("$HOME variable is empty")
	}

	if val, err := readDataFromFile(homeDir + "/.ftp/server"); err != nil {
		return nil, errors.New("can't read file ~/.ftp/server")
	} else {
		config.Server = val
	}

	if val, err := readDataFromFile(homeDir + "/.ftp/login"); err != nil {
		return nil, errors.New("can't read file ~/.ftp/login")
	} else {
		config.Login = val
	}

	if val, err := readDataFromFile(homeDir + "/.ftp/pass"); err != nil {
		return nil, errors.New("can't read file ~/.ftp/pass")
	} else {
		config.Pass = val
	}

	return &config, nil
}

func readFromJson() (*config_t, error) {
	config := config_t{}

	data, err := ioutil.ReadFile("config.json")
	if err != nil {
		configExample()
		return nil, errors.New("Can't read data fom config.json")
	}

	if err := json.Unmarshal(data, config); err != nil {
		configExample()
		return nil, errors.New("Can't parse json config")
	}

	return &config, nil
}

func readConfig() (*config_t, error) {
	f_ReadFromJson := true
	if _, err := os.Stat("config.json"); os.IsNotExist(err) {
		f_ReadFromJson = false
	}

	if f_ReadFromJson {
		return readFromJson()
	}
	return readFromHomeDirectory()
}
