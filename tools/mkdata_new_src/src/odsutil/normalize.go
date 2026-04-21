package odsutil

import (
	"archive/zip"
	"io"
	"time"
)


func ResetZipTimestamps(inputFile io.ReaderAt, size int64, outputFile io.Writer) error {
	reader, err := zip.NewReader(inputFile, size)
	if err != nil {
		return err
	}

	writer := zip.NewWriter(outputFile)
	defer writer.Close()

	// 1980-01-01 is the effective "epoch" for ZIP (MS-DOS time)
	epoch := time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)

	for _, file := range reader.File {
		// Prepare the header based on the original file
		header := file.FileHeader
		header.Modified = epoch
		
		// Create the new entry in the output ZIP
		target, err := writer.CreateHeader(&header)
		if err != nil {
			return err
		}

		// Open the original compressed data
		source, err := file.Open()
		if err != nil {
			return err
		}

		// Copy the content bit-for-bit
		_, err = io.Copy(target, source)
		source.Close()
		if err != nil {
			return err
		}
	}

	return nil
}
