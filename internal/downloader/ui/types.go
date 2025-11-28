package ui

type DownloadProgress struct {
	Percentage float64
	Downloaded string
	Speed      string
	ETA        string
	Status     string
}

type UploadProgress struct {
	Percentage float64
	Uploaded   string
	TotalSize  string
	Speed      string
}
