package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/360EntSecGroup-Skylar/excelize"
)

func main() {

	integrationPath := "/Users/i.wirananta/go/src/github.com/tokopedia/grpc/mitraapp/grpc_testData" // change this
	protos := "/Users/i.wirananta/go/src/github.com/tokopedia/grpc/mitraapp/protos/mitraapp.proto"  // change this
	documentName := "./ITSWEEP.xlsx"                                                                // change this

	xlsx := excelize.NewFile()
	sheet1Name := "Sheet1"
	xlsx.SetSheetName(xlsx.GetSheetName(1), sheet1Name)

	xlsx.SetCellValue(sheet1Name, "A1", "Endpoint")
	xlsx.SetCellValue(sheet1Name, "B1", "Test Case Name")
	xlsx.SetCellValue(sheet1Name, "C1", "File Name")
	xlsx.SetCellValue(sheet1Name, "D1", "Scenario")
	xlsx.SetCellValue(sheet1Name, "E1", "Expected Response")
	xlsx.SetCellValue(sheet1Name, "F1", "Request Param")
	xlsx.SetCellValue(sheet1Name, "G1", "Status")
	xlsx.SetCellValue(sheet1Name, "H1", "Notes")
	xlsx.SetCellValue(sheet1Name, "I1", "PIC")

	protosFile, _ := ioutil.ReadFile(protos)

	err := xlsx.AutoFilter(sheet1Name, "A1", "J1", "")
	if err != nil {
		log.Fatal("ERROR ", err.Error())
	}

	dvRange := excelize.NewDataValidation(true)
	dvRange.Sqref = "G:G"
	dvRange.SetDropList([]string{"Live", "On Progress", "Not Yet", "Pending", "No TestCase", "Not Checked", "Need Fix", "Wont Do", "Endpoint Need Adjustment"})
	xlsx.AddDataValidation(sheet1Name, dvRange)

	mapGrpcList := regex(string(protosFile))
	var total int
	var prevValue string
	err = filepath.Walk(integrationPath,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if path[len(path)-4:] == "json" {
				total++

				jsonFile, err := os.Open(path)
				if err != nil {
					fmt.Println(err)
				}
				defer jsonFile.Close()

				byteValue, _ := ioutil.ReadAll(jsonFile)
				var result IntegrationTest
				content, _ := ioutil.ReadFile(path)
				variables := extractValue(string(content), "variables")

				json.Unmarshal([]byte(byteValue), &result)

				apiName := regexGetBareEndpoint(result.ApiName)
				if _, ok := mapGrpcList[apiName]; ok {
					mapGrpcList[apiName] = false
				}
				xlsx.SetCellValue(sheet1Name, fmt.Sprintf("A%d", total+1), apiName)
				xlsx.SetCellValue(sheet1Name, fmt.Sprintf("B%d", total+1), result.QueryName)
				xlsx.SetCellValue(sheet1Name, fmt.Sprintf("C%d", total+1), filepath.Base(path))
				xlsx.SetCellValue(sheet1Name, fmt.Sprintf("E%d", total+1), result.Structure[0].ResponseCode)
				xlsx.SetCellValue(sheet1Name, fmt.Sprintf("F%d", total+1), variables)
				xlsx.SetCellValue(sheet1Name, fmt.Sprintf("G%d", total+1), "Live")

				if prevValue == apiName {
					xlsx.MergeCell(sheet1Name, "A"+strconv.Itoa(total+1), "A"+strconv.Itoa(total))
				}

				prevValue = apiName

			}
			return nil
		})

	keys := make([]string, 0, len(mapGrpcList))

	for k := range mapGrpcList {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	fmt.Println("Got a total of " + strconv.Itoa(total) + " testcases")
	for _, k := range keys {
		if mapGrpcList[k] {
			total++
			xlsx.SetCellValue(sheet1Name, fmt.Sprintf("A%d", total+1), k)
			xlsx.SetCellValue(sheet1Name, fmt.Sprintf("G%d", total+1), "No TestCase")
		}
	}

	fmt.Println("Scanned a total of " + strconv.Itoa(len(mapGrpcList)) + " endpoint")
	if err != nil {
		log.Println(err)
	}

	err = xlsx.SaveAs(documentName)
	if err != nil {
		fmt.Println(err)
	}
}

type IntegrationTest struct {
	QueryName  string      `json:"queryName"`
	HttpMethod string      `json:"httpMethod"`
	ApiName    string      `json:"apiName"`
	Structure  []Structure `json:"structure"`
}

type Structure struct {
	Env            string                 `json:"env"`
	ResponseCode   int                    `json:"responseCode"`
	ApiParamMap    map[string]interface{} `json:"apiParamMap"`
	Variables      map[string]interface{} `json:"variables"`
	ResponseString map[string]interface{} `json:"responseString"`
}

func regex(body string) map[string]bool {
	apiList := make(map[string]bool)
	r := regexp.MustCompile(`(rpc )([a-zA-Z0-9]*)`)
	matches := r.FindAllStringSubmatch(body, -1)
	for _, v := range matches {
		apiList[v[2]] = true
	}
	return apiList
}

func regexGetBareEndpoint(body string) string {
	r := regexp.MustCompile(`({host}/function/mitraapp.Mitraapp.)([a-zA-Z0-9]*)(/invoke)`)
	matches := r.FindAllStringSubmatch(body, -1)
	for _, v := range matches {
		return v[2]
	}
	return ""
}

func extractValue(body string, key string) string {
	start := strings.Index(body, key)
	var end, openCurlyBracket, closeCurlyBracket int
	if start < 0 {
		return "{}"
	}
	for i := strings.Index(body, key); i < len(body); i++ {
		if string(body[i]) == "{" {
			openCurlyBracket++
		} else if string(body[i]) == "}" {
			closeCurlyBracket++
			if openCurlyBracket == closeCurlyBracket {
				end = i
				break
			}
		}
	}
	return body[start+12 : end+1]
}
