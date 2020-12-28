package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type test interface {
	hello()
}

type test1 interface {
	hello1()
}

type struct2 struct{

}

func (s *struct2) hello1() {

}

type struct1 struct {
	f  string
	t1 test1
}

func (s *struct1) hello() {
	fmt.Printf("hello")
}

func func1() *struct2 {
	s := struct1{f:"test", t1:&struct2{}}
	if t1, ok := s.t1.(*struct2); ok {
		return t1
	}
	return nil
}

func main() {
//	json1 := `{
//  "Identifier":"b7cb44c8-4687-11eb-9ba2-f45c89b00d3d",
//  "Sequence":1,
//  "CType":2,
//  "Payload":{
//    "Schema":"",
//    "Method":"GET",
//    "Host":"",
//    "Port":9081,
//    "BasePath":"/",
//    "Path":"streams",
//    "Headers": {
//      "Connection":["keep-alive"],
//      "Content-Length":["0"],
//      "Content-Type":["text/plain; charset=ISO-8859-1"],
//      "User-Agent":["Apache-HttpClient/4.5.5 (Java/1.8.0_202)"]
//    },
//    "Body":""
//  }
//} `
	//cmd := common.HttpCommand{
	//	Command:     common.Command{
	//
	//	},
	//	HttpRequest: common.HttpRequest{},
	//}
	//e := json.Unmarshal([]byte(json1), &cmd)
	//fmt.Printf("%s, %v", e, cmd)

	//hcmd := common.HttpCommand{
	//	BasicCommand: common.BasicCommand{
	//		Identifier: "11111",
	//		Sequence:   1,
	//		CType:      2,
	//	},
	//	HttpRequest: common.HttpRequest{
	//		Schema:   "http",
	//		Method:   "post",
	//		Host:     "",
	//		Port:     0,
	//		BasePath: "",
	//		Path:     "",
	//		Headers:  nil,
	//		Body:     nil,
	//	},
	//}
	//fmt.Printf("%s\n", hcmd.Json())
	resp, _ := http.Get("http://localhost:9081/streams")
	fmt.Printf("%#v\n", *resp)

	respJ, _ := json.Marshal(resp)
	fmt.Println("---")
	fmt.Println(respJ)
}
