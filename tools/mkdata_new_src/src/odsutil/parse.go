package odsutil

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
)

type OdsSheet interface {
	Name() string
	ValueAt(row int, col int) string
}

type OdsFile interface {
	GetSheet(index int) OdsSheet
	GetSheetByName(name string) OdsSheet
}

type documentContent struct {
	XMLName xml.Name `xml:"document-content"`
	Body    struct {
		Spreadsheet struct {
			Tables []table `xml:"table"`
		} `xml:"spreadsheet"`
	} `xml:"body"`
}

type table struct {
	Name string     `xml:"name,attr"`
	Rows []tableRow `xml:"table-row"`
}

type tableRow struct {
	Cells []tableCell `xml:"table-cell"`
}

type tableCell struct {
	Text            string `xml:"p"`
	ColumnsRepeated uint   `xml:"number-columns-repeated,attr"`
}

type sheetImpl struct {
	name string
	data table
}

func (s *sheetImpl) Name() string {
	return s.name
}

func (s *sheetImpl) ValueAt(row int, col int) string {
	if row < 0 || row >= len(s.data.Rows) {
		return ""
	}
	r := s.data.Rows[row]

	// Dynamically calculate the actual column index to account for ODF compression.
	currentCol := 0
	for _, cell := range r.Cells {
		repeat := int(cell.ColumnsRepeated)
		if repeat == 0 {
			repeat = 1 // Default to 1 if the attribute is missing
		}

		if col >= currentCol && col < currentCol+repeat {
			return cell.Text
		}
		currentCol += repeat

		// Short-circuit if we've iterated past the requested column
		if currentCol > col {
			break
		}
	}
	return ""
}

type fileImpl struct {
	sheets []sheetImpl
}

func (f *fileImpl) GetSheet(index int) OdsSheet {
	if index < 0 || index >= len(f.sheets) {
		return nil
	}
	return &f.sheets[index]
}

func (f *fileImpl) GetSheetByName(name string) OdsSheet {
	for i := range f.sheets {
		if f.sheets[i].name == name {
			return &f.sheets[i]
		}
	}
	return nil
}

// ParseOds utilizes io.ReaderAt and size (e.g., from an *os.File) to fulfill
// archive/zip constraints without memory-buffering the entire file.
func ParseOds(r io.ReaderAt, size int64) (OdsFile, error) {
	zipReader, err := zip.NewReader(r, size)
	if err != nil {
		return nil, fmt.Errorf("failed to open zip: %s", err.Error())
	}

	var contentFile *zip.File
	for _, f := range zipReader.File {
		if f.Name == "content.xml" {
			contentFile = f
			break
		}
	}

	if contentFile == nil {
		return nil, fmt.Errorf("content.xml not found in archive")
	}

	rc, err := contentFile.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open content.xml: %s", err.Error())
	}
	defer rc.Close()

	var content documentContent
	if err := xml.NewDecoder(rc).Decode(&content); err != nil {
		return nil, fmt.Errorf("failed to parse XML: %s", err.Error())
	}

	file := &fileImpl{}
	for _, t := range content.Body.Spreadsheet.Tables {
		file.sheets = append(file.sheets, sheetImpl{
			name: t.Name,
			data: t,
		})
	}

	return file, nil
}
