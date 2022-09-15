package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"rotom/constants"
	"strconv"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v2"
)

var (
	authorization string
	sessionid     string
	version       string
	scheme        string
	host          string
	scanner       *bufio.Scanner
)

type RequestDef struct {
	Name        string              `yaml:"name"`
	URI         string              `yaml:"uri"`
	Method      string              `yaml:"method"`
	Query       []*QueryDef         `yaml:"query"`
	Description string              `yaml:"desc"`
	Body        []*MemberDefWrapper `yaml:"body"`
}

type MemberDefWrapper struct {
	MemberDef      `yaml:",inline"`
	SubTypeNameMap map[string]bool
	JSON           string `yaml:"json"`
	Validate       string `yaml:"validate"`
}

type MemberDef struct {
	Name        string      `yaml:"name"`
	Type        string      `yaml:"type"`
	Value       string      `yaml:"value"`
	StringArray []string    `yaml:"stringArray"`
	Description string      `yaml:"desc,omitempty"`
	Example     interface{} `yaml:"example,omitempty"`
}

type QueryDef struct {
	Key   string `yaml:"key"`
	Value string `yaml:"value"`
}

func getRequestDefinitions() ([]*RequestDef, error) {
	var requestDefs []*RequestDef
	bytes, err := ioutil.ReadFile("./request.yml")
	if err != nil {
		return requestDefs, err
	}

	if err := yaml.Unmarshal(bytes, &requestDefs); err != nil {
		return requestDefs, err
	}

	return requestDefs, nil
}

func init() {
	err := godotenv.Load(".env")
	if err != nil {
		fmt.Println(err)
		return
	}

	version = os.Getenv("X-VERSION")
	scheme = os.Getenv("SCHEME")
	host = os.Getenv("HOST")

	// ログイン（対応セッション）が必要であれば参考
	// scanner = bufio.NewScanner(os.Stdin)
	// fmt.Print("Enter the id of local user you want to test: \n")
	// scanner.Scan()
	// userID := scanner.Text()
	// fmt.Println("\nuserID: ", userID)

	// authorization = userID
	// sessionid = userID

	// uLogin := &url.URL{}
	// uLogin.Scheme = scheme
	// uLogin.Host = host
	// uLogin.Path = "/login"
	// uLoginStr := uLogin.String()

	// reqLogin, _ := http.NewRequest("POST", uLoginStr, nil)
	// reqLogin.Header.Add("x-version", version)
	// reqLogin.Header.Add("Authorization", authorization)
	// reqLogin.Header.Add("x-sessionid", sessionid)

	// client := &http.Client{}
	// _, err = client.Do(reqLogin)
	// if err != nil {
	// 	log.Fatalln(err)
	// }
}

func main() {
	ymlRequests, err := getRequestDefinitions()
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, r := range ymlRequests {
		err := r.sendRequest()
		if err != nil {
			fmt.Println(err)
			return
		}
	}

	fmt.Println("Finished!")
}

func newHttpRequest(httpMethod string, uStr string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(httpMethod, uStr, body)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	req.Header.Add("x-version", version)
	req.Header.Add("Authorization", authorization)
	req.Header.Add("x-sessionid", sessionid)

	return req, nil
}

func saveByteToJson(b []byte, fileName string) error {
	f, err := os.Create("./responseBody/" + fileName + ".json")
	if err != nil {
		return err
	}

	defer f.Close()

	var buf bytes.Buffer

	err = json.Indent(&buf, b, "", "\t")
	if err != nil {
		return err
	}
	writer := bufio.NewWriter(f)
	indentJson := buf.Bytes()
	_, err = writer.Write(indentJson)
	if err != nil {
		return err
	}
	writer.Flush()

	return nil
}

func (r *RequestDef) sendRequest() error {
	u := &url.URL{}
	u.Scheme = scheme
	u.Host = host
	u.Path = r.URI
	uStr := u.String()

	var req *http.Request
	var err error
	switch r.Method {
	case "GET", "get":
		req, err = newHttpRequest(constants.MethodGet, uStr, nil)

	case "POST", "post":
		buf, marshalErr := MarshalBodyByte(r.Body)
		if marshalErr != nil {
			return marshalErr
		}
		req, err = newHttpRequest(constants.MethodPost, uStr, buf)
		fmt.Println(buf.String())

	case "PATCH", "patch":
		buf, marshalErr := MarshalBodyByte(r.Body)
		if marshalErr != nil {
			return marshalErr
		}
		req, err = newHttpRequest(constants.MethodPatch, uStr, buf)
		fmt.Println(buf.String())
	default:
		err = errors.New("invalid method type")
	}
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	if len(r.Query) > 0 {
		for _, query := range r.Query {
			params := req.URL.Query()
			params.Add(query.Key, query.Value)
			req.URL.RawQuery = params.Encode()
		}
	}

	fmt.Println(r.Method)
	fmt.Println(req.URL.String())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return err
	}

	b, err := httputil.DumpResponse(resp, true)
	if err != nil {
		fmt.Println(err)
		return err
	}
	fmt.Println(string(b))

	bodyByte, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return err
	}
	fmt.Println(resp.Status)

	defer resp.Body.Close()
	err = saveByteToJson(bodyByte, r.Name)
	if err != nil {
		fmt.Println(err)
	}
	return nil
}

func MarshalBodyByte(members []*MemberDefWrapper) (*bytes.Buffer, error) {
	var buf bytes.Buffer
	var keyValueBytes []byte
	var err error
	var keyValue interface{}

	buf.Write([]byte("{"))
	for i, m := range members {
		switch m.Type {
		case "bool":
			var value bool
			value, err = strconv.ParseBool(m.Value)
			if err != nil {
				return &buf, err
			}
			keyValue = map[string]bool{
				m.JSON: value,
			}
		case "string":
			keyValue = map[string]string{
				m.JSON: m.Value,
			}
		case "stringArray":
			keyValue = map[string][]string{
				m.JSON: m.StringArray,
			}
		case "number", "uint64", "uint32", "uint8", "uint", "int64", "int32", "int8", "int":
			var value int
			value, err = strconv.Atoi(m.Value)
			if err != nil {
				return &buf, err
			}
			keyValue = map[string]int{
				m.JSON: value,
			}
		}
		keyValueBytes, err = json.Marshal(keyValue)
		if err != nil {
			return &buf, err
		}

		keyValueBytes = bytes.ReplaceAll(keyValueBytes, []byte("{"), []byte(""))
		keyValueBytes = bytes.ReplaceAll(keyValueBytes, []byte("}"), []byte(""))
		if i > 0 {
			buf.Write([]byte{','})
		}
		buf.Write(keyValueBytes)
	}
	buf.Write([]byte("}"))

	return &buf, nil
}
