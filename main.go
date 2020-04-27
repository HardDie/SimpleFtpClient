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
	"sort"
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

// Methods for sorting entries
type ftpEntry []*ftp.Entry

func (e ftpEntry) Len() int {
	return len(e)
}
func (e ftpEntry) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}
func (e ftpEntry) Less(i, j int) bool {
	return (e[i].Time.UnixNano()) < (e[j].Time.UnixNano())
}

func main() {
	var fileslist []string
	f_removeOnly := false
	f_list := false
	f_help := false

	/**
	 * Parse arguments
	 */
	for _, token := range os.Args[1:] {
		if token == "-d" || token == "--delete" {
			f_removeOnly = true
			continue
		} else if token == "-l" || token == "--list" {
			f_list = true
			continue
		} else if token == "-h" || token == "--help" {
			f_help = true
			break
		}
		fileslist = append(fileslist, token)
	}

	if len(os.Args) == 1 || f_help {
		fmt.Println("Usage:", os.Args[0], "[-h]", "[-l]", "[-d]", "[FILES]...")
		fmt.Println()
		fmt.Println("optional arguments:")
		fmt.Println("  -h, --help       show this help message and exit")
		fmt.Println("  -l, --list       show available files on the server and exit")
		fmt.Println("  -d, --delete     delete files without downloading")
		return
	}

	/**
	 * Connect to server
	 */
	client, err := connectToFtp()
	check(err)

	/**
	 * Show list files
	 */
	if f_list {
		entries, err := client.List("/")
		if err != nil {
			log.Fatalf("Can't get list of files")
		}
		sort.Sort(ftpEntry(entries))
		for _, entry := range entries {
			fmt.Printf("%v %8s - %s\n", entry.Time.Format("2006-01-02 15:04:05"),
				byteUnitStr(entry.Size), entry.Name)
		}
		return
	}

	/**
	 * Download all files one by one
	 */
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

	/**
	 * Quit
	 */
	err = client.Quit()
	check(err)
}
