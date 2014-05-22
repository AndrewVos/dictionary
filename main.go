package main

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

func main() {
	dictionary := "dictd_www.dict.org_web1913.dict.cz"
	index := "dictd_www.dict.org_web1913.idx"

	indexContent, err := read(index, 0, 0)
	if err != nil {
		panic(err)
	}

	i := 0
	for {
		currentWord := ""
		for {
			if indexContent[i] == 0 {
				i += 1
				break
			} else {
				currentWord += string(indexContent[i])
				i += 1
			}
		}

		offset := readInt32(indexContent[i : i+4])
		i += 4

		size := readInt32(indexContent[i : i+4])
		i += 4

		if currentWord == os.Args[1] {
			wordData, err := read(dictionary, offset, size)
			if err != nil {
				panic(err)
			}
			fmt.Println(string(wordData))
			os.Exit(1)
		}
	}
}

func readInt32(b []byte) int {
	var result int32
	buf := bytes.NewReader(b)
	err := binary.Read(buf, binary.BigEndian, &result)
	if err != nil {
		fmt.Println("binary.Read failed:", err)
	}
	return int(result)

}

func decompress(inputPath string, outputPath string) error {
	input, err := os.Open(inputPath)
	if err != nil {
		return err
	}
	defer input.Close()
	reader, err := gzip.NewReader(input)
	if err != nil {
		return err
	}
	defer reader.Close()

	output, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer output.Close()

	_, err = io.Copy(output, reader)
	if err != nil {
		return err
	}

	return nil
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func read(path string, offset int, size int) ([]byte, error) {
	if strings.HasSuffix(path, ".cz") {
		exists, _ := exists(path + ".decompressed")
		if exists == false {
			err := decompress(path, path+".decompressed")
			if err != nil {
				return nil, err
			}
		}
		path = path + ".decompressed"
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	if offset > 0 {
		_, err = file.Seek(int64(offset), 0)
		if err != nil {
			return nil, err
		}
	}

	if size > 0 {
		b := make([]byte, size)
		_, err := file.Read(b)
		if err != nil {
			return nil, err
		}
		return b, nil
	} else {
		b, err := ioutil.ReadAll(file)
		if err != nil {
			return nil, err
		}
		return b, nil
	}
}
