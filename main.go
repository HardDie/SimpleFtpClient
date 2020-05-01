package main

import (
	"bufio"
	"crypto/md5"
	"errors"
	"fmt"
	"github.com/jlaffaye/ftp"
	"github.com/mitchellh/ioprogress"
	"io"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strconv"
	"time"
)

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func connectToFtp() (*ftp.ServerConn, error) {
	config, err := readConfig()
	check(err)

	ftpClient, err := ftp.Dial(config.Server+":21", ftp.DialWithTimeout(5*time.Second))
	if err != nil {
		return nil, errors.New("can't connect to server")
	}

	if err := ftpClient.Login(config.Login, config.Pass); err != nil {
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

		bar_len := int(size.Col) - len(filename) - len(progress) - 14

		bar := newProgressBar(int64(bar_len))
		return fmt.Sprintf("%s: %s %s", filename, bar(a, b), progress)
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

func calcMD5(filename string) string {
	dat, err := ioutil.ReadFile(filename)
	if err != nil {
		return ""
	}

	return fmt.Sprintf("%x", md5.Sum(dat))
}

func byteUnitStr(n uint64) string {
	var byteUnits = []string{"B", "KB", "MB", "GB", "TB", "PB"}

	var unit string
	size := float64(n)
	for i := 1; i < len(byteUnits); i++ {
		if size < 1000 {
			unit = byteUnits[i-1]
			break
		}

		size = size / 1000
	}

	return fmt.Sprintf("%.3g %s", size, unit)
}

func printListFiles(ftpClient *ftp.ServerConn) ([]*ftp.Entry, error) {
	entries, err := ftpClient.List("/")
	if err != nil {
		return nil, errors.New("Can't get list of files")
	}

	if len(entries) == 0 {
		fmt.Println("FTP server is empty!")
		return nil, nil
	}

	sort.SliceStable(entries, func(i, j int) bool {
		return (entries[i].Time.UnixNano()) < (entries[j].Time.UnixNano())
	})
	for i, entry := range entries {
		fmt.Printf("%d) %v %8s - %s\n", i, entry.Time.Format("2006-01-02 15:04:05"),
			byteUnitStr(entry.Size), entry.Name)
	}
	return entries, nil
}

type mode_t int

const (
	MODE_HELP mode_t = iota
	MODE_LIST
	MODE_DOWNLOAD
	MODE_DELETE_ALL
	MODE_DOWNLOAD_BY_INDEX
)

func main() {
	mode := MODE_DOWNLOAD_BY_INDEX

	var fileslist []string
	f_removeOnly := false

	/**
	 * Parse arguments
	 */
	for _, token := range os.Args[1:] {
		if token == "-d" || token == "--delete" {
			f_removeOnly = true
			continue
		} else if token == "-l" || token == "--list" {
			mode = MODE_LIST
			break
		} else if token == "-h" || token == "--help" {
			mode = MODE_HELP
			break
		} else if token == "--delete-all" {
			mode = MODE_DELETE_ALL
			break
		}
		mode = MODE_DOWNLOAD
		fileslist = append(fileslist, token)
	}

	switch mode {
	case MODE_HELP:
		fmt.Println("Usage:", os.Args[0], "[-h]", "[-l]", "[-d]", "[FILES]...")
		fmt.Println()
		fmt.Println("optional arguments:")
		fmt.Println("  -h, --help       show this help message and exit")
		fmt.Println("  -l, --list       show available files on the server and exit")
		fmt.Println("  -d, --delete     delete files without downloading")
		fmt.Println("  --delete-all     delete all available files on server")
		return
	}

	/**
	 * Connect to server
	 */
	client, err := connectToFtp()
	check(err)

	switch mode {
	case MODE_LIST:
		_, err := printListFiles(client)
		check(err)
		return
	case MODE_DELETE_ALL:
		entries, err := printListFiles(client)
		check(err)
		if entries == nil {
			return
		}

		fmt.Println()
		fmt.Println("Are you sure you are want delete all this files? [Type: YES]:")

		reader := bufio.NewReader(os.Stdin)
		text, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalf("Can't read user input")
		}

		if text != "YES\n" {
			fmt.Println("Operation canceled")
			return
		}

		for _, entry := range entries {
			if err = deleteFile(client, entry.Name); err != nil {
				fmt.Printf("%s: Skiped!\n", entry.Name)
			} else {
				fmt.Printf("%s: Deleted!\n", entry.Name)
			}
		}
		return
	case MODE_DOWNLOAD:
		for _, file := range fileslist {
			fmt.Printf("%v: Wait...\r", file)
			size, err := waitForFile(client, file)
			check(err)

			if !f_removeOnly {
				err = downloadFile(client, file, size)
				check(err)
			}

			err = deleteFile(client, file)
			check(err)

			if !f_removeOnly {
				fmt.Printf("%s: Done! md5sum = %s\n", file, calcMD5(file))
			} else {
				fmt.Printf("%s: Deleted!\n", file)
			}
		}
	case MODE_DOWNLOAD_BY_INDEX:
		entries, err := printListFiles(client)
		check(err)
		if entries == nil {
			return
		}

		fmt.Println()
		fmt.Println("Choose file:")

		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		if err != nil {
			log.Fatalf("Can't read user input")
		}

		val, err := strconv.Atoi(scanner.Text())
		check(err)

		if val < 0 || val >= len(entries) {
			log.Fatalf("Wrong value!")
		}

		file := entries[val].Name
		size := entries[val].Size
		if !f_removeOnly {
			err = downloadFile(client, file, size)
			check(err)
		}

		err = deleteFile(client, file)
		check(err)

		if !f_removeOnly {
			fmt.Printf("%s: Done! md5sum = %s\n", file, calcMD5(file))
		} else {
			fmt.Printf("%s: Deleted!\n", file)
		}
	}

	/**
	 * Quit
	 */
	err = client.Quit()
	check(err)
}
