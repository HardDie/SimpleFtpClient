package main

import (
	"crypto/md5"
	"errors"
	"fmt"
	"github.com/jlaffaye/ftp"
	"github.com/mitchellh/ioprogress"
	"io"
	"io/ioutil"
	"log"
	"os"
	"time"
)

var logEnabled bool = false

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func connectToFtp(server, user, pass string) (*ftp.ServerConn, error) {
	ftpClient, err := ftp.Dial(server, ftp.DialWithTimeout(5*time.Second))
	if err != nil {
		return nil, errors.New("can't connect to server")
	}

	if err := ftpClient.Login(user, pass); err != nil {
		return nil, errors.New("can't auth on server")
	}

	return ftpClient, nil
}

func downloadFile(ftpClient *ftp.ServerConn, filename string, size uint64) error {
	// Read data from ftp
	data, err := ftpClient.Retr(filename)
	if err != nil {
		return errors.New("can't download file")
	}
	defer data.Close()

	// Progress bar
	myDraw := func(a, b int64) string {
		size := getTTYSize()
		progress := ioprogress.DrawTextFormatBytes(a, b)

		bar_len := int(size.Col) - len(filename) - len(progress) - 3

		bar := ioprogress.DrawTextFormatBar(int64(bar_len))
		return fmt.Sprintf("%s: %s %s\n", filename, bar(a, b), progress)
	}
	progressR := &ioprogress.Reader{
		Reader:   data,
		Size:     int64(size),
		DrawFunc: ioprogress.DrawTerminalf(os.Stdout, myDraw),
	}

	// Write to file
	outFile, err := os.Create(filename)
	if err != nil {
		return errors.New("can't create local file")
	}
	defer outFile.Close()
	if _, err := io.Copy(outFile, progressR); err != nil {
		return errors.New("can't copy downloaded data to file")
	}

	return nil
}

func deleteFile(ftpClient *ftp.ServerConn, filename string) error {
	if err := ftpClient.Delete(filename); err != nil {
		return errors.New("can't delete file")
	}

	return nil
}

func waitForFile(ftpClient *ftp.ServerConn, filename string) (size uint64, err error) {
loop:
	for {
		entries, err := ftpClient.List("/")
		if err != nil {
			return 0, errors.New("can't get list of files")
		}
		for _, element := range entries {
			if element.Name == filename {
				size = element.Size
				break loop
			}
		}
		time.Sleep(1 * time.Second)
	}
	return size, nil
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

func calcMD5(filename string) string {
	dat, err := ioutil.ReadFile(filename)
	if err != nil {
		return ""
	}

	return fmt.Sprintf("%x", md5.Sum(dat))
}

func main() {
	/**
	 * Read variables
	 */
	homeDir := os.Getenv("HOME")
	if len(homeDir) == 0 {
		log.Fatal("$HOME variable is empty")
	}

	server, err := readDataFromFile(homeDir + "/.ftp/server")
	check(err)
	login, err := readDataFromFile(homeDir + "/.ftp/login")
	check(err)
	pass, err := readDataFromFile(homeDir + "/.ftp/pass")
	check(err)

	var channels []chan bool

//	if len(os.Args) == 2 {
		logEnabled = true
//	}

	for i := 1; i < len(os.Args); i++ {
		loc_chan := make(chan bool)
		channels = append(channels, loc_chan)
		filename := os.Args[i]
		go func() {
			client, err := connectToFtp(server+":21", login, pass)
			if err != nil {
				log.Fatal(err)
			}

			if logEnabled {
				fmt.Printf("%v: Wait...\r", filename)
			}
			size, err := waitForFile(client, filename)
			if err != nil {
				log.Fatal(err)
			}
			if err := downloadFile(client, filename, size); err != nil {
				log.Fatal(err)
			}
			if err := deleteFile(client, filename); err != nil {
				log.Fatal(err)
			}
			fmt.Printf("%s: Done! md5sum = %s\n", filename, calcMD5(filename))

			if err := client.Quit(); err != nil {
				log.Fatal(err)
			}

			loc_chan <- true
		}()
	}

	for _, channel := range channels {
		<-channel
	}
}
