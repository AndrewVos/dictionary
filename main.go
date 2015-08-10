package main

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"text/template"
	"time"
)

type Word struct {
	Word           string
	wordDataIndex  int64
	wordDataLength int64
}

type Dictionary struct {
	dictionaryPath string
	words          []Word
}

func (d *Dictionary) FindRandomWords() ([]string, error) {
	reader, err := reader(d.dictionaryPath)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	rand.Seed(time.Now().UTC().UnixNano())

	var result []string
	for i := 0; i < 10; i++ {
		r := d.words[rand.Intn(len(d.words))]
		result = append(result, r.Word)
	}
	return result, nil
}

func (d *Dictionary) FindWord(word string) (string, error) {
	word = strings.ToLower(word)
	var result []string
	reader, err := reader(d.dictionaryPath)
	if err != nil {
		return "", err
	}
	defer reader.Close()

	for _, indexedWord := range d.words {
		if strings.ToLower(indexedWord.Word) == word {
			reader.Seek(indexedWord.wordDataIndex, 0)
			b := make([]byte, indexedWord.wordDataLength)
			_, err := reader.Read(b)
			if err != nil {
				return "", err
			}
			result = append(result, string(b))
		}
	}
	return strings.Join(result, "\n"), nil
}

func NewDictionary(indexPath string, dictionaryPath string) (*Dictionary, error) {
	dictionary := &Dictionary{
		words:          []Word{},
		dictionaryPath: dictionaryPath,
	}

	indexContent, err := read(indexPath)
	if err != nil {
		return nil, err
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

		word := Word{
			Word:           currentWord,
			wordDataIndex:  int64(offset),
			wordDataLength: int64(size),
		}
		dictionary.words = append(dictionary.words, word)

		if i == len(indexContent) {
			break
		}
	}
	return dictionary, nil
}

func main() {
	dictionary, err := NewDictionary("dictd_www.dict.org_web1913.idx", "dictd_www.dict.org_web1913.dict.dz")
	if err != nil {
		log.Fatal(err)
	}
	if os.Args[1] == "--serve" {
		t, err := template.ParseFiles("index.html")
		if err != nil {
			panic(err)
		}
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			var result string
			var random []string

			word := r.URL.Query().Get("word")

			if word != "" {
				result, err = dictionary.FindWord(word)
				if err != nil {
					w.WriteHeader(500)
					return
				}
			}

			if result == "" {
				result = "Not Found"
				random, err = dictionary.FindRandomWords()
				if err != nil {
					w.WriteHeader(500)
					return
				}
			}
			result = strings.Replace(result, "\n", "<br>", -1)

			t.Execute(w, map[string]interface{}{
				"Result": result,
				"Random": random,
			})
		})

		port := "8080"
		if p := os.Getenv("PORT"); p != "" {
			port = p
		}

		err = http.ListenAndServe(":"+port, nil)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		fmt.Println(dictionary.FindWord(os.Args[1]))
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

func read(path string) ([]byte, error) {
	if strings.HasSuffix(path, ".dz") || strings.HasSuffix(path, ".gz") {
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

	b, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func reader(path string) (*os.File, error) {
	if strings.HasSuffix(path, ".dz") || strings.HasSuffix(path, ".gz") {
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
	return file, nil
}
