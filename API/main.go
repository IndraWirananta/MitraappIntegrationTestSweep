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

	integrationPath := "/Users/i.wirananta/go/src/github.com/tokopedia/te-api-automation-testdata/mitraapp/ApiIntegrationTest/TestCases" // change this
	mitraappPath := "/Users/i.wirananta/go/src/github.com/tokopedia/mitraapp/pkg/server/http.go"                                         // change this
	documentName := "./ITSWEEP.xlsx"                                                                                                     // change this

	xlsx := excelize.NewFile()
	sheet1Name := "Sheet1"
	xlsx.SetSheetName(xlsx.GetSheetName(1), sheet1Name)

	xlsx.SetCellValue(sheet1Name, "A1", "Endpoint")
	xlsx.SetCellValue(sheet1Name, "B1", "Type")
	xlsx.SetCellValue(sheet1Name, "C1", "Test Case Name")
	xlsx.SetCellValue(sheet1Name, "D1", "File Name")
	xlsx.SetCellValue(sheet1Name, "E1", "Scenario")
	xlsx.SetCellValue(sheet1Name, "F1", "Expected Response")
	xlsx.SetCellValue(sheet1Name, "G1", "Request Param")
	xlsx.SetCellValue(sheet1Name, "H1", "Status")
	xlsx.SetCellValue(sheet1Name, "I1", "Notes")
	xlsx.SetCellValue(sheet1Name, "J1", "PIC")

	mitraappHttp, _ := ioutil.ReadFile(mitraappPath)

	err := xlsx.AutoFilter(sheet1Name, "A1", "J1", "")
	if err != nil {
		log.Fatal("ERROR ", err.Error())
	}

	dvRange := excelize.NewDataValidation(true)
	dvRange.Sqref = "H:H"
	dvRange.SetDropList([]string{"Live", "On Progress", "Not Yet", "Pending", "No TestCase", "Not Checked", "Need Fix", "Wont Do", "Endpoint Need Adjustment"})
	xlsx.AddDataValidation(sheet1Name, dvRange)

	mapApiList := regex(string(mitraappHttp))

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

				apiName := strings.Replace(result.ApiName, "{host}", "", -1)
				httpMethod := strings.ToUpper(result.HttpMethod)
				if _, ok := mapApiList[apiName+httpMethod]; ok {
					mapApiList[apiName+httpMethod][1] = ""
				}

				xlsx.SetCellValue(sheet1Name, fmt.Sprintf("A%d", total+1), apiName)
				xlsx.SetCellValue(sheet1Name, fmt.Sprintf("B%d", total+1), strings.ToUpper(result.HttpMethod))
				xlsx.SetCellValue(sheet1Name, fmt.Sprintf("C%d", total+1), result.QueryName)
				xlsx.SetCellValue(sheet1Name, fmt.Sprintf("D%d", total+1), filepath.Base(path))
				xlsx.SetCellValue(sheet1Name, fmt.Sprintf("F%d", total+1), result.Structure[0].ResponseCode)
				xlsx.SetCellValue(sheet1Name, fmt.Sprintf("G%d", total+1), variables)
				xlsx.SetCellValue(sheet1Name, fmt.Sprintf("H%d", total+1), "Live")

				if prevValue == apiName {
					xlsx.MergeCell(sheet1Name, "A"+strconv.Itoa(total+1), "A"+strconv.Itoa(total))
					xlsx.MergeCell(sheet1Name, "B"+strconv.Itoa(total+1), "B"+strconv.Itoa(total))
				}

				prevValue = apiName

			}
			return nil
		})

	keys := make([]string, 0, len(mapApiList))

	for k := range mapApiList {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	fmt.Println("Got a total of " + strconv.Itoa(total) + " testcases")
	for _, k := range keys {
		if mapApiList[k][1] != "" {
			total++
			xlsx.SetCellValue(sheet1Name, fmt.Sprintf("A%d", total+1), mapApiList[k][0])
			xlsx.SetCellValue(sheet1Name, fmt.Sprintf("B%d", total+1), mapApiList[k][1])
			if strings.Contains(mapApiList[k][0], "intools") {
				xlsx.SetCellValue(sheet1Name, fmt.Sprintf("H%d", total+1), "Wont Do")
				xlsx.SetCellValue(sheet1Name, fmt.Sprintf("I%d", total+1), "Intools")
			} else {
				xlsx.SetCellValue(sheet1Name, fmt.Sprintf("H%d", total+1), "No TestCase")
			}
		}
	}

	fmt.Println("Scanned a total of " + strconv.Itoa(len(mapApiList)) + " endpoint")
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

func regex(body string) map[string][]string {
	apiList := make(map[string][]string)
	r := regexp.MustCompile(`r\.(Get|Delete|Patch|Post)\(\"(\/[a-z\/_]*)`)
	matches := r.FindAllStringSubmatch(body, -1)
	for _, v := range matches {
		temp := strings.ToUpper(v[1])
		apiList[v[2]+temp] = []string{v[2], temp}
	}
	return apiList
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
