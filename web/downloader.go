package web

import (
	"fmt"
	"io"
	"net/http"
	"os"
	//	"time"
)

type Status int

const (
	InProgress Status = iota
	Finished   Status = iota
	NotStarted Status = iota
	Failed     Status = iota
)

type Downloader struct {
	downloadedDir string
	incompleteDir string
	statChan      chan DownloadStat
}

type DownloadStat struct {
	Status      Status
	Error       error
	Progress    int
	Size        int64
	Downloaded  int64
	OutFilename string
	Url         string
}

func NewDownloader(downloadedDir string, incompleteDir string) *Downloader {
	return &Downloader{
		downloadedDir: downloadedDir,
		incompleteDir: incompleteDir,
	}
}

func (d *Downloader) All(lastN int) {

}

func (d *Downloader) Download(url string, outFilename string) {

}

func (d *Downloader) Stat(filename string) {

}

func (d *Downloader) eventLoop() {

}

func download(fileUrl string, outFilename string, statChan chan DownloadStat) {
	of, err := os.Create(outFilename)
	if err != nil {
		statChan <- DownloadStat{
			Status:      Failed,
			Error:       err,
			OutFilename: outFilename,
			Url:         fileUrl,
		}
		return
	}
	defer of.Close()

	req, err := http.Get(fileUrl)
	if err != nil {
		statChan <- DownloadStat{
			Status:      Failed,
			Error:       err,
			OutFilename: outFilename,
			Url:         fileUrl,
		}
		return
	}
	if req.StatusCode != 200 {
		statChan <- DownloadStat{
			Status:      Failed,
			Error:       http.ErrMissingFile,
			OutFilename: outFilename,
			Url:         fileUrl,
		}
		return
	}
	defer req.Body.Close()

	var buf [4096]byte
	var downloadedN int64 = 0
	stat := DownloadStat{
		Status:      InProgress,
		Progress:    0,
		Size:        req.ContentLength,
		Downloaded:  downloadedN,
		OutFilename: outFilename,
		Url:         fileUrl,
	}
	for true {
		n, err := req.Body.Read(buf[:])
		downloadedN += int64(n)
		if n > 0 {
			_, err := of.Write(buf[:n])
			if err != nil {
				stat.Error = err
				stat.Status = Failed
				statChan <- stat
				return
			}
			stat.Downloaded = downloadedN
			stat.Progress = int(float32(downloadedN) / float32(req.ContentLength) * 100)
		}
		if err == io.EOF {
			stat.Status = Finished
			stat.Error = nil
			statChan <- stat
			return
		} else if err != nil {
			stat.Status = Failed
			stat.Error = err
			statChan <- stat
			return

		}
		statChan <- stat
	}
}

func Test() {
	//	downloader := NewDownloader("/tmp/", "/tmp/", "/tmp/stat")
	progresses := make(chan DownloadStat)
	go download("http://ipv4.download.thinkbroadband.com/5MB.zip", "asds", progresses)
	for true {
		c := <-progresses
		fmt.Println(c)
	}
}
