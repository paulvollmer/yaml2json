package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	yaml "gopkg.in/yaml.v2"
)

// LoadHTTPTimeout the default timeout for load requests
const LoadHTTPTimeout = 30 * time.Second

var (
	output string
)

func init() {
	flag.StringVar(&output, "output", "", "Write to the file instead of to stdout")
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s [YAML FILE OR URL]\n\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "you need to provide the file path or url to load")
		os.Exit(1)
	}

	json, err := YAMLDoc(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	if output == "" {
		fmt.Println(string(json))
	} else {
		err := ioutil.WriteFile(output, json, 0600) // umask settings will be respected this way
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
	}
}

// LoadFromFileOrHTTP loads the bytes from a file or a remote http server based on the path passed in
func LoadFromFileOrHTTP(path string) ([]byte, error) {
	return LoadStrategy(path, ioutil.ReadFile, loadHTTPBytes(LoadHTTPTimeout))(path)
}

// LoadStrategy returns a loader function for a given path or uri
func LoadStrategy(path string, local, remote func(string) ([]byte, error)) func(string) ([]byte, error) {
	if strings.HasPrefix(path, "http") {
		return remote
	}
	return local
}

func loadHTTPBytes(timeout time.Duration) func(path string) ([]byte, error) {
	return func(path string) ([]byte, error) {
		client := &http.Client{Timeout: timeout}
		req, err := http.NewRequest("GET", path, nil)
		if err != nil {
			return nil, err
		}
		resp, err := client.Do(req)
		defer func() {
			if resp != nil {
				if e := resp.Body.Close(); e != nil {
					log.Println(e)
				}
			}
		}()
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("could not access document at %q [%s] ", path, resp.Status)
		}

		return ioutil.ReadAll(resp.Body)
	}
}

// YAMLToJSON converts YAML unmarshaled data into json compatible data
func YAMLToJSON(data interface{}) (json.RawMessage, error) {
	jm, err := transformData(data)
	if err != nil {
		return nil, err
	}
	b, err := json.Marshal(jm)
	return json.RawMessage(b), err
}

func bytesToYAMLDoc(data []byte) (interface{}, error) {
	var document map[interface{}]interface{}
	if err := yaml.Unmarshal(data, &document); err != nil {
		return nil, err
	}

	return document, nil
}

func transformData(in interface{}) (out interface{}, err error) {
	switch in.(type) {
	case map[interface{}]interface{}:
		o := make(map[string]interface{})
		for k, v := range in.(map[interface{}]interface{}) {
			sk := ""
			switch k.(type) {
			case string:
				sk = k.(string)
			case int:
				sk = strconv.Itoa(k.(int))
			default:
				return nil, fmt.Errorf("types don't match: expect map key string or int get: %T", k)
			}
			v, err = transformData(v)
			if err != nil {
				return nil, err
			}
			o[sk] = v
		}
		return o, nil
	case []interface{}:
		in1 := in.([]interface{})
		len1 := len(in1)
		o := make([]interface{}, len1)
		for i := 0; i < len1; i++ {
			o[i], err = transformData(in1[i])
			if err != nil {
				return nil, err
			}
		}
		return o, nil
	}
	return in, nil
}

// YAMLDoc loads a yaml document from either http or a file and converts it to json
func YAMLDoc(path string) (json.RawMessage, error) {
	yamlDoc, err := YAMLData(path)
	if err != nil {
		return nil, err
	}

	data, err := YAMLToJSON(yamlDoc)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// YAMLData loads a yaml document from either http or a file
func YAMLData(path string) (interface{}, error) {
	data, err := LoadFromFileOrHTTP(path)
	if err != nil {
		return nil, err
	}

	return bytesToYAMLDoc(data)
}