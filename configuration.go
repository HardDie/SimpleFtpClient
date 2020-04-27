package main

import (
	"errors"
	"io/ioutil"
	"os"
)

type config_t struct {
	Server string
	Login  string
	Pass   string
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

func readConfig() (*config_t, error) {
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
